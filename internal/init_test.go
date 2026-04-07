package fzfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
)

func TestRunStep_Success(t *testing.T) {
	// Ensure Log is initialized for the test
	Log = log.New(os.Stderr)

	step := runStep("test-step", func(l *log.Logger) (map[string]string, error) {
		return map[string]string{"key": "value"}, nil
	})

	if step.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", step.Status)
	}
	if step.Name != "test-step" {
		t.Errorf("expected name 'test-step', got %q", step.Name)
	}
	if step.Duration <= 0 {
		t.Error("expected positive duration")
	}
	if step.Snapshot["key"] != "value" {
		t.Errorf("expected snapshot key=value, got %v", step.Snapshot)
	}
}

func TestRunStep_Error(t *testing.T) {
	Log = log.New(os.Stderr)

	step := runStep("fail-step", func(l *log.Logger) (map[string]string, error) {
		return nil, os.ErrNotExist
	})

	if step.Status != "error" {
		t.Errorf("expected status 'error', got %q", step.Status)
	}
	if step.Message == "" {
		t.Error("expected non-empty error message")
	}
}

func TestRunInit_WithConfig(t *testing.T) {
	// Set up a temp config directory
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "modules")
	os.MkdirAll(modDir, 0755)

	mainConfig := `logger:
  default:
    level: info
    console: true
    format: text
options:
  default:
    ansi: true
    reverse: true
commands:
  test_cmd: echo hello
profiles:
  test_profile:
    command: echo test
    options:
      ansi: true
`
	mainPath := filepath.Join(tmpDir, "fzfg.yaml")
	os.WriteFile(mainPath, []byte(mainConfig), 0644)

	// Point to this config
	t.Setenv("FZFG_CONF", mainPath)

	// Reset flags
	InitFlag = true
	CommandFlag = "test_cmd"
	OptionsFlag = ""
	ProfileFlag = ""

	Log = log.New(os.Stderr)

	result, err := RunInit()
	if err != nil {
		t.Fatalf("RunInit error: %v", err)
	}

	// Verify all steps ran
	expectedSteps := []string{"start", "config", "validate", "rsc-load", "env-load", "env-set"}
	if len(result.Steps) != len(expectedSteps) {
		t.Fatalf("expected %d steps, got %d", len(expectedSteps), len(result.Steps))
	}
	for i, expected := range expectedSteps {
		if result.Steps[i].Name != expected {
			t.Errorf("step %d: expected %q, got %q", i, expected, result.Steps[i].Name)
		}
	}

	// Start and config should be ok
	if result.Steps[0].Status != "ok" {
		t.Errorf("start step should be ok, got %q: %s", result.Steps[0].Status, result.Steps[0].Message)
	}
	if result.Steps[1].Status != "ok" {
		t.Errorf("config step should be ok, got %q: %s", result.Steps[1].Status, result.Steps[1].Message)
	}

	// Total duration should be positive
	if result.Total <= 0 {
		t.Error("expected positive total duration")
	}

	// Config should have our data
	if len(result.Config.Commands) == 0 {
		t.Error("expected commands in result config")
	}

	// FinalCmd should be set since we specified CommandFlag
	if result.FinalCmd == "" {
		t.Error("expected FinalCmd to be set")
	}
}

func TestRunInit_NoConfig(t *testing.T) {
	// Point to nonexistent config
	t.Setenv("FZFG_CONF", "")
	t.Setenv("HOME", t.TempDir())

	origDir, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(origDir)

	Log = log.New(os.Stderr)
	InitFlag = true
	CommandFlag = ""
	OptionsFlag = ""
	ProfileFlag = ""

	_, err := RunInit()
	if err == nil {
		t.Error("expected error when no config file exists")
	}
}

func TestInitStep_Snapshot(t *testing.T) {
	step := InitStep{
		Name:     "test",
		Status:   "ok",
		Snapshot: map[string]string{"a": "1", "b": "2"},
	}

	if step.Snapshot["a"] != "1" {
		t.Error("snapshot should contain key 'a'")
	}
	if step.Snapshot["b"] != "2" {
		t.Error("snapshot should contain key 'b'")
	}
}
