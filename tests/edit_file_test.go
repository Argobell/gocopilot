package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocopilot/internal/tools"
)

func TestEditFileReplace(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "edit.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	payload, err := json.Marshal(tools.EditFileInput{
		Path:   filePath,
		OldStr: "world",
		NewStr: "gocopilot",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result, err := tools.EditFile(payload)
	if err != nil {
		t.Fatalf("EditFile returned error: %v", err)
	}

	if result != "OK" {
		t.Fatalf("expected result OK, got %q", result)
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	if string(updated) != "hello gocopilot" {
		t.Fatalf("unexpected file content %q", string(updated))
	}
}

func TestEditFileCreatesFileWhenMissing(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "new.txt")

	payload, err := json.Marshal(tools.EditFileInput{
		Path:   filePath,
		OldStr: "",
		NewStr: "generated content",
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result, err := tools.EditFile(payload)
	if err != nil {
		t.Fatalf("EditFile returned error: %v", err)
	}

	if !strings.Contains(result, "Successfully created file") {
		t.Fatalf("unexpected result: %q", result)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}

	if string(content) != "generated content" {
		t.Fatalf("unexpected file content %q", string(content))
	}
}
