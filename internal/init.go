package fzfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v2"
)

// InitStep tracks timing and status for one pipeline step.
type InitStep struct {
	Name     string
	Duration time.Duration
	Status   string // "ok", "warn", "error", "skip"
	Message  string
	Snapshot map[string]string // config state after this step
}

// InitResult holds the complete result of the init pipeline.
type InitResult struct {
	Steps     []InitStep
	Total     time.Duration
	Config    Config
	FinalCmd  string // parsed FZF_DEFAULT_COMMAND
	FinalOpts string // parsed FZF_DEFAULT_OPTS
}

// RunInit executes the full initialization pipeline with step-by-step
// logging, timing, and optional debug output.
func RunInit() (*InitResult, error) {
	result := &InitResult{}
	pipelineStart := time.Now()

	// Step 1: start — first boot, locate config
	step := runStep("start", func(l *log.Logger) (map[string]string, error) {
		l.Info("initializing fzfg")
		confFile, err := configFile()
		if err != nil {
			return nil, err
		}
		l.Info("config file located", "path", confFile)
		return map[string]string{"config_file": confFile}, nil
	})
	result.Steps = append(result.Steps, step)
	if step.Status == "error" {
		return result, fmt.Errorf("start: %s", step.Message)
	}
	confFile := step.Snapshot["config_file"]

	// Step 2: config — load and merge global config with modules
	var conf Config
	step = runStep("config", func(l *log.Logger) (map[string]string, error) {
		l.Info("preprocessing includes")
		combinedPath, err := preprocessIncludes(confFile)
		if err != nil {
			return nil, fmt.Errorf("preprocessing: %w", err)
		}
		l.Info("combined config written", "path", combinedPath)

		confRaw, err := os.ReadFile(combinedPath)
		if err != nil {
			return nil, err
		}

		var raw map[string]interface{}
		if err := yaml.Unmarshal(confRaw, &raw); err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(confRaw, &conf); err != nil {
			return nil, err
		}

		// Merge modules
		knownKeys := map[string]bool{
			"commands": true, "options": true, "profiles": true,
			"bindings": true, "previews": true, "logger": true,
		}
		moduleCount := 0
		for key, val := range raw {
			if knownKeys[key] {
				continue
			}
			moduleMap, ok := val.(map[interface{}]interface{})
			if !ok || !isModuleSection(moduleMap) {
				continue
			}
			moduleBytes, err := yaml.Marshal(moduleMap)
			if err != nil {
				continue
			}
			var mod ModuleConfig
			if err := yaml.Unmarshal(moduleBytes, &mod); err != nil {
				continue
			}
			mergeModule(&conf, &mod)
			moduleCount++
		}
		l.Info("config loaded",
			"commands", len(conf.Commands),
			"options", len(conf.Options),
			"profiles", len(conf.Profiles),
			"modules", moduleCount,
		)

		snap := map[string]string{
			"commands": fmt.Sprintf("%d", len(conf.Commands)),
			"options":  fmt.Sprintf("%d", len(conf.Options)),
			"profiles": fmt.Sprintf("%d", len(conf.Profiles)),
			"modules":  fmt.Sprintf("%d", moduleCount),
		}
		return snap, nil
	})
	result.Steps = append(result.Steps, step)
	if step.Status == "error" {
		return result, fmt.Errorf("config: %s", step.Message)
	}

	// Initialize loggers from config
	if conf.Logger != nil {
		if _, err := InitLoggers(conf.Logger); err != nil {
			Log.Warn("logger init failed, using defaults", "err", err)
		}
	}

	// Step 3: validate — validate user config against definitions
	step = runStep("validate", func(l *log.Logger) (map[string]string, error) {
		defsDir := filepath.Join(filepath.Dir(confFile), "definitions")
		if _, err := os.Stat(defsDir); os.IsNotExist(err) {
			l.Warn("definitions directory not found, skipping validation", "path", defsDir)
			return map[string]string{"status": "skipped"}, nil
		}

		errCount, warnCount := 0, 0
		for groupName, opts := range conf.Options {
			vResult, err := ValidateConfig(opts, defsDir)
			if err != nil {
				l.Warn("validation failed for group", "group", groupName, "err", err)
				continue
			}
			for _, w := range vResult.Warnings {
				l.Warn("validation warning", "group", groupName, "option", w.Option, "msg", w.Message)
				warnCount++
			}
			for _, e := range vResult.Errors {
				l.Error("validation error", "group", groupName, "option", e.Option, "msg", e.Message)
				errCount++
			}
		}
		l.Info("validation complete", "errors", errCount, "warnings", warnCount)
		return map[string]string{
			"errors":   fmt.Sprintf("%d", errCount),
			"warnings": fmt.Sprintf("%d", warnCount),
		}, nil
	})
	result.Steps = append(result.Steps, step)

	// Step 4: rsc-load — load FZF_DEFAULT_OPTS_FILE (~/.fzfrc)
	step = runStep("rsc-load", func(l *log.Logger) (map[string]string, error) {
		optsFile := os.Getenv("FZF_DEFAULT_OPTS_FILE")
		if optsFile == "" {
			home, _ := os.UserHomeDir()
			optsFile = filepath.Join(home, ".fzfrc")
		}

		if !isFile(optsFile) {
			l.Info("no FZF_DEFAULT_OPTS_FILE found", "path", optsFile)
			return map[string]string{"status": "no-file"}, nil
		}

		data, err := os.ReadFile(optsFile)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", optsFile, err)
		}

		content := strings.TrimSpace(string(data))
		optCount := len(strings.Fields(content))
		l.Info("loaded fzfrc", "path", optsFile, "options", optCount)
		return map[string]string{
			"file":    optsFile,
			"options": fmt.Sprintf("%d", optCount),
			"content": content,
		}, nil
	})
	result.Steps = append(result.Steps, step)

	// Step 5: env-load — parse FZF_DEFAULT_OPTS and FZF_DEFAULT_COMMAND from env
	step = runStep("env-load", func(l *log.Logger) (map[string]string, error) {
		envOpts := os.Getenv("FZF_DEFAULT_OPTS")
		envCmd := os.Getenv("FZF_DEFAULT_COMMAND")

		snap := map[string]string{}
		if envOpts != "" {
			optCount := len(strings.Fields(envOpts))
			l.Info("FZF_DEFAULT_OPTS from env", "options", optCount)
			snap["FZF_DEFAULT_OPTS"] = envOpts
		} else {
			l.Info("FZF_DEFAULT_OPTS not set in environment")
		}

		if envCmd != "" {
			l.Info("FZF_DEFAULT_COMMAND from env", "command", envCmd)
			snap["FZF_DEFAULT_COMMAND"] = envCmd
		} else {
			l.Info("FZF_DEFAULT_COMMAND not set in environment")
		}

		return snap, nil
	})
	result.Steps = append(result.Steps, step)

	// Step 6: env-set — parse user config into option strings for FZF env vars
	step = runStep("env-set", func(l *log.Logger) (map[string]string, error) {
		// If specific flags are provided, generate the output
		snap := map[string]string{}

		if CommandFlag != "" || OptionsFlag != "" || ProfileFlag != "" {
			var cmdConf Command
			var optsConf Options

			if ProfileFlag != "" {
				if p, ok := conf.Profiles[ProfileFlag]; ok {
					cmdConf = p.Command
					optsConf = p.Options
				}
			} else {
				if CommandFlag != "" {
					cmdConf = conf.Commands[CommandFlag]
				}
				if OptionsFlag != "" {
					optsConf = conf.Options[OptionsFlag]
				}
			}

			command := parseCommand(cmdConf)
			options := parseOptions(optsConf)

			if len(command) > 0 {
				cmdStr := strings.Join(command, " ")
				snap["FZF_DEFAULT_COMMAND"] = cmdStr
				result.FinalCmd = cmdStr
				l.Info("FZF_DEFAULT_COMMAND set", "value_len", len(cmdStr))
			}
			if len(options) > 0 {
				optsStr := strings.Join(options, " ")
				snap["FZF_DEFAULT_OPTS"] = optsStr
				result.FinalOpts = optsStr
				l.Info("FZF_DEFAULT_OPTS set", "options", len(options))
			}
		} else {
			l.Info("no command/options/profile flags specified")
		}

		return snap, nil
	})
	result.Steps = append(result.Steps, step)

	result.Total = time.Since(pipelineStart)
	result.Config = conf

	// Print debug summary if in debug/init mode
	printInitSummary(result)

	return result, nil
}

