package tests

import (
	"testing"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/tools"
)

func TestToolDefinitionToolConfig(t *testing.T) {
	td := tools.ToolDefinition{
		Name:        "read_file",
		Description: "Read a file",
		InputSchema: openai.FunctionParameters{"type": "object"},
	}

	config := td.ToolConfig()

	functionDef := config.GetFunction()
	if functionDef == nil {
		t.Fatalf("expected function definition, got nil")
	}

	if functionDef.Name != td.Name {
		t.Fatalf("expected function name %q, got %q", td.Name, functionDef.Name)
	}
}
