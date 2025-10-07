package tools

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/openai/openai-go/v3"
)

type Registry struct {
	tools     map[string]ToolDefinition
	mu        sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolDefinition),
	}
}

func (r *Registry) Register(tool ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

func (r *Registry) Get(name string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

func (r *Registry) List() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

func (r *Registry) ToolConfigs() []openai.ChatCompletionToolUnionParam {
	tools := r.List()
	configs := make([]openai.ChatCompletionToolUnionParam, len(tools))

	for i, tool := range tools {
		configs[i] = tool.ToolConfig()
	}

	return configs
}

type ToolResult struct {
	Output string
	Error  error
	CallID string
}

func (r *Registry) ExecuteTool(name string, arguments json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	tool, exists := r.Get(name)
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", name)
	}

	return tool.Function(arguments, log)
}