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

func TestNewPanicOption(t *testing.T) {
	opt := NewPanicOption()
	if !opt.PanicOnError {
		t.Error("NewPanicOption().PanicOnError = false, want true")
	}
}

func TestNewAllowedCodesOption(t *testing.T) {
	opt := NewAllowedCodesOption(1, 2, 3)
	if len(opt.AllowedCodes) != 3 {
		t.Errorf("len(AllowedCodes) = %d, want 3", len(opt.AllowedCodes))
	}
	if opt.AllowedCodes[0] != 1 || opt.AllowedCodes[1] != 2 || opt.AllowedCodes[2] != 3 {
		t.Errorf("AllowedCodes = %v, want [1, 2, 3]", opt.AllowedCodes)
	}
}

func TestNewCommandByString(t *testing.T) {
	cmd := NewCommandByString("ffmpeg", "-i input.mp4 output.mp4")
	if cmd.Command != "ffmpeg" {
		t.Errorf("Command = %q, want %q", cmd.Command, "ffmpeg")
	}
	if len(cmd.Params) != 3 {
		t.Errorf("len(Params) = %d, want 3", len(cmd.Params))
	}
}

func TestCommand_AddParam(t *testing.T) {
	cmd := NewCommand("ffmpeg")
	result := cmd.AddParam("-i")

	if len(cmd.Params) != 1 {
		t.Errorf("len(Params) = %d, want 1", len(cmd.Params))
	}
	if cmd.Params[0] != "-i" {
		t.Errorf("Params[0] = %q, want %q", cmd.Params[0], "-i")
	}
	if result != cmd {
		t.Error("AddParam should return the same command pointer")
	}
}

func TestCommand_SetWorkDir(t *testing.T) {
	cmd := NewCommand("ffmpeg")
	result := cmd.SetWorkDir("/tmp")

	if cmd.WorkDir != "/tmp" {
		t.Errorf("WorkDir = %q, want %q", cmd.WorkDir, "/tmp")
	}
	if result != cmd {
		t.Error("SetWorkDir should return the same command pointer")
	}
}

func TestCommand_SetEnv(t *testing.T) {
	cmd := NewCommand("ffmpeg")
	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	result := cmd.SetEnv(env)

	if len(cmd.Env) != 2 {
		t.Errorf("len(Env) = %d, want 2", len(cmd.Env))
	}
	if result != cmd {
		t.Error("SetEnv should return the same command pointer")
	}
}

func TestCommand_AddEnv(t *testing.T) {
	cmd := NewCommand("ffmpeg")
	originalLen := len(cmd.Env)
	result := cmd.AddEnv("MY_VAR=value")

	if len(cmd.Env) != originalLen+1 {
		t.Errorf("len(Env) = %d, want %d", len(cmd.Env), originalLen+1)
	}
	if result != cmd {
		t.Error("AddEnv should return the same command pointer")
	}
}

func TestCommand_SetStdoutFunc(t *testing.T) {
	cmd := NewCommand("ffmpeg")
	fn := func(buffer []byte, exit bool) {}
	result := cmd.SetStdoutFunc(fn)

	if cmd.StdoutFunc == nil {
		t.Error("StdoutFunc should not be nil")
	}
	if result != cmd {
		t.Error("SetStdoutFunc should return the same command pointer")
	}
}

func TestCommand_SetStderrFunc(t *testing.T) {
	cmd := NewCommand("ffmpeg")
	fn := func(buffer []byte, exit bool) {}
	result := cmd.SetStderrFunc(fn)

	if cmd.SterrFunc == nil {
		t.Error("SterrFunc should not be nil")
	}
	if result != cmd {
		t.Error("SetStderrFunc should return the same command pointer")
	}
}

func TestCommand_Run_Success(t *testing.T) {
	cmd := NewCommand("echo", "hello")
	exitCode, err := cmd.Run()

	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("Run() exitCode = %d, want 0", exitCode)
	}
}

func TestCommand_Run_CommandNotFound(t *testing.T) {
	cmd := NewCommand("nonexistent_command_12345")
	exitCode, err := cmd.Run()

	if err == nil {
		t.Error("Run() should return error for nonexistent command")
	}
	if exitCode != 127 {
		t.Errorf("Run() exitCode = %d, want 127 (command not found)", exitCode)
	}
}

func TestCommand_RunWithAllowedCodes(t *testing.T) {
	cmd := NewCommand("sh", "-c", "exit 1")
	exitCode, err := cmd.Run(NewAllowedCodesOption(1))

	if err != nil {
		t.Errorf("Run() with allowed code 1 should not return error, got %v", err)
	}
	if exitCode != 1 {
		t.Errorf("Run() exitCode = %d, want 1", exitCode)
	}
}

func TestCommand_RunWithNonAllowedCode(t *testing.T) {
	cmd := NewCommand("sh", "-c", "exit 1")
	exitCode, err := cmd.Run()

	if err == nil {
		t.Error("Run() without allowed codes should return error for exit 1")
	}
	if exitCode != 1 {
		t.Errorf("Run() exitCode = %d, want 1", exitCode)
	}
}

func TestIsPanicOpt(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected bool
	}{
		{
			name:     "no options",
			opts:     []Option{},
			expected: false,
		},
		{
			name:     "panic option",
			opts:     []Option{NewPanicOption()},
			expected: true,
		},
		{
			name:     "allowed codes only",
			opts:     []Option{NewAllowedCodesOption(1)},
			expected: false,
		},
		{
			name:     "both options",
			opts:     []Option{NewAllowedCodesOption(1), NewPanicOption()},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPanicOpt(tt.opts)
			if result != tt.expected {
				t.Errorf("isPanicOpt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAllowedCodes(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		exitCode int
		expected bool
	}{
		{
			name:     "no options, exit 0",
			opts:     []Option{},
			exitCode: 0,
			expected: true,
		},
		{
			name:     "no options, exit 1",
			opts:     []Option{},
			exitCode: 1,
			expected: false,
		},
		{
			name:     "allowed code 1, exit 1",
			opts:     []Option{NewAllowedCodesOption(1)},
			exitCode: 1,
			expected: true,
		},
		{
			name:     "allowed code 1, exit 2",
			opts:     []Option{NewAllowedCodesOption(1)},
			exitCode: 2,
			expected: false,
		},
		{
			name:     "allowed codes 1,2,3, exit 2",
			opts:     []Option{NewAllowedCodesOption(1, 2, 3)},
			exitCode: 2,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowedCodes(tt.opts, tt.exitCode)
			if result != tt.expected {
				t.Errorf("allowedCodes() = %v, want %v", result, tt.expected)
			}
		})
	}
}
