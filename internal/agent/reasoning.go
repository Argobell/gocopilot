package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
)

type ReasoningChain struct {
	steps    []ReasoningStep
	maxSteps int
	logger   Logger
}

type ReasoningStep struct {
	Type    StepType
	Content string
	ToolCalls []openai.ChatCompletionMessageToolCallUnion
	Result   string
}

type StepType string

const (
	StepTypeThought    StepType = "thought"
	StepTypeAction     StepType = "action"
	StepTypeObservation StepType = "observation"
	StepTypeFinal      StepType = "final"
)

func NewReasoningChain(maxSteps int, logger Logger) *ReasoningChain {
	if maxSteps <= 0 {
		maxSteps = 10
	}

	return &ReasoningChain{
		maxSteps: maxSteps,
		logger:   logger,
	}
}

func (rc *ReasoningChain) Execute(
	ctx context.Context,
	agent *Agent,
	userInput string,
) (string, error) {
	rc.logger.Info("Starting reasoning chain for user input: %q", userInput)

	// Reset memory for new reasoning chain
	agent.memory.ResetHistory()
	agent.memory.Append(openai.UserMessage(userInput))

	for step := 0; step < rc.maxSteps; step++ {
		rc.logger.Debug("Reasoning step %d", step+1)

		response, err := agent.runInference(ctx, agent.memory.Context())
		if err != nil {
			return "", fmt.Errorf("reasoning step %d failed: %w", step+1, err)
		}

		message := response.Choices[0].Message
		stepType := rc.analyzeStepType(message)

		currentStep := ReasoningStep{
			Type:    stepType,
			Content: message.Content,
			ToolCalls: message.ToolCalls,
		}

		rc.steps = append(rc.steps, currentStep)
		agent.memory.Append(message.ToParam())

		// Handle assistant message
		if message.Content != "" {
			agent.output.PrintAssistantMessage(message.Content)
		}

		// Handle tool calls
		if len(message.ToolCalls) > 0 {
			toolMessages, err := agent.executor.ExecuteToolCalls(ctx, message.ToolCalls)
			if err != nil {
				return "", fmt.Errorf("tool execution failed at step %d: %w", step+1, err)
			}

			// Add tool results to memory
			for _, toolMsg := range toolMessages {
				agent.memory.Append(toolMsg)
			}

			// Continue to next reasoning step
			continue
		}

		// Check if this is a final answer
		if rc.isFinalAnswer(message.Content) {
			rc.logger.Info("Reasoning chain completed with final answer after %d steps", step+1)
			return message.Content, nil
		}

		// Check for reasoning completion without explicit final marker
		if stepType == StepTypeFinal || rc.isCompleteAnswer(message.Content) {
			rc.logger.Info("Reasoning chain completed after %d steps", step+1)
			return message.Content, nil
		}
	}

	rc.logger.Warn("Reasoning chain reached maximum steps (%d) without completion", rc.maxSteps)
	return "", fmt.Errorf("reasoning chain exceeded maximum steps (%d)", rc.maxSteps)
}

func (rc *ReasoningChain) analyzeStepType(message openai.ChatCompletionMessage) StepType {
	content := strings.ToLower(message.Content)

	// Check for final answer indicators
	if strings.Contains(content, "final answer") ||
		strings.Contains(content, "answer:") ||
		strings.Contains(content, "conclusion:") ||
		(len(message.ToolCalls) == 0 && !strings.Contains(content, "let me")) {
		return StepTypeFinal
	}

	// Check for thought indicators
	if strings.Contains(content, "thinking") ||
		strings.Contains(content, "thought:") ||
		strings.Contains(content, "reason:") {
		return StepTypeThought
	}

	// Check for action indicators
	if len(message.ToolCalls) > 0 {
		return StepTypeAction
	}

	// Default to observation
	return StepTypeObservation
}

func (rc *ReasoningChain) isFinalAnswer(content string) bool {
	lowerContent := strings.ToLower(content)
	return strings.Contains(lowerContent, "final answer") ||
		strings.Contains(lowerContent, "answer:") ||
		strings.Contains(lowerContent, "conclusion:")
}

func (rc *ReasoningChain) isCompleteAnswer(content string) bool {
	// Simple heuristic: if the content doesn't suggest more actions and is reasonably long
	lowerContent := strings.ToLower(content)

	actionWords := []string{
		"let me", "i'll", "i will", "next", "now", "then",
		"search", "read", "execute", "run", "check", "verify",
	}

	for _, word := range actionWords {
		if strings.Contains(lowerContent, word) {
			return false
		}
	}

	// Consider it complete if it's more than a short response
	return len(strings.TrimSpace(content)) > 50
}

func (rc *ReasoningChain) GetSteps() []ReasoningStep {
	return rc.steps
}

func (rc *ReasoningChain) Reset() {
	rc.steps = nil
}