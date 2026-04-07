package fzfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFzfrcBindings(t *testing.T) {
	tmpDir := t.TempDir()
	fzfrc := filepath.Join(tmpDir, ".fzfrc")

	content := `--ansi
--multi
# delete query to start of line
--bind 'ctrl-u:clear-query'
# scrolling: half-page smooth scroll up
--bind 'ctrl-b:half-page-up+offset-middle'
# no comment for this one
--bind 'ctrl-g:top'
`
	os.WriteFile(fzfrc, []byte(content), 0644)

	binds, err := ParseFzfrcBindings(fzfrc)
	if err != nil {
		t.Fatalf("ParseFzfrcBindings error: %v", err)
	}

	if len(binds) != 3 {
		t.Fatalf("expected 3 bindings, got %d", len(binds))
	}

	// First binding should have the comment description
	if binds[0].Key != "ctrl-u" {
		t.Errorf("bind[0].Key = %q, want 'ctrl-u'", binds[0].Key)
	}
	if binds[0].Action != "clear-query" {
		t.Errorf("bind[0].Action = %q, want 'clear-query'", binds[0].Action)
	}
	if binds[0].Description != "delete query to start of line" {
		t.Errorf("bind[0].Description = %q", binds[0].Description)
	}
	if binds[0].Source != "fzfrc" {
		t.Errorf("bind[0].Source = %q, want 'fzfrc'", binds[0].Source)
	}

	// Second binding
	if binds[1].Description != "scrolling: half-page smooth scroll up" {
		t.Errorf("bind[1].Description = %q", binds[1].Description)
	}

	// Third binding has no directly preceding comment
	// (the "no comment" line is followed by a blank implicit reset,
	// but actually the comment IS directly above it)
	if binds[2].Key != "ctrl-g" {
		t.Errorf("bind[2].Key = %q, want 'ctrl-g'", binds[2].Key)
	}
}

func TestParseConfigBindings(t *testing.T) {
	tmpDir := t.TempDir()
	confPath := filepath.Join(tmpDir, "fzfg.yaml")

	content := `bindings:
  jump:
    alt-g: jump
    alt-j: jump-accept
  scroll:
    alt-d: preview-half-page-down
    alt-u: preview-half-page-up
`
	os.WriteFile(confPath, []byte(content), 0644)

	binds, err := ParseConfigBindings(confPath)
	if err != nil {
		t.Fatalf("ParseConfigBindings error: %v", err)
	}

	if len(binds) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(binds))
	}

	// All should be source "config"
	for _, b := range binds {
		if b.Source != "config" {
			t.Errorf("bind %q source = %q, want 'config'", b.Key, b.Source)
		}
		if b.Category == "" {
			t.Errorf("bind %q should have a category", b.Key)
		}
	}
}

func TestFzfDefaultBindings(t *testing.T) {
	defaults := FzfDefaultBindings()
	if len(defaults) == 0 {
		t.Fatal("expected non-empty default bindings")
	}

	// Check a known default
	found := false
	for _, b := range defaults {
		if b.Key == "enter" && b.Action == "accept" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enter -> accept in defaults")
	}

	// All should have source "default"
	for _, b := range defaults {
		if b.Source != "default" {
			t.Errorf("default bind %q source = %q", b.Key, b.Source)
		}
	}
}

func TestCollectAllBindings_Precedence(t *testing.T) {
	tmpDir := t.TempDir()

	// fzfrc overrides ctrl-u from default (unix-line-discard -> clear-query)
	fzfrc := filepath.Join(tmpDir, ".fzfrc")
	os.WriteFile(fzfrc, []byte("# clear query\n--bind 'ctrl-u:clear-query'\n"), 0644)

	// config overrides alt-d from default (kill-word -> preview-half-page-down)
	confPath := filepath.Join(tmpDir, "fzfg.yaml")
	os.WriteFile(confPath, []byte("bindings:\n  scroll:\n    alt-d: preview-half-page-down\n"), 0644)

	binds := CollectAllBindings(fzfrc, confPath)

	byKey := map[string]Keybind{}
	for _, b := range binds {
		byKey[b.Key] = b
	}

	// ctrl-u should come from fzfrc (overrides default)
	if b, ok := byKey["ctrl-u"]; ok {
		if b.Source != "fzfrc" {
			t.Errorf("ctrl-u source = %q, want 'fzfrc'", b.Source)
		}
		if b.Action != "clear-query" {
			t.Errorf("ctrl-u action = %q, want 'clear-query'", b.Action)
		}
	} else {
		t.Error("expected ctrl-u in collected bindings")
	}

	// alt-d should come from config (overrides default and fzfrc)
	if b, ok := byKey["alt-d"]; ok {
		if b.Source != "config" {
			t.Errorf("alt-d source = %q, want 'config'", b.Source)
		}
	} else {
		t.Error("expected alt-d in collected bindings")
	}

	// enter should still be from default
	if b, ok := byKey["enter"]; ok {
		if b.Source != "default" {
			t.Errorf("enter source = %q, want 'default'", b.Source)
		}
	}
}

func TestAvailableKeys(t *testing.T) {
	bound := []Keybind{
		{Key: "ctrl-a"},
		{Key: "ctrl-b"},
		{Key: "enter"},
	}

	available := AvailableKeys(bound)

	// ctrl-a and ctrl-b should NOT be in available
	for _, k := range available {
		if k == "ctrl-a" || k == "ctrl-b" || k == "enter" {
			t.Errorf("key %q should not be available (it's bound)", k)
		}
	}

	// ctrl-c should be available (not in our bound list)
	found := false
	for _, k := range available {
		if k == "ctrl-c" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ctrl-c should be available")
	}

	// Should be fewer than total bindable keys
	if len(available) >= len(AllBindableKeys) {
		t.Error("available should be fewer than total bindable keys")
	}
}

func TestFormatKeybindPreview(t *testing.T) {
	binds := []Keybind{
		{Key: "ctrl-a", Action: "beginning-of-line", Description: "Start of line", Source: "default", Category: "editing"},
		{Key: "ctrl-u", Action: "clear-query", Description: "Clear query", Source: "fzfrc", Category: "editing"},
	}

	output := FormatKeybindPreview(binds)

	if !strings.Contains(output, "Key Bindings") {
		t.Error("output should contain header")
	}
	if !strings.Contains(output, "ctrl-a") {
		t.Error("output should contain ctrl-a")
	}
	if !strings.Contains(output, "ctrl-u") {
		t.Error("output should contain ctrl-u")
	}
}

func TestFormatAvailableKeys(t *testing.T) {
	keys := []string{"ctrl-c", "ctrl-d", "alt-x", "f1", "space"}
	output := FormatAvailableKeys(keys)

	if !strings.Contains(output, "Available") {
		t.Error("output should contain header")
	}
	if !strings.Contains(output, "ctrl-c") {
		t.Error("output should contain ctrl-c")
	}
}
