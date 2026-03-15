package task

import (
	"testing"

	"gearr/model"
)

func TestAcceptedJobs_IsAccepted(t *testing.T) {
	tests := []struct {
		name     string
		jobs     AcceptedJobs
		jobType  model.JobType
		expected bool
	}{
		{
			name:     "empty accepted jobs",
			jobs:     AcceptedJobs{},
			jobType:  model.EncodeJobType,
			expected: false,
		},
		{
			name:     "encode job accepted",
			jobs:     AcceptedJobs{model.EncodeJobType},
			jobType:  model.EncodeJobType,
			expected: true,
		},
		{
			name:     "pgs job accepted",
			jobs:     AcceptedJobs{model.PGSToSrtJobType},
			jobType:  model.PGSToSrtJobType,
			expected: true,
		},
		{
			name:     "both jobs accepted - encode",
			jobs:     AcceptedJobs{model.EncodeJobType, model.PGSToSrtJobType},
			jobType:  model.EncodeJobType,
			expected: true,
		},
		{
			name:     "both jobs accepted - pgs",
			jobs:     AcceptedJobs{model.EncodeJobType, model.PGSToSrtJobType},
			jobType:  model.PGSToSrtJobType,
			expected: true,
		},
		{
			name:     "job not in accepted list",
			jobs:     AcceptedJobs{model.EncodeJobType},
			jobType:  model.PGSToSrtJobType,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.jobs.IsAccepted(tt.jobType)
			if result != tt.expected {
				t.Errorf("IsAccepted(%q) = %v, want %v", tt.jobType, result, tt.expected)
			}
		})
	}
}

func TestTimeHourMinute_String(t *testing.T) {
	tests := []struct {
		name     string
		time     TimeHourMinute
		expected string
	}{
		{
			name:     "zero time",
			time:     TimeHourMinute{Hour: 0, Minute: 0},
			expected: "00:00",
		},
		{
			name:     "morning time",
			time:     TimeHourMinute{Hour: 9, Minute: 30},
			expected: "09:30",
		},
		{
			name:     "afternoon time",
			time:     TimeHourMinute{Hour: 14, Minute: 45},
			expected: "14:45",
		},
		{
			name:     "late night",
			time:     TimeHourMinute{Hour: 23, Minute: 59},
			expected: "23:59",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.time.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTimeHourMinute_Set(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    TimeHourMinute
		expectError bool
	}{
		{
			name:     "valid time",
			input:    "09:30",
			expected: TimeHourMinute{Hour: 9, Minute: 30},
		},
		{
			name:     "valid time zero",
			input:    "00:00",
			expected: TimeHourMinute{Hour: 0, Minute: 0},
		},
		{
			name:     "valid time late",
			input:    "23:59",
			expected: TimeHourMinute{Hour: 23, Minute: 59},
		},
		{
			name:        "invalid format - no colon",
			input:       "0930",
			expectError: true,
		},
		{
			name:        "invalid format - too many colons",
			input:       "09:30:00",
			expectError: true,
		},
		{
			name:        "invalid format - not a number",
			input:       "ab:cd",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var time TimeHourMinute
			err := time.Set(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Set(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Set(%q) unexpected error: %v", tt.input, err)
				}
				if time.Hour != tt.expected.Hour || time.Minute != tt.expected.Minute {
					t.Errorf("Set(%q) = {Hour: %d, Minute: %d}, want {Hour: %d, Minute: %d}",
						tt.input, time.Hour, time.Minute, tt.expected.Hour, tt.expected.Minute)
				}
			}
		})
	}
}

func TestConfig_HaveSetPeriodTime(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{
			name:     "no period set",
			config:   Config{StartAfter: TimeHourMinute{}, StopAfter: TimeHourMinute{}},
			expected: false,
		},
		{
			name:     "start after set",
			config:   Config{StartAfter: TimeHourMinute{Hour: 9, Minute: 0}, StopAfter: TimeHourMinute{}},
			expected: true,
		},
		{
			name:     "stop after set",
			config:   Config{StartAfter: TimeHourMinute{}, StopAfter: TimeHourMinute{Hour: 17, Minute: 0}},
			expected: true,
		},
		{
			name:     "both set",
			config:   Config{StartAfter: TimeHourMinute{Hour: 9, Minute: 0}, StopAfter: TimeHourMinute{Hour: 17, Minute: 0}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HaveSetPeriodTime()
			if result != tt.expected {
				t.Errorf("HaveSetPeriodTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTimeHourMinute_Type(t *testing.T) {
	time := TimeHourMinute{Hour: 9, Minute: 30}
	result := time.Type()

	if result != "TimeHourMinute" {
		t.Errorf("Type() = %q, want %q", result, "TimeHourMinute")
	}
}

func TestConfig_Fields(t *testing.T) {
	config := Config{
		UpdateMode:        true,
		TemporalPath:      "/tmp/gearr",
		Name:              "test-worker",
		Threads:           4,
		MaxPrefetchJobs:   10,
		Jobs:              AcceptedJobs{model.EncodeJobType, model.PGSToSrtJobType},
		EncodeJobs:        2,
		PgsJobs:           2,
		StartAfter:        TimeHourMinute{Hour: 9, Minute: 0},
		StopAfter:         TimeHourMinute{Hour: 17, Minute: 0},
		Paused:            false,
		PGSTOSrtDLLPath:   "/usr/lib/pgs",
		TesseractDataPath: "/usr/share/tessdata",
		DotnetPath:        "/usr/bin/dotnet",
	}

	if config.Name != "test-worker" {
		t.Errorf("config.Name = %q, want %q", config.Name, "test-worker")
	}
	if config.Threads != 4 {
		t.Errorf("config.Threads = %d, want 4", config.Threads)
	}
	if config.MaxPrefetchJobs != 10 {
		t.Errorf("config.MaxPrefetchJobs = %d, want 10", config.MaxPrefetchJobs)
	}
	if len(config.Jobs) != 2 {
		t.Errorf("len(config.Jobs) = %d, want 2", len(config.Jobs))
	}
	if !config.Jobs.IsAccepted(model.EncodeJobType) {
		t.Error("config.Jobs should accept EncodeJobType")
	}
	if !config.Jobs.IsAccepted(model.PGSToSrtJobType) {
		t.Error("config.Jobs should accept PGSToSrtJobType")
	}
	if config.EncodeJobs != 2 {
		t.Errorf("config.EncodeJobs = %d, want 2", config.EncodeJobs)
	}
	if config.PgsJobs != 2 {
		t.Errorf("config.PgsJobs = %d, want 2", config.PgsJobs)
	}
	if config.Paused {
		t.Error("config.Paused should be false")
	}
	if config.TemporalPath != "/tmp/gearr" {
		t.Errorf("config.TemporalPath = %q, want %q", config.TemporalPath, "/tmp/gearr")
	}
}
