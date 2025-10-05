package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

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
	memory         *Memory
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
		memory:         NewMemory(DefaultMemoryCapacity),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	if a.verbose {
		log.Println("Starting chat session")
	}

	a.memory.ResetHistory()

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
		a.memory.Append(userMessage)

		if a.verbose {
			log.Printf("Sending message to Gocopilot, conversation length: %d", a.memory.MessageCount())
		}

		response, err := a.runInference(ctx, a.memory.Context())
		if err != nil {
			if a.verbose {
				log.Printf("Error during inference: %v", err)
			}
			return err
		}

		message := response.Choices[0].Message
		// Track assistant reply for context
		a.memory.Append(message.ToParam())

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

				results := make([]struct {
					output  string
					err     error
					id      string
					handled bool
				}, len(message.ToolCalls))
				var wg sync.WaitGroup

				for idx, toolCallUnion := range message.ToolCalls {
					call := toolCallUnion.AsAny()
					switch tc := call.(type) {
					case openai.ChatCompletionMessageFunctionToolCall:
						if a.verbose {
							log.Printf("Tool call detected: %s with input: %s", tc.Function.Name, tc.Function.Arguments)
						}
						fmt.Printf("\u001b[96mtool\u001b[0m: %s(%s)\n", tc.Function.Name, tc.Function.Arguments)

						toolDef, found := a.toolIndex[tc.Function.Name]
						if !found {
							err := fmt.Errorf("tool '%s' not found", tc.Function.Name)
							result := err.Error()
							fmt.Printf("\u001b[91merror\u001b[0m: %s\n", err.Error())
							fmt.Printf("\u001b[92mresult\u001b[0m: %s\n", result)
							a.memory.Append(openai.ToolMessage(fmt.Sprintf("Error: %s", err.Error()), tc.ID))
							continue
						}

						input := json.RawMessage([]byte(tc.Function.Arguments))

						wg.Add(1)
						go func(index int, callID string, def tools.ToolDefinition, payload json.RawMessage) {
							defer wg.Done()

							toolResult, toolErr := def.Function(payload)
							results[index] = struct {
								output  string
								err     error
								id      string
								handled bool
							}{
								output:  toolResult,
								err:     toolErr,
								id:      callID,
								handled: true,
							}

							if a.verbose {
								if toolErr != nil {
									log.Printf("Tool execution failed: %v", toolErr)
								} else {
									log.Printf("Tool execution successful, result length: %d chars", len(toolResult))
								}
							}
						}(idx, tc.ID, toolDef, input)

					case openai.ChatCompletionMessageCustomToolCall:
						err := fmt.Errorf("unsupported custom tool call: %s", tc.Custom.Name)
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", err.Error())
						a.memory.Append(openai.ToolMessage(fmt.Sprintf("Error: %s", err.Error()), tc.ID))
					default:
						err := fmt.Errorf("unsupported tool call type")
						if a.verbose {
							log.Printf("Encountered unsupported tool call variant: %T", call)
						}
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", err.Error())
						a.memory.Append(openai.ToolMessage(fmt.Sprintf("Error: %s", err.Error()), toolCallUnion.ID))
					}
				}

				wg.Wait()

				for idx, toolCallUnion := range message.ToolCalls {
					call := toolCallUnion.AsAny()
					tc, ok := call.(openai.ChatCompletionMessageFunctionToolCall)
					if !ok {
						continue
					}

					res := results[idx]
					if !res.handled {
						continue
					}

					fmt.Printf("\u001b[92mresult\u001b[0m: %s\n", res.output)
					if res.err != nil {
						fmt.Printf("\u001b[91merror\u001b[0m: %s\n", res.err.Error())
						a.memory.Append(openai.ToolMessage(fmt.Sprintf("Error: %s", res.err.Error()), tc.ID))
						continue
					}

					a.memory.Append(openai.ToolMessage(res.output, tc.ID))
				}

			}

			if !hasToolCalls {
				break
			}

			if a.verbose {
				log.Printf("Sending tool results back to Gocopilot")
			}

			var err error
			response, err = a.runInference(ctx, a.memory.Context())
			if err != nil {
				if a.verbose {
					log.Printf("Error during followup inference: %v", err)
				}
				return err
			}

			message = response.Choices[0].Message
			a.memory.Append(message.ToParam())

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
