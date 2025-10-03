package tests

import (
	"testing"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/tools"
)

func TestToolDefinitionFunctionDefinition(t *testing.T) {
	td := tools.ToolDefinition{
		Name:        "sample_tool",
		Description: "sample description",
		InputSchema: openai.FunctionParameters{"type": "object"},
	}

	definition := td.FunctionDefinition()

	if definition.Name != td.Name {
		t.Fatalf("expected name %q, got %q", td.Name, definition.Name)
	}

	if !definition.Description.Valid() || definition.Description.Value != td.Description {
		t.Fatalf("expected description %q, got %#v", td.Description, definition.Description)
	}

	typeValue, ok := definition.Parameters["type"].(string)
	if !ok || typeValue != "object" {
		t.Fatalf("expected parameters type 'object', got %#v", definition.Parameters["type"])
	}
}
