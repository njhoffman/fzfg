package fzfg

import (
	"testing"
)

func TestDebugModes(t *testing.T) {
	tests := []struct {
		input string
		want  map[string]bool
	}{
		{"", map[string]bool{}},
		{"summary", map[string]bool{"summary": true}},
		{"summary,timings", map[string]bool{"summary": true, "timings": true}},
		{"summary,diffs,timings,envs", map[string]bool{
			"summary": true, "diffs": true, "timings": true, "envs": true,
		}},
		{"trace", map[string]bool{"trace": true}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DebugModes(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("DebugModes(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for k := range tt.want {
				if !got[k] {
					t.Errorf("DebugModes(%q) missing key %q", tt.input, k)
				}
			}
		})
	}
}

func TestWrapLong(t *testing.T) {
	short := "hello world"
	if wrapLong(short, 80) != short {
		t.Error("short string should not be wrapped")
	}

	long := "a bb ccc dddd eeeee ffffff ggggggg hhhhhhhh iiiiiiiii jjjjjjjjjj"
	wrapped := wrapLong(long, 30)
	if len(wrapped) <= len(long) {
		t.Error("long string should be wrapped with newlines")
	}
}
