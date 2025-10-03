package tests

import (
	"encoding/json"
	"os/exec"
	"testing"

	"gocopilot/internal/tools"
)

func TestBash(t *testing.T) {
	if _, err := exec.LookPath("nu"); err != nil {
		t.Skip("nushell is not available, skipping bash tool test")
	}

	payload, err := json.Marshal(tools.BashInput{Command: "echo test"})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	output, err := tools.Bash(payload)
	if err != nil {
		t.Fatalf("Bash returned error: %v", err)
	}

	if output != "test" {
		t.Fatalf("expected output \"test\", got %q", output)
	}
}
