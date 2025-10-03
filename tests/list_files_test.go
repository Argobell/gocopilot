package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gocopilot/internal/tools"
)

func TestListFiles(t *testing.T) {
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("failed to create root file: %v", err)
	}

	nestedDir := filepath.Join(root, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nestedDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	payload, err := json.Marshal(tools.ListFilesInput{Path: root})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result, err := tools.ListFiles(payload)
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}

	var files []string
	if err := json.Unmarshal([]byte(result), &files); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	sort.Strings(files)
	expected := []string{"nested/", "nested/nested.txt", "root.txt"}

	if len(files) != len(expected) {
		t.Fatalf("expected %d items, got %d (%v)", len(expected), len(files), files)
	}

	for i, want := range expected {
		if files[i] != want {
			t.Fatalf("expected entry %d to be %q, got %q", i, want, files[i])
		}
	}
}
