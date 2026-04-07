package repl

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want 'info'", cfg.LogLevel)
	}
	if cfg.MaxItems != 100 {
		t.Errorf("MaxItems = %d, want 100", cfg.MaxItems)
	}
	if cfg.JSONIndent != 2 {
		t.Errorf("JSONIndent = %d, want 2", cfg.JSONIndent)
	}
	if !cfg.ColorOutput {
		t.Error("ColorOutput should default to true")
	}
}

func TestColorizeJSON(t *testing.T) {
	input := `{"key": "value", "num": 42, "bool": true}`
	result := colorizeJSON(input)

	// Should produce non-empty output
	if len(result) == 0 {
		t.Error("colorizeJSON returned empty string")
	}
	// Should contain all the original content (possibly with ANSI codes)
	// In non-TTY contexts lipgloss may strip colors, so just check content preserved
	plain := strings.ReplaceAll(result, "\x1b", "")
	if !strings.Contains(plain, "key") || !strings.Contains(plain, "value") {
		t.Error("colorizeJSON lost content")
	}
}

func TestRunNonInteractive_Mock(t *testing.T) {
	sockPath, cleanup := startMockFzf(t)
	defer cleanup()

	client := NewSocketClient(sockPath, "")
	var buf strings.Builder
	err := RunNonInteractive(client, "up", &buf)
	if err != nil {
		t.Fatalf("RunNonInteractive error: %v", err)
	}
}
