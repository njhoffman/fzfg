package fzfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	CONFIG_FILE_NAME = "fzfg.yaml"
	CONFIG_FILE_PATH = "$HOME/.config/fzfg/fzfg.yaml"
)

type Config struct {
	Logger   *LoggerConfig      `yaml:"logger"`
	Commands map[string]Command `yaml:"commands"`
	Options  map[string]Options `yaml:"options"`
	Profiles map[string]Profile `yaml:"profiles"`
}

type Profile struct {
	Command Command
	Options Options
}

// ModuleConfig represents a module's commands, options, and profiles.
type ModuleConfig struct {
	Commands map[string]Command `yaml:"commands"`
	Options  map[string]Options `yaml:"options"`
	Profiles map[string]Profile `yaml:"profiles"`
}

func LoadConfig() (Config, error) {
	var conf Config

	confFile, err := configFile()
	if err != nil {
		return conf, err
	}

	// Preprocess includes to generate combined config.yaml
	combinedPath, err := preprocessIncludes(confFile)
	if err != nil {
		return conf, fmt.Errorf("preprocessing includes: %w", err)
	}

	confRaw, err := os.ReadFile(combinedPath)
	if err != nil {
		return conf, err
	}

	// First unmarshal into generic map to discover module sections
	var raw map[string]interface{}
	if err := yaml.Unmarshal(confRaw, &raw); err != nil {
		return conf, err
	}

	// Unmarshal into Config struct for top-level commands/options/profiles
	if err := yaml.Unmarshal(confRaw, &conf); err != nil {
		return conf, err
	}

	// Discover and merge module sections.
	// Module sections are top-level keys (other than commands, options, profiles,
	// bindings, previews) that contain a map with commands/options/profiles sub-keys.
	knownKeys := map[string]bool{
		"commands": true, "options": true, "profiles": true,
		"bindings": true, "previews": true, "logger": true,
		"layout": true,
	}

	for key, val := range raw {
		if knownKeys[key] {
			continue
		}
		moduleMap, ok := val.(map[interface{}]interface{})
		if !ok {
			continue
		}
		if !isModuleSection(moduleMap) {
			continue
		}

		// Re-marshal and unmarshal this section as a ModuleConfig
		moduleBytes, err := yaml.Marshal(moduleMap)
		if err != nil {
			continue
		}
		var mod ModuleConfig
		if err := yaml.Unmarshal(moduleBytes, &mod); err != nil {
			continue
		}

		mergeModule(&conf, &mod)
	}

	// Run validation if -v flag is set
	if ValidateFlag {
		defsDir := filepath.Join(filepath.Dir(confFile), "definitions")
		validationFailed := false
		for groupName, opts := range conf.Options {
			vResult, err := ValidateConfig(opts, defsDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "validation [%s]: %v\n", groupName, err)
				continue
			}
			for _, w := range vResult.Warnings {
				fmt.Fprintf(os.Stderr, "warning [%s]: %s\n", groupName, w)
			}
			for _, e := range vResult.Errors {
				fmt.Fprintf(os.Stderr, "error [%s]: %s\n", groupName, e)
			}
			if vResult.HasErrors() {
				validationFailed = true
			}
		}
		if validationFailed {
			return conf, fmt.Errorf("configuration validation failed")
		}
	}

	return conf, nil
}

// isModuleSection checks if a map looks like a module config (has at least one
// of commands, options, or profiles as a key).
func isModuleSection(m map[interface{}]interface{}) bool {
	for k := range m {
		if s, ok := k.(string); ok {
			switch s {
			case "commands", "options", "profiles":
				return true
			}
		}
	}
	return false
}

// mergeModule merges a module's commands, options, and profiles into the
// top-level Config maps. Module keys override global keys on collision.
func mergeModule(conf *Config, mod *ModuleConfig) {
	if conf.Commands == nil {
		conf.Commands = make(map[string]Command)
	}
	for k, v := range mod.Commands {
		conf.Commands[k] = v
	}

	if conf.Options == nil {
		conf.Options = make(map[string]Options)
	}
	for k, v := range mod.Options {
		conf.Options[k] = v
	}

	if conf.Profiles == nil {
		conf.Profiles = make(map[string]Profile)
	}
	for k, v := range mod.Profiles {
		conf.Profiles[k] = v
	}
}

func ParseConfig(commandConf Command, optionsConf Options) string {
	command := parseCommand(commandConf)
	options := parseOptions(optionsConf)
	output := ""
	if !RawFlag {
		if len(command) > 0 {
			output += fmt.Sprintf(
				"export %s=%q\n", "FZF_DEFAULT_COMMAND", strings.Join(command, " "),
			)
		}
		if len(options) > 0 {
			output += fmt.Sprintf(
				"export %s=%q\n", "FZF_DEFAULT_OPTS", strings.Join(options, " "),
			)
		}
	} else {
		if len(command) > 0 {
			output += fmt.Sprintf("%s\n", strings.Join(command, "\n"))
		}
		if len(options) > 0 {
			if len(command) > 0 {
				output += "\x1E"
			}
			output += fmt.Sprintf("%s\n", strings.Join(options, "\n"))
		}
	}
	return output
}

func configFile() (string, error) {
	var confFile string

	envConfigFile := os.Getenv("FZFG_CONF")
	xdgConfigFile := os.ExpandEnv(CONFIG_FILE_PATH)

	if envConfigFile != "" && isFile(envConfigFile) {
		return envConfigFile, nil
	} else if isFile(xdgConfigFile) {
		return xdgConfigFile, nil
	} else if isFile(CONFIG_FILE_NAME) {
		return CONFIG_FILE_NAME, nil
	} else {
		return confFile, fmt.Errorf("Unable to find the configuration file")
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
