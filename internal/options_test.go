package fzfg

import (
	"strings"
	"testing"
)

func TestOptionFromBool(t *testing.T) {
	tests := []struct {
		name  string
		flag  string
		value interface{}
		want  string
	}{
		{"true bool", "ansi", true, "--ansi"},
		{"false bool", "exact", false, "--no-exact"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optionFromBool(tt.flag, tt.value)
			if got != tt.want {
				t.Errorf("optionFromBool(%q, %v) = %q, want %q", tt.flag, tt.value, got, tt.want)
			}
		})
	}
}

func TestOptionFromNumber(t *testing.T) {
	got := optionFromNumber("height", 50)
	want := "--height=50"
	if got != want {
		t.Errorf("optionFromNumber() = %q, want %q", got, want)
	}
}

func TestOptionFromString(t *testing.T) {
	RawFlag = false
	tests := []struct {
		name  string
		key   string
		value interface{}
		want  string
	}{
		{"simple value", "delimiter", ":", "--delimiter=:"},
		{"value with spaces", "prompt", "search: ", "--prompt='search: '"},
		{"bind key always quoted", "bind", "ctrl-a:select-all", "--bind='ctrl-a:select-all'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optionFromString(tt.key, tt.value)
			if got != tt.want {
				t.Errorf("optionFromString(%q, %v) = %q, want %q", tt.key, tt.value, got, tt.want)
			}
		})
	}
}

func TestOptionFromSlice_Preview(t *testing.T) {
	RawFlag = false
	input := []interface{}{"bat --color=always", "-- {}"}
	got := optionFromSlice("preview", input)
	// preview joins with spaces, not commas
	if !strings.Contains(got, "--preview=") {
		t.Errorf("expected --preview= prefix, got %q", got)
	}
	if strings.Contains(got, ",") {
		t.Errorf("preview should join with spaces not commas, got %q", got)
	}
}

func TestOptionFromSlice_NonPreview(t *testing.T) {
	RawFlag = false
	input := []interface{}{"val1", "val2", "val3"}
	got := optionFromSlice("header-lines", input)
	want := "--header-lines=val1,val2,val3"
	if got != want {
		t.Errorf("optionFromSlice() = %q, want %q", got, want)
	}
}

func TestParseOptions_Nil(t *testing.T) {
	got := parseOptions(nil)
	if len(got) != 0 {
		t.Errorf("parseOptions(nil) should return empty slice, got %v", got)
	}
}

func TestParseOptions_Sorted(t *testing.T) {
	opts := Options{
		"reverse": true,
		"ansi":    true,
		"exact":   false,
	}
	got := parseOptions(opts)
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("options not sorted: %v", got)
			break
		}
	}
}
