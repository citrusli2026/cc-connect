package dsh

import (
	"os/exec"
	"testing"

	"github.com/chenhg5/cc-connect/agent/acp"
)

func TestApplyDSHDefaults_FillsUnsetFields(t *testing.T) {
	got := applyDSHDefaults(map[string]any{})
	if got["command"] != "dsh" {
		t.Errorf("command = %v, want dsh", got["command"])
	}
	args, ok := got["args"].([]string)
	if !ok || len(args) != 2 || args[0] != "--profile" || args[1] != "acp" {
		t.Errorf("args = %v, want [--profile acp]", got["args"])
	}
	if got["display_name"] != "DeepSeek Harness" {
		t.Errorf("display_name = %v, want DeepSeek Harness", got["display_name"])
	}
}

func TestApplyDSHDefaults_UserOptsWin(t *testing.T) {
	got := applyDSHDefaults(map[string]any{
		"command":      "/opt/test-dsh",
		"args":         []string{"--custom", "value"},
		"display_name": "DSH staging",
		"work_dir":     "/tmp/project",
		"env":          map[string]string{"DSH_HOME": "/tmp/dsh"},
	})
	if got["command"] != "/opt/test-dsh" {
		t.Errorf("command was overwritten: %v", got["command"])
	}
	args := got["args"].([]string)
	if len(args) != 2 || args[0] != "--custom" || args[1] != "value" {
		t.Errorf("args were overwritten: %v", got["args"])
	}
	if got["display_name"] != "DSH staging" {
		t.Errorf("display_name was overwritten: %v", got["display_name"])
	}
	if got["work_dir"] != "/tmp/project" {
		t.Errorf("work_dir was lost: %v", got["work_dir"])
	}
	if env := got["env"].(map[string]string); env["DSH_HOME"] != "/tmp/dsh" {
		t.Errorf("env was lost: %v", got["env"])
	}
}

func TestApplyDSHDefaults_NilOptions(t *testing.T) {
	got := applyDSHDefaults(nil)
	if got == nil || got["command"] != "dsh" {
		t.Errorf("nil options should yield DSH defaults, got %v", got)
	}
}

func TestApplyDSHDefaults_BlankFieldsUseDefaults(t *testing.T) {
	got := applyDSHDefaults(map[string]any{
		"command":      "  ",
		"display_name": "\t",
	})
	if got["command"] != "dsh" {
		t.Errorf("command = %v, want dsh", got["command"])
	}
	if got["display_name"] != "DeepSeek Harness" {
		t.Errorf("display_name = %v, want DeepSeek Harness", got["display_name"])
	}
}

func TestNew_ReturnsDSHWrapper(t *testing.T) {
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("'true' not in PATH — unusual environment, skipping")
	}
	a, err := New(map[string]any{
		"command":  "true",
		"work_dir": "/tmp/project",
		"env":      map[string]string{"DSH_HOME": "/tmp/dsh"},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := a.Name(); got != "dsh" {
		t.Fatalf("Name() = %q, want dsh", got)
	}
	wrapper, ok := a.(*Agent)
	if !ok {
		t.Fatalf("New() returned %T, want *dsh.Agent", a)
	}
	var _ *acp.Agent = wrapper.Agent
	if got := wrapper.CLIDisplayName(); got != "DeepSeek Harness" {
		t.Fatalf("CLIDisplayName() = %q, want DeepSeek Harness", got)
	}
	if got := wrapper.GetWorkDir(); got != "/tmp/project" {
		t.Fatalf("GetWorkDir() = %q, want /tmp/project", got)
	}
	if got := wrapper.WorkspaceAgentOptions()["env"].(map[string]string)["DSH_HOME"]; got != "/tmp/dsh" {
		t.Fatalf("WorkspaceAgentOptions env DSH_HOME = %q, want /tmp/dsh", got)
	}
}

func TestNew_DisplayNameOverride(t *testing.T) {
	if _, err := exec.LookPath("true"); err != nil {
		t.Skip("'true' not in PATH — unusual environment, skipping")
	}
	a, err := New(map[string]any{
		"command":      "true",
		"display_name": "DSH production",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := a.(*Agent).CLIDisplayName(); got != "DSH production" {
		t.Fatalf("CLIDisplayName() = %q, want DSH production", got)
	}
}
