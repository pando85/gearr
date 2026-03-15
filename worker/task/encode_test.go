package task

import "testing"

func TestDurToSec(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"00:00:00", 0},
		{"00:00:01", 1},
		{"00:01:00", 60},
		{"01:00:00", 3600},
		{"01:30:45", 5445},
		{"12:34:56", 45296},
		{"", 0},
		{"invalid", 0},
		{"00:00", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := durToSec(tt.input)
			if result != tt.expected {
				t.Errorf("durToSec(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetSpeed(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"speed=1.0x", 1.0},
		{"speed=2.5x", 2.5},
		{"speed=0.5x", 0.5},
		{"speed=100.123x", 100.123},
		{"frame=  123 fps= 30 speed=1.5x", 1.5},
		{"no speed here", -1},
		{"speed=", -1},
		{"", -1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getSpeed(tt.input)
			if result != tt.expected {
				t.Errorf("getSpeed(%q) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"frame=123 time=00:00:00 bitrate=1000", 0},
		{"frame=456 time=00:00:01 speed=1.5x", 1},
		{"frame=789 time=00:01:30 bitrate=2000", 90},
		{"frame=000 time=01:00:00 speed=2.0x", 3600},
		{"frame=123 time=00:05:30 bitrate=1000", 330},
		{"no time here", -1},
		{"time=", -1},
		{"time=00:00:00", -1},
		{"", -1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getDuration(tt.input)
			if result != tt.expected {
				t.Errorf("getDuration(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFFProbeFrameRate(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		expectError bool
	}{
		{"24/1", 24, false},
		{"30/1", 30, false},
		{"30000/1001", 29, false},
		{"24000/1001", 23, false},
		{"60/1", 60, false},
		{"25/1", 25, false},
		{"", 0, true},
		{"24", 0, true},
		{"24/0", 0, true},
		{"abc/def", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := FFProbeFrameRate(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("FFProbeFrameRate(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("FFProbeFrameRate(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("FFProbeFrameRate(%q) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}
