package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/tools"
)

type ToolExecutor struct {
	registry   *tools.Registry
	maxWorkers int
	logger     Logger
}

func NewToolExecutor(registry *tools.Registry, maxWorkers int, logger Logger) *ToolExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 5
	}

	return &ToolExecutor{
		registry:   registry,
		maxWorkers: maxWorkers,
		logger:     logger,
	}
}

func (e *ToolExecutor) ExecuteToolCalls(
	ctx context.Context,
	toolCalls []openai.ChatCompletionMessageToolCallUnion,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	results := make([]tools.ToolResult, len(toolCalls))
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrent tool executions
	semaphore := make(chan struct{}, e.maxWorkers)

	for idx, toolCallUnion := range toolCalls {
		call := toolCallUnion.AsAny()
		switch tc := call.(type) {
		case openai.ChatCompletionMessageFunctionToolCall:
			e.logger.Debug("Executing tool: %s with args: %s", tc.Function.Name, tc.Function.Arguments)

			wg.Add(1)
			go func(index int, callID, toolName string, arguments json.RawMessage) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Check if context is cancelled
				select {
				case <-ctx.Done():
					results[index] = tools.ToolResult{
						Output: fmt.Sprintf("tool execution cancelled: %v", ctx.Err()),
						Error:  ctx.Err(),
						CallID: callID,
					}
					return
				default:
				}

				output, err := e.registry.ExecuteTool(toolName, arguments, e.logger)
				results[index] = tools.ToolResult{
					Output: output,
					Error:  err,
					CallID: callID,
				}

				if err != nil {
					e.logger.Warn("Tool execution failed: %s, error: %v", toolName, err)
				} else {
					e.logger.Debug("Tool execution successful: %s, output length: %d", toolName, len(output))
				}
			}(idx, tc.ID, tc.Function.Name, json.RawMessage(tc.Function.Arguments))

		case openai.ChatCompletionMessageCustomToolCall:
			results[idx] = tools.ToolResult{
				Output: fmt.Sprintf("unsupported custom tool call: %s", tc.Custom.Name),
				Error:  fmt.Errorf("unsupported custom tool call: %s", tc.Custom.Name),
				CallID: tc.ID,
			}
			e.logger.Warn("Unsupported custom tool call: %s", tc.Custom.Name)

		default:
			results[idx] = tools.ToolResult{
				Output: "unsupported tool call type",
				Error:  fmt.Errorf("unsupported tool call type: %T", call),
				CallID: toolCallUnion.ID,
			}
			e.logger.Warn("Unsupported tool call type: %T", call)
		}
	}

	wg.Wait()

	// Convert results to tool messages
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(results))
	for _, result := range results {
		if result.CallID == "" {
			continue
		}

		var content string
		if result.Error != nil {
			content = fmt.Sprintf("Error: %s", result.Error.Error())
		} else {
			content = result.Output
		}

		messages = append(messages, openai.ToolMessage(content, result.CallID))
	}

	return messages, nil
}