package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/tools"
)

type Agent struct {
	client         *openai.Client
	getUserMessage func() (string, bool)
	tools          []tools.ToolDefinition
	toolParams     []openai.ChatCompletionToolUnionParam
	toolIndex      map[string]tools.ToolDefinition
	verbose        bool
}

func NewAgent(
	client *openai.Client,
	getUserMessage func() (string, bool),
	toolDefs []tools.ToolDefinition,
	verbose bool,
) *Agent {
	params := make([]openai.ChatCompletionToolUnionParam, 0, len(toolDefs))
	index := make(map[string]tools.ToolDefinition, len(toolDefs))
	for _, t := range toolDefs {
		params = append(params, t.ToolConfig())
		index[t.Name] = t
	}

	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          toolDefs,
		toolParams:     params,
		toolIndex:      index,
		verbose:        verbose,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []openai.ChatCompletionMessageParamUnion{}

	if a.verbose {
		log.Println("Starting chat session with tools enabled")
	}

	for {
		// The caller prints prompts and reads input via getUserMessage
		userInput, ok := a.getUserMessage()
		if !ok {
			if a.verbose {
				log.Println("User input ended, breaking from chat loop")
			}
			break
		}

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
		// Track assistant reply for context
		conversation = append(conversation, message.ToParam())

		for {
			if message.Content != "" {
				fmt.Printf("\u001b[93mGocopilot\u001b[0m: %s\n", message.Content)
			}

			var hasToolCalls bool

			if len(message.ToolCalls) > 0 {
				hasToolCalls = true
				if a.verbose {
					log.Printf("Processing %d tool calls", len(message.ToolCalls))
				}

				for _, toolCallUnion := range message.ToolCalls {
					call := toolCallUnion.AsAny()
					switch tc := call.(type) {
					case openai.ChatCompletionMessageFunctionToolCall:
						if a.verbose {
							log.Printf("Tool call detected: %s with input: %s", tc.Function.Name, tc.Function.Arguments)
						}
						fmt.Printf("\u001b[96mtool\u001b[0m: %s(%s)\n", tc.Function.Name, tc.Function.Arguments)

						toolDef, found := a.toolIndex[tc.Function.Name]
						var (
							toolResult string
							toolError  error
						)
						input := json.RawMessage([]byte(tc.Function.Arguments))
						if found {
							toolResult, toolError = toolDef.Function(input)
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
						} else {
							toolError = fmt.Errorf("tool '%s' not found", tc.Function.Name)
							toolResult = toolError.Error()
							fmt.Printf("\u001b[91merror\u001b[0m: %s\n", toolError.Error())
							fmt.Printf("\u001b[92mresult\u001b[0m: %s\n", toolResult)
						}

						if toolError != nil {
							conversation = append(conversation, openai.ToolMessage(fmt.Sprintf("Error: %s", toolError.Error()), tc.ID))
						} else {
							conversation = append(conversation, openai.ToolMessage(toolResult, tc.ID))
						}

					case openai.ChatCompletionMessageCustomToolCall:
						err := fmt.Errorf("unsupported custom tool call: %s", tc.Custom.Name)
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", err.Error())
						conversation = append(conversation, openai.ToolMessage(fmt.Sprintf("Error: %s", err.Error()), tc.ID))
					default:
						err := fmt.Errorf("unsupported tool call type")
						if a.verbose {
							log.Printf("Encountered unsupported tool call variant: %T", call)
						}
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", err.Error())
						conversation = append(conversation, openai.ToolMessage(fmt.Sprintf("Error: %s", err.Error()), toolCallUnion.ID))
					}
				}
			}

			if !hasToolCalls {
				break
			}

			if a.verbose {
				log.Printf("Sending tool results back to Gocopilot")
			}

			var err error
			response, err = a.runInference(ctx, conversation)
			if err != nil {
				if a.verbose {
					log.Printf("Error during followup inference: %v", err)
				}
				return err
			}

			message = response.Choices[0].Message
			conversation = append(conversation, message.ToParam())

			if a.verbose {
				log.Printf("Received followup response")
			}
		}
	}

	if a.verbose {
		log.Println("Chat session ended")
	}
	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []openai.ChatCompletionMessageParamUnion) (*openai.ChatCompletion, error) {
	model := os.Getenv("MODEL")
	params := openai.ChatCompletionNewParams{
		Model:     model,
		MaxTokens: openai.Int(1024),
		Messages:  conversation,
	}
	if len(a.toolParams) > 0 {
		params.Tools = a.toolParams
	}

	message, err := a.client.Chat.Completions.New(ctx, params)

	if a.verbose {
		if err != nil {
			log.Printf("API call failed: %v", err)
		} else {
			log.Printf("API call successful, response received")
		}
	}

	return message, err
}