// runStep executes a pipeline step function with timing and error capture.
func runStep(name string, fn func(*log.Logger) (map[string]string, error)) InitStep {
	l := WithPrefix(name)
	start := time.Now()

	snapshot, err := fn(l)
	duration := time.Since(start)

	step := InitStep{
		Name:     name,
		Duration: duration,
		Snapshot: snapshot,
	}

	if err != nil {
		step.Status = "error"
		step.Message = err.Error()
		l.Error("step failed", "err", err, "duration", duration)
	} else {
		step.Status = "ok"
		l.Debug("step complete", "duration", duration)
	}

	return step
}

// printInitSummary outputs a human-readable timing table and config diff.
func printInitSummary(result *InitResult) {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))
	stepStyle := lipgloss.NewStyle().
		PaddingLeft(2)
	okStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))
	errStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))
	warnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, headerStyle.Render("fzfg init pipeline"))
	fmt.Fprintln(os.Stderr, dimStyle.Render(strings.Repeat("─", 60)))

	for _, step := range result.Steps {
		status := okStyle.Render("ok")
		if step.Status == "error" {
			status = errStyle.Render("err")
		} else if step.Status == "warn" || step.Status == "skip" {
			status = warnStyle.Render(step.Status)
		}

		line := fmt.Sprintf("%-12s  [%s]  %s",
			step.Name,
			status,
			dimStyle.Render(step.Duration.String()),
		)
		fmt.Fprintln(os.Stderr, stepStyle.Render(line))

		if step.Message != "" {
			fmt.Fprintln(os.Stderr, stepStyle.Render(
				errStyle.Render(fmt.Sprintf("  %s", step.Message)),
			))
		}
	}

	fmt.Fprintln(os.Stderr, dimStyle.Render(strings.Repeat("─", 60)))
	fmt.Fprintf(os.Stderr, "%s  %s\n",
		stepStyle.Render("total"),
		dimStyle.Render(result.Total.String()),
	)

	// Print final env strings if set
	if result.FinalCmd != "" {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, headerStyle.Render("FZF_DEFAULT_COMMAND"))
		fmt.Fprintln(os.Stderr, stepStyle.Render(result.FinalCmd))
	}
	if result.FinalOpts != "" {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, headerStyle.Render("FZF_DEFAULT_OPTS"))
		fmt.Fprintln(os.Stderr, stepStyle.Render(result.FinalOpts))
	}
	fmt.Fprintln(os.Stderr)
}
