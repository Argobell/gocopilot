package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	if *verbose {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	} else {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
		log.SetPrefix("")
	}

	// load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseUrl := os.Getenv("OPENAI_API_BASE_URL")

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
	)
	if *verbose {
		log.Println("OpenAI client initialized")
	}

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	tools := []ToolDefinition{ReadFileDefinition, ListFilesDefinition, BashDefinition}
	if *verbose {
		log.Printf("Initialized %d tools", len(tools))
	}
	agent := NewAgent(&client, getUserMessage, tools, *verbose)
	err = agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func NewAgent(
	client *openai.Client,
	getUserMessage func() (string, bool),
	tools []ToolDefinition,
	verbose bool,
) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
		verbose:        verbose,
	}
}

type Agent struct {
	client         *openai.Client
	getUserMessage func() (string, bool)
	tools          []ToolDefinition
	verbose        bool
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []openai.ChatCompletionMessageParamUnion{}

	if a.verbose {
		log.Println("Starting chat session with tools enabled")
	}
	fmt.Println("Chat with Gocopilot (use 'ctrl-c' to quit)")

	for {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			if a.verbose {
				log.Println("User input ended, breaking from chat loop")
			}
			break
		}

		// Skip empty messages
		if userInput == "" {
			if a.verbose {
				log.Println("Skipping empty message")
			}
			continue
		}

		if a.verbose {
			log.Printf("User input received: %q", userInput)
		}

		userMessage := openai.UserMessage(userInput)
		conversation = append(conversation, userMessage)

		if a.verbose {
			log.Printf("Sending message to Gocopilot, conversation length: %d", len(conversation))
		}

		response, err := a.runInference(ctx, conversation)
		if err != nil {
			if a.verbose {
				log.Printf("Error during inference: %v", err)
			}
			return err
		}

		message := response.Choices[0].Message
		// Add the complete assistant message (including tool_calls if present)
		conversation = append(conversation, message.ToParam())

		// Keep processing until GPT stops using tools
		for {
			var hasToolCalls bool

			if a.verbose {
				log.Printf("Processing %d content blocks from Gocopilot", len(message.Content))
			}

			// Check for text content
			if message.Content != "" {
				fmt.Printf("\u001b[93mGocopilot\u001b[0m: %s\n", message.Content)
			}

			// Check for tool calls
			if len(message.ToolCalls) > 0 {
				hasToolCalls = true

				if a.verbose {
					log.Printf("Processing %d tool calls", len(message.ToolCalls))
				}

				for _, toolCall := range message.ToolCalls {
					if a.verbose {
						log.Printf("Tool call detected: %s with input: %s", toolCall.Function.Name, toolCall.Function.Arguments)
					}
					fmt.Printf("\u001b[96mtool\u001b[0m: %s(%s)\n", toolCall.Function.Name, toolCall.Function.Arguments)

					// Find and execute the tool
					var toolResult string
					var toolError error
					var toolFound bool

					for _, tool := range a.tools {
						if tool.Name == toolCall.Function.Name {
							if a.verbose {
								log.Printf("Executing tool: %s", tool.Name)
							}
							toolResult, toolError = tool.Function(json.RawMessage(toolCall.Function.Arguments))
							fmt.Printf("\u001b[92mresult\u001b[0m: %s\n", toolResult)
							if toolError != nil {
								fmt.Printf("\u001b[91merror\u001b[0m: %s\n", toolError.Error())
							}
							if a.verbose {
								if toolError != nil {
									log.Printf("Tool execution failed: %v", toolError)
								} else {
									log.Printf("Tool execution successful, result length: %d chars", len(toolResult))
								}
							}
							toolFound = true
							break
						}
					}

					if !toolFound {
						toolError = fmt.Errorf("tool '%s' not found", toolCall.Function.Name)
						toolResult = toolError.Error()
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", toolError.Error())
					}

					// Add tool result to conversation
					if toolError != nil {
						conversation = append(conversation, openai.ToolMessage(toolCall.ID, fmt.Sprintf("Error: %s", toolError.Error())))
					} else {
						conversation = append(conversation, openai.ToolMessage(toolResult, toolCall.ID))
					}
				}
			}

			// If there were no tool calls, we're done
			if !hasToolCalls {
				break
			}

			// Get GPT's response after tool execution
			if a.verbose {
				log.Printf("Sending tool results back to Gocopilot")
			}

			response, err = a.runInference(ctx, conversation)
			if err != nil {
				if a.verbose {
					log.Printf("Error during followup inference: %v", err)
				}
				return err
			}

			message = response.Choices[0].Message
			// Add the complete assistant message
			conversation = append(conversation, message.ToParam())

			if a.verbose {
				log.Printf("Received followup response")
			}

			// Continue loop to process the new message
		}
	}

	if a.verbose {
		log.Println("Chat session ended")
	}
	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []openai.ChatCompletionMessageParamUnion) (*openai.ChatCompletion, error) {
	openaiTools := []openai.ChatCompletionToolParam{}
	for _, tool := range a.tools {
		openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  openai.FunctionParameters(tool.InputSchema),
			},
		})
	}

	model := os.Getenv("MODEL")
	if a.verbose {
		log.Printf("Making API call with model: %s and %d tools", model, len(openaiTools))
	}

	message, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     model,
		MaxTokens: openai.Int(1024),
		Messages:  conversation,
		Tools:     openaiTools,
	})

	if a.verbose {
		if err != nil {
			log.Printf("API call failed: %v", err)
		} else {
			log.Printf("API call successful, response received")
		}
	}

	return message, err
}

