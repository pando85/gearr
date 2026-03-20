package model

import "testing"

func TestJobPriorityIsValid(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     bool
	}{
		{"low is valid", JobPriorityLow, true},
		{"normal is valid", JobPriorityNormal, true},
		{"high is valid", JobPriorityHigh, true},
		{"urgent is valid", JobPriorityUrgent, true},
		{"negative is invalid", -1, false},
		{"above range is invalid", 4, false},
		{"far above is invalid", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JobPriorityIsValid(tt.priority); got != tt.want {
				t.Errorf("JobPriorityIsValid(%d) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestJobPriorityString(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     string
	}{
		{"low", JobPriorityLow, "low"},
		{"normal", JobPriorityNormal, "normal"},
		{"high", JobPriorityHigh, "high"},
		{"urgent", JobPriorityUrgent, "urgent"},
		{"unknown", 99, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JobPriorityString(tt.priority); got != tt.want {
				t.Errorf("JobPriorityString(%d) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

func TestJobPriorityFromString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"low", "low", JobPriorityLow},
		{"normal", "normal", JobPriorityNormal},
		{"high", "high", JobPriorityHigh},
		{"urgent", "urgent", JobPriorityUrgent},
		{"unknown defaults to normal", "unknown", JobPriorityNormal},
		{"empty defaults to normal", "", JobPriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JobPriorityFromString(tt.s); got != tt.want {
				t.Errorf("JobPriorityFromString(%s) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestJobPriorityValidate(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     int
	}{
		{"low passes through", JobPriorityLow, JobPriorityLow},
		{"normal passes through", JobPriorityNormal, JobPriorityNormal},
		{"high passes through", JobPriorityHigh, JobPriorityHigh},
		{"urgent passes through", JobPriorityUrgent, JobPriorityUrgent},
		{"invalid defaults to normal", -1, JobPriorityNormal},
		{"above range defaults to normal", 99, JobPriorityNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JobPriorityValidate(tt.priority); got != tt.want {
				t.Errorf("JobPriorityValidate(%d) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}
