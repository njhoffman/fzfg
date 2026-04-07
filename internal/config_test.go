package fzfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsModuleSection(t *testing.T) {
	tests := []struct {
		name string
		m    map[interface{}]interface{}
		want bool
	}{
		{
			name: "has commands",
			m:    map[interface{}]interface{}{"commands": nil, "module": nil},
			want: true,
		},
		{
			name: "has options",
			m:    map[interface{}]interface{}{"options": nil},
			want: true,
		},
		{
			name: "has profiles",
			m:    map[interface{}]interface{}{"profiles": nil},
			want: true,
		},
		{
			name: "no module keys",
			m:    map[interface{}]interface{}{"bindings": nil, "previews": nil},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isModuleSection(tt.m)
			if got != tt.want {
				t.Errorf("isModuleSection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeModule(t *testing.T) {
	conf := Config{
		Commands: map[string]Command{"global_cmd": "echo global"},
		Options:  map[string]Options{"global_opt": {"ansi": true}},
		Profiles: map[string]Profile{},
	}

	mod := ModuleConfig{
		Commands: map[string]Command{"mod_cmd": "echo module"},
		Options:  map[string]Options{"mod_opt": {"reverse": true}},
		Profiles: map[string]Profile{"mod_profile": {Command: "echo test"}},
	}

	mergeModule(&conf, &mod)

	if _, ok := conf.Commands["mod_cmd"]; !ok {
		t.Error("module command not merged")
	}
	if _, ok := conf.Commands["global_cmd"]; !ok {
		t.Error("global command lost during merge")
	}
	if _, ok := conf.Options["mod_opt"]; !ok {
		t.Error("module options not merged")
	}
	if _, ok := conf.Profiles["mod_profile"]; !ok {
		t.Error("module profile not merged")
	}
}

func TestMergeModule_NilMaps(t *testing.T) {
	conf := Config{}
	mod := ModuleConfig{
		Commands: map[string]Command{"cmd": "test"},
	}

	mergeModule(&conf, &mod)

	if conf.Commands == nil {
		t.Error("Commands map should be initialized")
	}
	if _, ok := conf.Commands["cmd"]; !ok {
		t.Error("module command not merged into nil map")
	}
}

func TestConfigFile_EnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "test.yaml")
	os.WriteFile(confPath, []byte("commands: {}"), 0644)

	t.Setenv("FZFG_CONF", confPath)

	got, err := configFile()
	if err != nil {
		t.Fatalf("configFile() error: %v", err)
	}
	if got != confPath {
		t.Errorf("configFile() = %q, want %q", got, confPath)
	}
}

func TestConfigFile_NotFound(t *testing.T) {
	t.Setenv("FZFG_CONF", "")
	t.Setenv("HOME", t.TempDir())

	// Change to temp dir so ./fzfg.yaml doesn't exist
	origDir, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(origDir)

	_, err := configFile()
	if err == nil {
		t.Error("expected error when no config file exists")
	}
}

func TestParseConfig_ExportFormat(t *testing.T) {
	RawFlag = false

	cmd := Command([]interface{}{"fd", "--hidden"})
	opts := Options{"ansi": true, "reverse": true}

	output := ParseConfig(cmd, opts)

	if !strings.Contains(output, "export FZF_DEFAULT_COMMAND=") {
		t.Error("expected FZF_DEFAULT_COMMAND export")
	}
	if !strings.Contains(output, "export FZF_DEFAULT_OPTS=") {
		t.Error("expected FZF_DEFAULT_OPTS export")
	}
}

func TestParseConfig_RawFormat(t *testing.T) {
	RawFlag = true
	defer func() { RawFlag = false }()

	cmd := Command("echo test")
	opts := Options{"ansi": true}

	output := ParseConfig(cmd, opts)

	if strings.Contains(output, "export") {
		t.Error("raw format should not contain export")
	}
	if !strings.Contains(output, "\x1E") {
		t.Error("raw format should contain record separator between command and options")
	}
}
