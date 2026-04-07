package fzfg

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

// LoggerConfig represents the logger configuration from fzfg.yaml.
type LoggerConfig struct {
	Default *LogOutputConfig `yaml:"default"`
	Debug   *LogOutputConfig `yaml:"debug"`
	HTTP    *LogOutputConfig `yaml:"http"`
}

// LogOutputConfig represents a single logger output configuration.
type LogOutputConfig struct {
	Level           string `yaml:"level"`
	Console         bool   `yaml:"console"`
	Format          string `yaml:"format"`
	ReportCaller    bool   `yaml:"report-caller"`
	ReportTimestamp bool   `yaml:"report-timestamp"`
	TimeFormat      string `yaml:"time-format"`
	File            string `yaml:"file"`
	ForceLevel      string `yaml:"force-level"`
	Prefix          string `yaml:"prefix"`
}

// Loggers holds all configured logger instances.
type Loggers struct {
	Default *log.Logger
	Debug   *log.Logger
	HTTP    *log.Logger
}

// Log is the package-level default logger, initialized after config load.
var Log *log.Logger

// DebugLog is the file-based debug logger for JSON output.
var DebugLog *log.Logger

// InitLoggers creates logger instances from the parsed config.
func InitLoggers(cfg *LoggerConfig) (*Loggers, error) {
	loggers := &Loggers{}

	if cfg == nil {
		// No logger config — create a minimal default
		Log = log.NewWithOptions(os.Stderr, log.Options{
			Level: log.InfoLevel,
		})
		loggers.Default = Log
		return loggers, nil
	}

	// Default (console) logger
	if cfg.Default != nil {
		logger, err := createLogger(cfg.Default, os.Stderr)
		if err != nil {
			return nil, fmt.Errorf("creating default logger: %w", err)
		}
		loggers.Default = logger
		Log = logger
	} else {
		Log = log.NewWithOptions(os.Stderr, log.Options{Level: log.InfoLevel})
		loggers.Default = Log
	}

	// Debug (file) logger
	if cfg.Debug != nil && cfg.Debug.File != "" {
		f, err := os.OpenFile(cfg.Debug.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			Log.Warn("failed to open debug log file", "path", cfg.Debug.File, "err", err)
		} else {
			logger, err := createLogger(cfg.Debug, f)
			if err != nil {
				return nil, fmt.Errorf("creating debug logger: %w", err)
			}
			loggers.Debug = logger
			DebugLog = logger
		}
	}

	// HTTP logger (standard log adapter)
	if cfg.HTTP != nil {
		var w io.Writer = os.Stderr
		if !cfg.HTTP.Console {
			w = io.Discard
		}
		lvl := parseLevel(cfg.HTTP.ForceLevel)
		httpLogger := log.NewWithOptions(w, log.Options{
			Level:  lvl,
			Prefix: cfg.HTTP.Prefix,
		})
		loggers.HTTP = httpLogger
	}

	return loggers, nil
}

// createLogger builds a charmbracelet/log.Logger from a config section.
func createLogger(cfg *LogOutputConfig, w io.Writer) (*log.Logger, error) {
	opts := log.Options{
		Level:           parseLevel(cfg.Level),
		ReportTimestamp: cfg.ReportTimestamp,
		ReportCaller:    cfg.ReportCaller,
		Prefix:          cfg.Prefix,
	}

	if cfg.TimeFormat != "" {
		opts.TimeFormat = resolveTimeFormat(cfg.TimeFormat)
	}

	logger := log.NewWithOptions(w, opts)

	switch cfg.Format {
	case "json":
		logger.SetFormatter(log.JSONFormatter)
	case "logfmt":
		logger.SetFormatter(log.LogfmtFormatter)
	case "text", "":
		logger.SetFormatter(log.TextFormatter)
	}

	return logger, nil
}

// WithPrefix returns a sub-logger with the given prefix for pipeline step logging.
func WithPrefix(prefix string) *log.Logger {
	if Log == nil {
		return log.NewWithOptions(os.Stderr, log.Options{
			Level:  log.InfoLevel,
			Prefix: prefix,
		})
	}
	return Log.WithPrefix(prefix)
}

// parseLevel converts a level string to a charmbracelet/log.Level.
func parseLevel(s string) log.Level {
	lvl, err := log.ParseLevel(s)
	if err != nil {
		return log.InfoLevel
	}
	return lvl
}

// resolveTimeFormat maps user-friendly time format names to Go format strings.
func resolveTimeFormat(s string) string {
	switch s {
	case "time.Kitchen":
		return time.Kitchen
	case "time.RFC3339":
		return time.RFC3339
	case "time.DateTime":
		return time.DateTime
	default:
		return s
	}
}
