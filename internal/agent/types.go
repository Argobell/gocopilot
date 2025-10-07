package agent

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type InferenceClient interface {
	ChatCompletion(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)
}

type UserInputProvider interface {
	GetUserMessage() (string, bool)
}

type OutputHandler interface {
	PrintAssistantMessage(content string)
	PrintToolCall(toolName, arguments string)
	PrintToolResult(output string)
	PrintToolError(error string)
}

type DefaultOutputHandler struct{}

func (d *DefaultOutputHandler) PrintAssistantMessage(content string) {
	fmt.Printf("\u001b[1;33m🤖 Gocopilot\u001b[0m: %s\n", content)
}

func (d *DefaultOutputHandler) PrintToolCall(toolName, arguments string) {
	fmt.Printf("\u001b[36m🔧 Tool\u001b[0m: %s(%s)\n", toolName, arguments)
}

func (d *DefaultOutputHandler) PrintToolResult(output string) {
	fmt.Printf("\u001b[32m✅ Result\u001b[0m: %s\n", output)
}

func (d *DefaultOutputHandler) PrintToolError(error string) {
	fmt.Printf("\u001b[31m❌ Error\u001b[0m: %s\n", error)
}