package task

import (
	"testing"
	"time"
)

func TestNewConsoleWorkerPrinter(t *testing.T) {
	printer := NewConsoleWorkerPrinter()

	if printer == nil {
		t.Fatal("NewConsoleWorkerPrinter returned nil")
	}

	if printer.pw == nil {
		t.Error("printer.pw is nil")
	}
}

func TestConsoleWorkerPrinter_AddTask(t *testing.T) {
	printer := NewConsoleWorkerPrinter()

	tests := []struct {
		name     string
		id       string
		stepType JobStepType
	}{
		{
			name:     "download task",
			id:       "job-123",
			stepType: DownloadJobStepType,
		},
		{
			name:     "upload task",
			id:       "job-456",
			stepType: UploadJobStepType,
		},
		{
			name:     "encode task",
			id:       "job-789",
			stepType: EncodeJobStepType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := printer.AddTask(tt.id, tt.stepType)

			if task == nil {
				t.Error("AddTask returned nil")
			}
			if task.id != tt.id {
				t.Errorf("task.id = %q, want %q", task.id, tt.id)
			}
			if task.stepType != tt.stepType {
				t.Errorf("task.stepType = %q, want %q", task.stepType, tt.stepType)
			}
			if task.progressTracker == nil {
				t.Error("task.progressTracker is nil")
			}
			if task.printer == nil {
				t.Error("task.printer is nil")
			}
		})
	}
}

func TestConsoleWorkerPrinter_Log(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	printer.Log("test message: %s", "value")
}

func TestConsoleWorkerPrinter_Warn(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	printer.Warn("warning message: %s", "value")
}

func TestConsoleWorkerPrinter_Cmd(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	printer.Cmd("command message: %s", "value")
}

func TestConsoleWorkerPrinter_Error(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	printer.Error("error message: %s", "value")
}

func TestTaskTracks_SetTotal(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.SetTotal(1000)

	if task.progressTracker.Total != 1000 {
		t.Errorf("progressTracker.Total = %d, want 1000", task.progressTracker.Total)
	}
}

func TestTaskTracks_ETA(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	eta := task.ETA()

	if eta < 0 {
		t.Errorf("ETA() = %v, want >= 0", eta)
	}
}

func TestTaskTracks_PercentDone(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.SetTotal(100)
	task.progressTracker.SetValue(50)

	percent := task.PercentDone()

	if percent < 0 || percent > 100 {
		t.Errorf("PercentDone() = %v, want 0-100", percent)
	}
}

func TestTaskTracks_UpdateValue(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.SetTotal(100)
	task.UpdateValue(75)

	if task.progressTracker.Value() != 75 {
		t.Errorf("progressTracker.Value() = %d, want 75", task.progressTracker.Value())
	}
}

func TestTaskTracks_Increment64(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.SetTotal(100)
	task.UpdateValue(50)
	task.Increment64(10)

	if task.progressTracker.Value() != 60 {
		t.Errorf("progressTracker.Value() = %d, want 60", task.progressTracker.Value())
	}
}

func TestTaskTracks_Increment(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.SetTotal(100)
	task.UpdateValue(50)
	task.Increment(5)

	if task.progressTracker.Value() != 55 {
		t.Errorf("progressTracker.Value() = %d, want 55", task.progressTracker.Value())
	}
}

func TestTaskTracks_Message(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.Message("processing frame 100")
}

func TestTaskTracks_ResetMessage(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.Message("processing frame 100")
	task.ResetMessage()
}

func TestTaskTracks_Done(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.SetTotal(100)
	task.UpdateValue(50)
	task.Done()

	if !task.progressTracker.IsDone() {
		t.Error("progressTracker should be done")
	}
}

func TestTaskTracks_Error(t *testing.T) {
	printer := NewConsoleWorkerPrinter()
	task := printer.AddTask("job-123", DownloadJobStepType)

	task.Error()

	if !task.progressTracker.IsErrored() {
		t.Error("progressTracker should be errored")
	}
}

func TestJobStepType_Constants(t *testing.T) {
	if DownloadJobStepType != "download" {
		t.Errorf("DownloadJobStepType = %q, want %q", DownloadJobStepType, "download")
	}
	if UploadJobStepType != "upload" {
		t.Errorf("UploadJobStepType = %q, want %q", UploadJobStepType, "upload")
	}
	if EncodeJobStepType != "encode" {
		t.Errorf("EncodeJobStepType = %q, want %q", EncodeJobStepType, "encode")
	}
}

func TestConsoleWorkerPrinter_Render(t *testing.T) {
	printer := NewConsoleWorkerPrinter()

	go printer.Render()

	time.Sleep(100 * time.Millisecond)
}

func TestConsoleWorkerPrinter_MultipleTasks(t *testing.T) {
	printer := NewConsoleWorkerPrinter()

	task1 := printer.AddTask("job-1", DownloadJobStepType)
	task2 := printer.AddTask("job-1", UploadJobStepType)
	task3 := printer.AddTask("job-1", EncodeJobStepType)

	task1.SetTotal(1000)
	task2.SetTotal(2000)
	task3.SetTotal(3000)

	if task1.progressTracker.Total != 1000 {
		t.Errorf("task1.Total = %d, want 1000", task1.progressTracker.Total)
	}
	if task2.progressTracker.Total != 2000 {
		t.Errorf("task2.Total = %d, want 2000", task2.progressTracker.Total)
	}
	if task3.progressTracker.Total != 3000 {
		t.Errorf("task3.Total = %d, want 3000", task3.progressTracker.Total)
	}
}
