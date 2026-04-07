package fzfg

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name  string
		input Command
		want  []string
	}{
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "string command",
			input: "find . -type f",
			want:  []string{"find . -type f"},
		},
		{
			name:  "sequence command",
			input: []interface{}{"fd", "--color=always", "--hidden"},
			want:  []string{"fd", "--color=always", "--hidden"},
		},
		{
			name: "nested sequence",
			input: []interface{}{
				[]interface{}{"fd", "--color=always"},
				"--type=f",
			},
			want: []string{"fd", "--color=always", "--type=f"},
		},
		{
			name:  "unsupported type returns empty",
			input: 42,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommand(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
