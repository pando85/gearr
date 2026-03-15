package task

import (
	"context"
	"testing"
)

func TestEncodeWorker_IsTypeAccepted(t *testing.T) {
	worker := &EncodeWorker{}

	tests := []struct {
		jobType  string
		expected bool
	}{
		{"encode", true},
		{"pgstosrt", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.jobType, func(t *testing.T) {
			result := worker.IsTypeAccepted(tt.jobType)
			if result != tt.expected {
				t.Errorf("IsTypeAccepted(%q) = %v, want %v", tt.jobType, result, tt.expected)
			}
		})
	}
}

func TestEncodeWorker_GetID(t *testing.T) {
	worker := &EncodeWorker{name: "test-worker-456"}
	if worker.GetID() != "test-worker-456" {
		t.Errorf("GetID() = %q, want %q", worker.GetID(), "test-worker-456")
	}
}

func TestEncodeWorker_PrefetchJobs(t *testing.T) {
	worker := &EncodeWorker{
		prefetchJobs: 5,
	}

	result := worker.PrefetchJobs()
	if result != 5 {
		t.Errorf("PrefetchJobs() = %d, want 5", result)
	}
}

func TestNewEncodeWorker(t *testing.T) {
	ctx := context.Background()
	config := Config{
		TemporalPath:    "/tmp/test-worker",
		MaxPrefetchJobs: 5,
		EncodeJobs:      2,
	}

	worker := NewEncodeWorker(ctx, config, "test-worker", NewConsoleWorkerPrinter())

	if worker == nil {
		t.Fatal("NewEncodeWorker() returned nil")
	}

	if worker.name != "test-worker" {
		t.Errorf("worker.name = %q, want %q", worker.name, "test-worker")
	}

	if worker.maxPrefetchJobs != 5 {
		t.Errorf("worker.maxPrefetchJobs = %d, want 5", worker.maxPrefetchJobs)
	}

	if cap(worker.downloadChan) != 100 {
		t.Errorf("cap(downloadChan) = %d, want 100", cap(worker.downloadChan))
	}

	if cap(worker.encodeChan) != 100 {
		t.Errorf("cap(encodeChan) = %d, want 100", cap(worker.encodeChan))
	}

	if cap(worker.uploadChan) != 100 {
		t.Errorf("cap(uploadChan) = %d, want 100", cap(worker.uploadChan))
	}
}

func TestEncodeWorker_Cancel(t *testing.T) {
	ctx := context.Background()
	config := Config{
		TemporalPath: "/tmp/test-worker",
	}

	worker := NewEncodeWorker(ctx, config, "test-worker", NewConsoleWorkerPrinter())

	select {
	case <-worker.ctx.Done():
		t.Error("context should not be done before Cancel()")
	default:
	}

	worker.Cancel()

	select {
	case <-worker.ctx.Done():
	default:
		t.Error("context should be done after Cancel()")
	}
}

func TestFFMPEGProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress FFMPEGProgress
		duration int
		speed    float64
		percent  float64
	}{
		{
			name:     "normal progress",
			progress: FFMPEGProgress{duration: 60, speed: 1.5, percent: 50.0},
			duration: 60,
			speed:    1.5,
			percent:  50.0,
		},
		{
			name:     "zero values",
			progress: FFMPEGProgress{duration: 0, speed: 0, percent: 0},
			duration: 0,
			speed:    0,
			percent:  0,
		},
		{
			name:     "high speed",
			progress: FFMPEGProgress{duration: 120, speed: 10.5, percent: 100.0},
			duration: 120,
			speed:    10.5,
			percent:  100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.progress.duration != tt.duration {
				t.Errorf("duration = %d, want %d", tt.progress.duration, tt.duration)
			}
			if tt.progress.speed != tt.speed {
				t.Errorf("speed = %f, want %f", tt.progress.speed, tt.speed)
			}
			if tt.progress.percent != tt.percent {
				t.Errorf("percent = %f, want %f", tt.progress.percent, tt.percent)
			}
		})
	}
}
