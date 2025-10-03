package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gocopilot/internal/tools"
)

func TestCodeSearch(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep is not available, skipping code search test")
	}

	dir := t.TempDir()
	filePath := filepath.Join(dir, "search.go")
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc target() {}\n"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	payload, err := json.Marshal(tools.CodeSearchInput{
		Pattern: "target",
		Path:    dir,
	})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result, err := tools.CodeSearch(payload)
	if err != nil {
		t.Fatalf("CodeSearch returned error: %v", err)
	}

	if !strings.Contains(result, "search.go") || !strings.Contains(result, "target") {
		t.Fatalf("unexpected search result: %q", result)
	}
}
