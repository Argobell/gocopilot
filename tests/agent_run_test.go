package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"gocopilot/internal/agent"
	"gocopilot/internal/tools"
)

func TestAgentRunWithSingleMessage(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}

		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatalf("expected messages in payload, got %#v", payload["messages"])
		}

		toolsField, ok := payload["tools"].([]any)
		if !ok || len(toolsField) == 0 {
			t.Fatalf("expected tools to be forwarded, got %#v", payload["tools"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 0,
			"model":   "gpt-test",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":       "assistant",
						"content":    "hi there",
						"tool_calls": []any{},
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer server.Close()

	client := openai.NewClient(
		option.WithAPIKey("test"),
		option.WithBaseURL(server.URL),
		option.WithHTTPClient(server.Client()),
	)

	messages := []string{"hello"}
	var idx int
	getUserMessage := func() (string, bool) {
		if idx >= len(messages) {
			return "", false
		}
		msg := messages[idx]
		idx++
		return msg, true
	}

	a := agent.NewAgent(&client, getUserMessage, []tools.ToolDefinition{tools.ReadFileDefinition}, false)

	t.Setenv("MODEL", "gpt-test")

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	err = a.Run(context.Background())
	w.Close()
	os.Stdout = origStdout
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	outBytes, _ := io.ReadAll(r)
	output := string(outBytes)

	if !strings.Contains(output, "hi there") {
		t.Fatalf("expected assistant reply in output, got %q", output)
	}

	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatalf("expected chat completions endpoint to be called at least once")
	}
}
