package fzfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
)

func TestRunStep_Success(t *testing.T) {
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

func TestStepsUpTo(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"start", 1},
		{"config", 2},
		{"validate", 3},
		{"rsc-load", 4},
		{"env-load", 5},
		{"env-set", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepsUpTo(tt.name)
			if len(got) != tt.want {
				t.Errorf("stepsUpTo(%q) = %d steps, want %d", tt.name, len(got), tt.want)
			}
		})
	}
}

func TestStepsUpTo_Unknown(t *testing.T) {
	got := stepsUpTo("nonexistent")
	if got != nil {
		t.Errorf("expected nil for unknown step, got %v", got)
	}
}

func TestRunInit_WithConfig(t *testing.T) {
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

	t.Setenv("FZFG_CONF", mainPath)

	CommandFlag = "test_cmd"
	OptionsFlag = ""
	ProfileFlag = ""
	ModuleFlag = "files"
	Log = log.New(os.Stderr)

	result, err := RunInit("")
	if err != nil {
		t.Fatalf("RunInit error: %v", err)
	}

	expectedSteps := AllInitSteps
	if len(result.Steps) != len(expectedSteps) {
		t.Fatalf("expected %d steps, got %d", len(expectedSteps), len(result.Steps))
	}
	for i, expected := range expectedSteps {
		if result.Steps[i].Name != expected {
			t.Errorf("step %d: expected %q, got %q", i, expected, result.Steps[i].Name)
		}
	}

	if result.Steps[0].Status != "ok" {
		t.Errorf("start should be ok, got %q: %s", result.Steps[0].Status, result.Steps[0].Message)
	}

	if result.Total <= 0 {
		t.Error("expected positive total duration")
	}

	if result.FinalCmd == "" {
		t.Error("expected FinalCmd to be set")
	}
}

func TestRunInit_SingleStep(t *testing.T) {
	tmpDir := t.TempDir()
	mainConfig := `options: {}`
	mainPath := filepath.Join(tmpDir, "fzfg.yaml")
	os.WriteFile(mainPath, []byte(mainConfig), 0644)

	t.Setenv("FZFG_CONF", mainPath)
	Log = log.New(os.Stderr)
	ModuleFlag = "files"
	CommandFlag = ""

	result, err := RunInit("start")
	if err != nil {
		t.Fatalf("RunInit(start) error: %v", err)
	}

	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step for 'start', got %d", len(result.Steps))
	}
	if result.Steps[0].Name != "start" {
		t.Errorf("expected step 'start', got %q", result.Steps[0].Name)
	}
}

func TestRunInit_UnknownStep(t *testing.T) {
	Log = log.New(os.Stderr)
	ModuleFlag = "files"

	_, err := RunInit("bogus")
	if err == nil {
		t.Error("expected error for unknown step")
	}
}

func TestRunInit_NoConfig(t *testing.T) {
	t.Setenv("FZFG_CONF", "")
	t.Setenv("HOME", t.TempDir())

	origDir, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(origDir)

	Log = log.New(os.Stderr)
	ModuleFlag = "files"
	CommandFlag = ""

	_, err := RunInit("")
	if err == nil {
		t.Error("expected error when no config file exists")
	}
}
