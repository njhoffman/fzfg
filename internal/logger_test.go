package fzfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  log.Level
	}{
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"fatal", log.FatalLevel},
		{"invalid", log.InfoLevel}, // fallback
		{"", log.InfoLevel},        // fallback
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTimeFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"time.Kitchen", "3:04PM"},
		{"time.RFC3339", "2006-01-02T15:04:05Z07:00"},
		{"time.DateTime", "2006-01-02 15:04:05"},
		{"custom-format", "custom-format"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveTimeFormat(tt.input)
			if got != tt.want {
				t.Errorf("resolveTimeFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInitLoggers_NilConfig(t *testing.T) {
	loggers, err := InitLoggers(nil)
	if err != nil {
		t.Fatalf("InitLoggers(nil) error: %v", err)
	}
	if loggers.Default == nil {
		t.Error("expected non-nil default logger")
	}
	if Log == nil {
		t.Error("expected package-level Log to be set")
	}
}

func TestInitLoggers_DefaultOnly(t *testing.T) {
	cfg := &LoggerConfig{
		Default: &LogOutputConfig{
			Level:   "debug",
			Console: true,
			Format:  "text",
		},
	}
	loggers, err := InitLoggers(cfg)
	if err != nil {
		t.Fatalf("InitLoggers error: %v", err)
	}
	if loggers.Default == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestInitLoggers_WithDebugFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := &LoggerConfig{
		Default: &LogOutputConfig{
			Level:   "info",
			Console: true,
			Format:  "text",
		},
		Debug: &LogOutputConfig{
			Level:  "debug",
			Format: "json",
			File:   logFile,
		},
	}

	loggers, err := InitLoggers(cfg)
	if err != nil {
		t.Fatalf("InitLoggers error: %v", err)
	}
	if loggers.Debug == nil {
		t.Error("expected non-nil debug logger")
	}

	// Write a log entry and verify file exists
	loggers.Debug.Info("test message")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("debug log file was not created")
	}
}

func TestInitLoggers_HTTPLogger(t *testing.T) {
	cfg := &LoggerConfig{
		Default: &LogOutputConfig{Level: "info", Console: true},
		HTTP: &LogOutputConfig{
			ForceLevel: "error",
			Console:    true,
			Prefix:     "http",
		},
	}

	loggers, err := InitLoggers(cfg)
	if err != nil {
		t.Fatalf("InitLoggers error: %v", err)
	}
	if loggers.HTTP == nil {
		t.Error("expected non-nil HTTP logger")
	}
}

func TestWithPrefix(t *testing.T) {
	// Ensure WithPrefix works even before Log is initialized
	origLog := Log
	Log = nil
	defer func() { Log = origLog }()

	l := WithPrefix("test-step")
	if l == nil {
		t.Error("WithPrefix returned nil")
	}
}

func TestCreateLogger_Formats(t *testing.T) {
	formats := []string{"text", "json", "logfmt", ""}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := &LogOutputConfig{
				Level:  "info",
				Format: format,
			}
			logger, err := createLogger(cfg, os.Stderr)
			if err != nil {
				t.Fatalf("createLogger error: %v", err)
			}
			if logger == nil {
				t.Error("expected non-nil logger")
			}
		})
	}
}
