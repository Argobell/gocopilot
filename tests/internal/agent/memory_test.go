package agent_test

import (
	"reflect"
	"testing"

	"github.com/openai/openai-go/v3"

	"gocopilot/internal/agent"
)

func TestMemoryContextIncludesSystem(t *testing.T) {
	t.Parallel()

	mem := agent.NewMemory(5)
	system := openai.SystemMessage("system context")
	mem.SetSystemMessages(system)

	user := openai.UserMessage("hello")
	mem.Append(user)

	ctx := mem.Context()
	if len(ctx) != 2 {
		t.Fatalf("expected 2 messages in context, got %d", len(ctx))
	}

	if !reflect.DeepEqual(ctx[0], system) {
		t.Fatalf("expected system message first, got %#v", ctx[0])
	}

	if !reflect.DeepEqual(ctx[1], user) {
		t.Fatalf("expected user message second, got %#v", ctx[1])
	}
}

func TestMemoryTrimRespectsMaxHistory(t *testing.T) {
	t.Parallel()

	mem := agent.NewMemory(2)
	mem.Append(openai.UserMessage("one"))
	mem.Append(openai.UserMessage("two"))
	mem.Append(openai.UserMessage("three"))

	expected := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage("two"),
		openai.UserMessage("three"),
	}

	if ctx := mem.Context(); !reflect.DeepEqual(ctx, expected) {
		t.Fatalf("expected context %#v, got %#v", expected, ctx)
	}
}

func TestResetHistoryPreservesSystem(t *testing.T) {
	t.Parallel()

	mem := agent.NewMemory(3)
	system := openai.SystemMessage("persist")
	mem.SetSystemMessages(system)

	mem.Append(openai.UserMessage("temp"))
	mem.ResetHistory()

	expected := []openai.ChatCompletionMessageParamUnion{system}

	if ctx := mem.Context(); !reflect.DeepEqual(ctx, expected) {
		t.Fatalf("expected context %#v, got %#v", expected, ctx)
	}
}
