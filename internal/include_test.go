package fzfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIncludeDirective(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantPat   string
		wantMatch bool
	}{
		{
			name:      "valid include",
			line:      "files: !include modules/*.yaml",
			wantKey:   "files",
			wantPat:   "modules/*.yaml",
			wantMatch: true,
		},
		{
			name:      "with leading spaces",
			line:      "  network: !include modules/network.yaml",
			wantKey:   "network",
			wantPat:   "modules/network.yaml",
			wantMatch: true,
		},
		{
			name:      "no include directive",
			line:      "options:",
			wantMatch: false,
		},
		{
			name:      "include in comment",
			line:      "# files: !include foo",
			wantKey:   "# files",
			wantPat:   "foo",
			wantMatch: true,
		},
		{
			name:      "empty pattern",
			line:      "files: !include",
			wantMatch: false,
		},
		{
			name:      "no colon",
			line:      "!include modules/*.yaml",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, pat, ok := parseIncludeDirective(tt.line)
			if ok != tt.wantMatch {
				t.Errorf("parseIncludeDirective(%q) matched=%v, want %v", tt.line, ok, tt.wantMatch)
				return
			}
			if !ok {
				return
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if pat != tt.wantPat {
				t.Errorf("pattern = %q, want %q", pat, tt.wantPat)
			}
		})
	}
}

func TestExpandInclude(t *testing.T) {
	// Create temp directory with a module file
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "modules")
	os.MkdirAll(modDir, 0755)

	moduleContent := "commands:\n  test_cmd: echo hello\n"
	os.WriteFile(filepath.Join(modDir, "test.yaml"), []byte(moduleContent), 0644)

	result, err := expandInclude(tmpDir, "testmod", "modules/*.yaml")
	if err != nil {
		t.Fatalf("expandInclude() error: %v", err)
	}

	if !strings.HasPrefix(result, "testmod:") {
		t.Errorf("result should start with key header, got: %s", result[:40])
	}
	if !strings.Contains(result, "  commands:") {
		t.Errorf("module content should be indented, got:\n%s", result)
	}
	if !strings.Contains(result, "    test_cmd: echo hello") {
		t.Errorf("module content should be nested, got:\n%s", result)
	}
}

func TestExpandInclude_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()
	result, err := expandInclude(tmpDir, "empty", "nonexistent/*.yaml")
	if err != nil {
		t.Fatalf("expandInclude() error: %v", err)
	}
	if !strings.Contains(result, "no matching files") {
		t.Errorf("expected 'no matching files' comment, got: %s", result)
	}
}

func TestPreprocessIncludes(t *testing.T) {
	tmpDir := t.TempDir()
	modDir := filepath.Join(tmpDir, "modules")
	os.MkdirAll(modDir, 0755)

	// Write a main config with an include
	mainConfig := "options:\n  default:\n    ansi: true\nfiles: !include modules/*.yaml\n"
	mainPath := filepath.Join(tmpDir, "fzfg.yaml")
	os.WriteFile(mainPath, []byte(mainConfig), 0644)

	// Write a module
	moduleContent := "commands:\n  fd: [fd, --hidden]\nprofiles:\n  view:\n    command: [ls]\n"
	os.WriteFile(filepath.Join(modDir, "files.yaml"), []byte(moduleContent), 0644)

	combinedPath, err := preprocessIncludes(mainPath)
	if err != nil {
		t.Fatalf("preprocessIncludes() error: %v", err)
	}

	if filepath.Base(combinedPath) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(combinedPath))
	}

	content, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatalf("reading combined config: %v", err)
	}

	combined := string(content)

	// Should have global options
	if !strings.Contains(combined, "options:") {
		t.Error("combined config missing global options")
	}

	// Should have module content nested under files:
	if !strings.Contains(combined, "files:") {
		t.Error("combined config missing files key")
	}
	if !strings.Contains(combined, "  commands:") {
		t.Error("combined config missing indented module commands")
	}

	// Should NOT have raw !include directive
	if strings.Contains(combined, "!include") && !strings.Contains(combined, "# !include") {
		t.Error("combined config still contains raw !include directive")
	}
}
