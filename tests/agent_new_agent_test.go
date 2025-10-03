package tests

import (
	"reflect"
	"testing"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/agent"
	"gocopilot/internal/tools"
)

func TestNewAgentInitializesToolCaches(t *testing.T) {
	client := &openai.Client{}
	getUserMessage := func() (string, bool) { return "", false }
	toolDefs := []tools.ToolDefinition{tools.ReadFileDefinition, tools.ListFilesDefinition}

	a := agent.NewAgent(client, getUserMessage, toolDefs, true)
	if a == nil {
		t.Fatal("expected agent instance")
	}

	agentValue := reflect.ValueOf(a).Elem()
	toolParams := agentValue.FieldByName("toolParams")
	if !toolParams.IsValid() {
		t.Fatal("toolParams field missing")
	}

	if toolParams.Len() != len(toolDefs) {
		t.Fatalf("expected %d tool params, got %d", len(toolDefs), toolParams.Len())
	}

	toolIndex := agentValue.FieldByName("toolIndex")
	if !toolIndex.IsValid() {
		t.Fatal("toolIndex field missing")
	}

	if toolIndex.Len() != len(toolDefs) {
		t.Fatalf("expected %d tool index entries, got %d", len(toolDefs), toolIndex.Len())
	}

	if toolIndex.MapIndex(reflect.ValueOf(tools.ReadFileDefinition.Name)).IsZero() {
		t.Fatalf("expected read_file tool to be registered")
	}
}
