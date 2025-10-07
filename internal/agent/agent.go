package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/config"
	"gocopilot/internal/tools"
)

type Agent struct {
	client      InferenceClient
	input       UserInputProvider
	output      OutputHandler
	memory      *Memory
	executor    *ToolExecutor
	logger      Logger
	config      *config.Config
	toolConfigs []openai.ChatCompletionToolUnionParam
}

func NewAgent(
	client InferenceClient,
	input UserInputProvider,
	output OutputHandler,
	registry *tools.Registry,
	cfg *config.Config,
	logger Logger,
) *Agent {
	if logger == nil {
		logger = &NoopLogger{}
	}

	if output == nil {
		output = &DefaultOutputHandler{}
	}

	memory := NewMemory(cfg.MemoryCapacity)
	executor := NewToolExecutor(registry, cfg.MaxConcurrency, logger)
	toolConfigs := registry.ToolConfigs()

	return &Agent{
		client:      client,
		input:       input,
		output:      output,
		memory:      memory,
		executor:    executor,
		logger:      logger,
		config:      cfg,
		toolConfigs: toolConfigs,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("Starting chat session")
	a.memory.ResetHistory()

	// Set system message if provided
	if systemMsg := os.Getenv("SYSTEM_MESSAGE"); systemMsg != "" {
		a.memory.SetSystemMessages(openai.SystemMessage(systemMsg))
	}

	for {
		userInput, ok := a.input.GetUserMessage()
		if !ok {
			a.logger.Info("User input ended, breaking from chat loop")
			break
		}

		if userInput == "" {
			a.logger.Debug("Skipping empty message")
			continue
		}

		a.logger.Debug("User input received: %q", userInput)

		userMessage := openai.UserMessage(userInput)
		a.memory.Append(userMessage)

		a.logger.Debug("Sending message to Gocopilot, conversation length: %d", a.memory.MessageCount())

		if err := a.processConversation(ctx); err != nil {
			a.logger.Error("Error during conversation processing: %v", err)
			return err
		}

		fmt.Println() // Add empty line between interactions
	}

	a.logger.Info("Chat session ended")
	return nil
}

func (a *Agent) processConversation(ctx context.Context) error {
	for {
		response, err := a.runInference(ctx, a.memory.Context())
		if err != nil {
			return err
		}

		message := response.Choices[0].Message
		a.memory.Append(message.ToParam())

		// Handle assistant message
		if message.Content != "" {
			a.output.PrintAssistantMessage(message.Content)
		}

		// Handle tool calls
		if len(message.ToolCalls) > 0 {
			a.logger.Debug("Processing %d tool calls", len(message.ToolCalls))

			// Print tool calls
			for _, toolCallUnion := range message.ToolCalls {
				call := toolCallUnion.AsAny()
				if tc, ok := call.(openai.ChatCompletionMessageFunctionToolCall); ok {
					a.output.PrintToolCall(tc.Function.Name, tc.Function.Arguments)
				}
			}

			// Execute tool calls
			toolMessages, err := a.executor.ExecuteToolCalls(ctx, message.ToolCalls)
			if err != nil {
				return err
			}

			// Print tool results and add to memory
			for _, toolMsg := range toolMessages {
				// For tool messages, we'll just add them to memory without printing
				// The actual tool results are already printed by the executor
				a.memory.Append(toolMsg)
			}

			// Continue processing with tool results
			continue
		}

		// No tool calls, conversation complete
		break
	}

	return nil
}


func (a *Agent) runInference(ctx context.Context, conversation []openai.ChatCompletionMessageParamUnion) (*openai.ChatCompletion, error) {
	params := openai.ChatCompletionNewParams{
		Model:     a.config.Model,
		MaxTokens: openai.Int(int64(a.config.MaxTokens)),
		Messages:  conversation,
	}

	if len(a.toolConfigs) > 0 {
		params.Tools = a.toolConfigs
	}

	response, err := a.client.ChatCompletion(ctx, params)

	if err != nil {
		a.logger.Error("API call failed: %v", err)
	} else {
		a.logger.Debug("API call successful, response received")
	}

	return response, err
}


type NoopLogger struct{}

func (n NoopLogger) Debug(format string, args ...interface{}) {}
func (n NoopLogger) Info(format string, args ...interface{})  {}
func (n NoopLogger) Warn(format string, args ...interface{})  {}
func (n NoopLogger) Error(format string, args ...interface{}) {}