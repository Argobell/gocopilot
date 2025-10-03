package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gocopilot/internal/tools"
)

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sample.txt")
	want := "hello world"
	if err := os.WriteFile(filePath, []byte(want), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	payload, err := json.Marshal(tools.ReadFileInput{Path: filePath})
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	got, err := tools.ReadFile(payload)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
