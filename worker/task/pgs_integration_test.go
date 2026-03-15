package task

import (
	"context"
	"gearr/model"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type mockManager struct {
	responsePGSJobCalled bool
	lastResponse         *model.TaskPGSResponse
}

func (m *mockManager) EventNotification(event model.TaskEvent) error {
	return nil
}

func (m *mockManager) ResponsePGSJob(response model.TaskPGSResponse) error {
	m.responsePGSJobCalled = true
	m.lastResponse = &response
	return nil
}

func (m *mockManager) RequestPGSJob(pgsJob model.TaskPGS) <-chan *model.TaskPGSResponse {
	ch := make(chan *model.TaskPGSResponse, 1)
	go func() {
		ch <- &model.TaskPGSResponse{
			Id:    pgsJob.Id,
			PGSID: pgsJob.PGSID,
			Srt:   []byte("1\n00:00:00,000 --> 00:00:01,000\nTest subtitle\n"),
			Err:   "",
			Queue: pgsJob.ReplyTo,
		}
		close(ch)
	}()
	return ch
}

func TestPGSWorker_Prepare(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pgs-worker-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		TemporalPath: tempDir,
	}

	worker := NewPGSWorker(ctx, config, "test-worker")

	task := model.TaskPGS{
		Id:          model.TaskEncode{}.Id,
		PGSID:       1,
		PGSdata:     []byte("test pgs data"),
		PGSLanguage: "eng",
		ReplyTo:     "test-queue",
	}

	taskData, err := jsonMarshal(task)
	if err != nil {
		t.Fatalf("failed to marshal task: %v", err)
	}

	manager := &mockManager{}
	err = worker.Prepare(taskData, manager)
	if err != nil {
		t.Errorf("Prepare() error = %v, want nil", err)
	}

	if worker.task.PGSID != task.PGSID {
		t.Errorf("Prepare() task.PGSID = %d, want %d", worker.task.PGSID, task.PGSID)
	}

	if worker.task.PGSLanguage != task.PGSLanguage {
		t.Errorf("Prepare() task.PGSLanguage = %s, want %s", worker.task.PGSLanguage, task.PGSLanguage)
	}
}

func TestPGSWorker_IsTypeAccepted(t *testing.T) {
	worker := &PGSWorker{}

	tests := []struct {
		jobType  string
		expected bool
	}{
		{"pgstosrt", true},
		{"encode", false},
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

func TestPGSWorker_GetID(t *testing.T) {
	worker := &PGSWorker{name: "test-worker-123"}
	if worker.GetID() != "test-worker-123" {
		t.Errorf("GetID() = %q, want %q", worker.GetID(), "test-worker-123")
	}
}

func TestPGSWorker_AcceptJobs(t *testing.T) {
	worker := &PGSWorker{}
	if !worker.AcceptJobs() {
		t.Error("AcceptJobs() = false, want true")
	}
}

func TestPGSWorker_Clean(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pgs-worker-clean-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	config := Config{
		TemporalPath: tempDir,
	}

	worker := NewPGSWorker(ctx, config, "test-worker")

	workerTempPath := filepath.Join(tempDir, "worker-test-worker")
	if err := os.MkdirAll(workerTempPath, os.ModePerm); err != nil {
		t.Fatalf("failed to create worker temp path: %v", err)
	}

	testFile := filepath.Join(workerTempPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), os.ModePerm); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("test file was not created")
	}

	err = worker.Clean()
	if err != nil {
		t.Errorf("Clean() error = %v, want nil", err)
	}

	if _, err := os.Stat(workerTempPath); !os.IsNotExist(err) {
		t.Errorf("Clean() did not remove temp path %s", workerTempPath)
	}
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return []byte(`{"id":"00000000-0000-0000-0000-000000000001","pgsid":1,"pgsdata":"dGVzdCBwZ3MgZGF0YQ==","pgslanguage":"eng","replyto":"test-queue"}`), nil
}

func TestPGSWorker_PrepareInvalidJSON(t *testing.T) {
	worker := &PGSWorker{}
	manager := &mockManager{}

	err := worker.Prepare([]byte("invalid json"), manager)
	if err == nil {
		t.Error("Prepare() with invalid JSON should return error")
	}
}

func init() {
	io.Discard.Write([]byte{})
}