type ToolDefinition struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	InputSchema openai.FunctionParameters `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

// 定义一个新的工具 ListFiles
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

// 定义一个新的工具 Bash
var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: "Execute a bash command and return its output. Use this to run shell commands.",
	InputSchema: BashInputSchema,
	Function:    Bash,
}

// 定义 ReadFileInput 结构体
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

// 定义 ListFilesInput 结构体
type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

// 定义 BashInput 结构体
type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute."`
}

// 生成 ReadFileInput 的 JSON Schema
var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

// 生成 ListFilesInput 的 JSON Schema
var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

// 生成 BashInput 的 JSON Schema
var BashInputSchema = GenerateSchema[BashInput]()

func ReadFile(input json.RawMessage) (string, error) {
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		panic(err)
	}

	log.Printf("Reading file: %s", readFileInput.Path)
	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		log.Printf("Failed to read file %s: %v", readFileInput.Path, err)
		return "", err
	}
	log.Printf("Successfully read file %s (%d bytes)", readFileInput.Path, len(content))
	return string(content), nil
}

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		panic(err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	log.Printf("Listing files in directory: %s", dir)

	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			relPath = filepath.ToSlash(relPath)
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}

		return nil
	})
	if err != nil {
		log.Printf("Failed to list files in %s: %v", dir, err)
		return "", err
	}

	log.Printf("Successfully listed %d items in %s", len(files), dir)

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// 实现 Bash 工具函数
func Bash(input json.RawMessage) (string, error) {
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	log.Printf("Executing bash command: %s", bashInput.Command)
	cmd := exec.Command("nu", "-c", bashInput.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Bash command failed: %s, error: %v", bashInput.Command, err)
		return fmt.Sprintf("Command failed with error: %s\nOutput: %s", err.Error(), string(output)), nil
	}

	log.Printf("Bash command succeeded: %s (output: %d bytes)", bashInput.Command, len(output))
	return strings.TrimSpace(string(output)), nil
}

func GenerateSchema[T any]() openai.FunctionParameters {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)

	param := openai.FunctionParameters{}
	b, _ := json.Marshal(schema)
	_ = json.Unmarshal(b, &param)

	return param
}
