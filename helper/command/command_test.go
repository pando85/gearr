package command

import (
	"reflect"
	"testing"
)

func TestStringToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "ffmpeg -i input.mp4 output.mp4",
			expected: []string{"ffmpeg", "-i", "input.mp4", "output.mp4"},
		},
		{
			name:     "double quoted word in middle",
			input:    `cmd "hello world" rest`,
			expected: []string{"cmd", "hello world", "rest"},
		},
		{
			name:     "single quoted word in middle",
			input:    `cmd 'hello world' rest`,
			expected: []string{"cmd", "hello world", "rest"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "single word",
			input:    "ffmpeg",
			expected: []string{"ffmpeg"},
		},
		{
			name:     "multiple spaces",
			input:    "cmd   arg1    arg2",
			expected: []string{"cmd", "arg1", "arg2"},
		},
		{
			name:     "quoted word followed by word",
			input:    `ffmpeg -i "input file.mp4" -c:v libx265`,
			expected: []string{"ffmpeg", "-i", "input file.mp4", "-c:v", "libx265"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToSlice(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("StringToSlice(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetWD(t *testing.T) {
	wd := GetWD()
	if wd == "" {
		t.Error("GetWD() returned empty string")
	}
}

func TestGetFullCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *Command
		expected string
	}{
		{
			name:     "simple command",
			cmd:      NewCommand("ffmpeg", "-i", "input.mp4"),
			expected: "ffmpeg -i input.mp4",
		},
		{
			name:     "multiple params",
			cmd:      NewCommand("echo", "hello", "world"),
			expected: "echo hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cmd.GetFullCommand()
			if result != tt.expected {
				t.Errorf("GetFullCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}
