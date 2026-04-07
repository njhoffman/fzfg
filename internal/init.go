package fzfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Steps      []InitStep
	Total      time.Duration
	Config     Config
	TermInfo   TerminalInfo
	Preview    PreviewLayout
	TmuxLayout TmuxLayout
	FinalCmd   string
	FinalOpts  string
	RscOpts    string // options loaded from fzfrc
	EnvOpts    string // options from FZF_DEFAULT_OPTS env
	EnvCmd     string // command from FZF_DEFAULT_COMMAND env
	ConfigFile string
}

// AllInitSteps is the ordered list of step names.
// config must run before validate since validation needs loaded config.
var AllInitSteps = []string{"start", "config", "validate", "rsc-load", "env-load", "env-set"}

// RunInit executes the init pipeline. If initStep is empty, runs all steps.
// If initStep names a specific step, runs only up to and including that step.
func RunInit(initStep string) (*InitResult, error) {
	result := &InitResult{}
	pipelineStart := time.Now()

	// Determine which steps to run
	stepsToRun := AllInitSteps
	if initStep != "" {
		stepsToRun = stepsUpTo(initStep)
		if len(stepsToRun) == 0 {
			return nil, fmt.Errorf("unknown init step %q, valid steps: %s",
				initStep, strings.Join(AllInitSteps, ", "))
		}
	}

	stepSet := map[string]bool{}
	for _, s := range stepsToRun {
		stepSet[s] = true
	}

	// =========================================================================
	// Step: start — detect terminal, locate config
	// =========================================================================
	if stepSet["start"] {
		step := runStep("start", func(l *log.Logger) (map[string]string, error) {
			l.Info("initializing fzfg", "module", ModuleFlag)

			// Detect terminal
			result.TermInfo = DetectTerminal()
			l.Info("terminal detected",
				"width", result.TermInfo.Width,
				"height", result.TermInfo.Height,
				"tmux", result.TermInfo.InTmux,
			)

			// Compute preview layout
			lcfg := DefaultLayoutConfig()
			result.Preview = ComputePreviewLayout(result.TermInfo, lcfg)
			l.Info("preview layout", "setting", result.Preview.Setting)

			// Compute tmux layout
			if result.TermInfo.InTmux {
				result.TmuxLayout = ComputeTmuxLayout(
					result.TermInfo, "85%", "75%", 2, 140,
				)
				l.Info("tmux layout",
					"fzf-tmux", result.TmuxLayout.UseFzfTmux,
					"popup", result.TmuxLayout.UsePopup,
				)
			}

			// Locate config
			confFile, err := configFile()
			if err != nil {
				return nil, err
			}
			result.ConfigFile = confFile
			l.Info("config file", "path", confFile)

			snap := FormatTerminalInfo(result.TermInfo)
			snap["config_file"] = confFile
			snap["preview"] = result.Preview.Setting
			return snap, nil
		})
		result.Steps = append(result.Steps, step)
		if step.Status == "error" {
			result.Total = time.Since(pipelineStart)
			return result, fmt.Errorf("start: %s", step.Message)
		}
	}

	// =========================================================================
	// Step: config — load global config + merge modules
	// Precedence layer 1: app defaults (fzf built-in defaults)
	// Precedence layer 2: loaded here (global config + module config)
	// =========================================================================
	if stepSet["config"] {
		step := runStep("config", func(l *log.Logger) (map[string]string, error) {
			confFile := result.ConfigFile
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
			if err := yaml.Unmarshal(confRaw, &result.Config); err != nil {
				return nil, err
			}

			// Merge modules
			knownKeys := map[string]bool{
				"commands": true, "options": true, "profiles": true,
				"bindings": true, "previews": true, "logger": true,
				"layout": true,
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
				mergeModule(&result.Config, &mod)
				moduleCount++
			}

			// Initialize loggers from config
			if result.Config.Logger != nil {
				if _, err := InitLoggers(result.Config.Logger); err != nil {
					l.Warn("logger init failed", "err", err)
				}
			}

			l.Info("config loaded",
				"commands", len(result.Config.Commands),
				"options", len(result.Config.Options),
				"profiles", len(result.Config.Profiles),
				"modules", moduleCount,
			)

			return map[string]string{
				"commands": fmt.Sprintf("%d", len(result.Config.Commands)),
				"options":  fmt.Sprintf("%d", len(result.Config.Options)),
				"profiles": fmt.Sprintf("%d", len(result.Config.Profiles)),
				"modules":  fmt.Sprintf("%d", moduleCount),
			}, nil
		})
		result.Steps = append(result.Steps, step)
		if step.Status == "error" {
			result.Total = time.Since(pipelineStart)
			return result, fmt.Errorf("config: %s", step.Message)
		}
	}

	// =========================================================================
	// Step: validate — check user config against definitions
	// =========================================================================
	if stepSet["validate"] {
		step := runStep("validate", func(l *log.Logger) (map[string]string, error) {
			confFile := result.ConfigFile
			defsDir := filepath.Join(filepath.Dir(confFile), "definitions")
			if _, err := os.Stat(defsDir); os.IsNotExist(err) {
				l.Warn("definitions not found, skipping", "path", defsDir)
				return map[string]string{"status": "skipped"}, nil
			}

			errCount, warnCount := 0, 0
			for groupName, opts := range result.Config.Options {
				vResult, err := ValidateConfig(opts, defsDir)
				if err != nil {
					l.Warn("validation failed", "group", groupName, "err", err)
					continue
				}
				for _, w := range vResult.Warnings {
					l.Warn("warn", "group", groupName, "option", w.Option, "msg", w.Message)
					warnCount++
				}
				for _, e := range vResult.Errors {
					l.Error("error", "group", groupName, "option", e.Option, "msg", e.Message)
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
	}

	// =========================================================================
	// Step: rsc-load — load FZF_DEFAULT_OPTS_FILE (~/.fzfrc)
	// Precedence layer 3: fzfrc options (merged over app defaults, before user config)
	// =========================================================================
	if stepSet["rsc-load"] {
		step := runStep("rsc-load", func(l *log.Logger) (map[string]string, error) {
			optsFile := os.Getenv("FZF_DEFAULT_OPTS_FILE")
			if optsFile == "" {
				home, _ := os.UserHomeDir()
				optsFile = filepath.Join(home, ".fzfrc")
			}

			if !isFile(optsFile) {
				l.Info("no fzfrc found", "path", optsFile)
				return map[string]string{"status": "no-file"}, nil
			}

			data, err := os.ReadFile(optsFile)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", optsFile, err)
			}

			content := strings.TrimSpace(string(data))
			// Collapse multi-line fzfrc into single line (lines ending with \ are continued)
			content = strings.ReplaceAll(content, "\\\n", "")
			content = strings.Join(strings.Fields(content), " ")

			result.RscOpts = content
			optCount := len(strings.Fields(content))
			l.Info("loaded fzfrc", "path", optsFile, "options", optCount)
			return map[string]string{
				"file":    optsFile,
				"options": fmt.Sprintf("%d", optCount),
				"content": content,
			}, nil
		})
		result.Steps = append(result.Steps, step)
	}

	// =========================================================================
	// Step: env-load — parse FZF_DEFAULT_OPTS and FZF_DEFAULT_COMMAND from env
	// Precedence layer 4: env vars (captured before we override them)
	// =========================================================================
	if stepSet["env-load"] {
		step := runStep("env-load", func(l *log.Logger) (map[string]string, error) {
			snap := map[string]string{}

			result.EnvOpts = os.Getenv("FZF_DEFAULT_OPTS")
			result.EnvCmd = os.Getenv("FZF_DEFAULT_COMMAND")

			if result.EnvOpts != "" {
				l.Info("FZF_DEFAULT_OPTS from env", "len", len(result.EnvOpts))
				snap["FZF_DEFAULT_OPTS"] = result.EnvOpts
			} else {
				l.Info("FZF_DEFAULT_OPTS not set")
			}

			if result.EnvCmd != "" {
				l.Info("FZF_DEFAULT_COMMAND from env", "cmd", result.EnvCmd)
				snap["FZF_DEFAULT_COMMAND"] = result.EnvCmd
			} else {
				l.Info("FZF_DEFAULT_COMMAND not set")
			}

			return snap, nil
		})
		result.Steps = append(result.Steps, step)
	}

	// =========================================================================
	// Step: env-set — generate final FZF env var values
	// Precedence: app defaults -> fzfrc -> config defaults -> module config
	// Module config is the highest precedence (most specific).
	// =========================================================================
	if stepSet["env-set"] {
		step := runStep("env-set", func(l *log.Logger) (map[string]string, error) {
			snap := map[string]string{}
			conf := result.Config

			// Resolve which profile/command/options to use
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
				result.FinalCmd = strings.Join(command, " ")
				snap["FZF_DEFAULT_COMMAND"] = result.FinalCmd
				l.Info("FZF_DEFAULT_COMMAND set", "len", len(result.FinalCmd))
			}
			if len(options) > 0 {
				result.FinalOpts = strings.Join(options, " ")
				snap["FZF_DEFAULT_OPTS"] = result.FinalOpts
				l.Info("FZF_DEFAULT_OPTS set", "count", len(options))
			}

			return snap, nil
		})
		result.Steps = append(result.Steps, step)
	}

	result.Total = time.Since(pipelineStart)
	return result, nil
}

// stepsUpTo returns all steps up to and including the named step.
func stepsUpTo(name string) []string {
	for i, s := range AllInitSteps {
		if s == name {
			return AllInitSteps[:i+1]
		}
	}
	return nil
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
