package tests

import (
	"testing"

	"gocopilot/internal/tools"
)

func TestGenerateSchemaIncludesFields(t *testing.T) {
	schema := tools.GenerateSchema[tools.ReadFileInput]()

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map in schema, got %#v", schema["properties"])
	}

	if _, exists := properties["path"]; !exists {
		t.Fatalf("expected path property in schema")
	}
}
