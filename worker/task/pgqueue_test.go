package task

import (
	"testing"

	"gearr/model"

	"github.com/google/uuid"
)

func TestNewPGSJobControl(t *testing.T) {
	taskID := uuid.New()
	task := model.TaskPGS{
		Id:          taskID,
		PGSID:       1,
		PGSdata:     []byte("test pgs data"),
		PGSLanguage: "eng",
		ReplyTo:     "test-queue",
	}

	control := NewPGSJobControl(task)

	if control == nil {
		t.Fatal("NewPGSJobControl returned nil")
	}

	if control.task.Id != task.Id {
		t.Errorf("task.Id = %v, want %v", control.task.Id, task.Id)
	}

	if control.task.PGSID != task.PGSID {
		t.Errorf("task.PGSID = %d, want %d", control.task.PGSID, task.PGSID)
	}

	if control.task.PGSLanguage != task.PGSLanguage {
		t.Errorf("task.PGSLanguage = %q, want %q", control.task.PGSLanguage, task.PGSLanguage)
	}

	if control.response == nil {
		t.Error("response channel is nil")
	}

	if cap(control.response) != 1 {
		t.Errorf("response channel capacity = %d, want 1", cap(control.response))
	}
}

func TestTaskPGSJobControl_Response(t *testing.T) {
	task := model.TaskPGS{
		Id:          uuid.New(),
		PGSID:       1,
		PGSdata:     []byte("test"),
		PGSLanguage: "eng",
	}

	control := NewPGSJobControl(task)

	response := &model.TaskPGSResponse{
		Id:    task.Id,
		PGSID: task.PGSID,
		Srt:   []byte("1\n00:00:00,000 --> 00:00:01,000\nTest\n"),
		Err:   "",
		Queue: "reply-queue",
	}

	go func() {
		control.response <- response
	}()

	received := <-control.response

	if received.Id != response.Id {
		t.Errorf("received.Id = %v, want %v", received.Id, response.Id)
	}

	if received.PGSID != response.PGSID {
		t.Errorf("received.PGSID = %d, want %d", received.PGSID, response.PGSID)
	}

	if string(received.Srt) != string(response.Srt) {
		t.Errorf("received.Srt = %q, want %q", string(received.Srt), string(response.Srt))
	}
}

func TestJobWorker_Active(t *testing.T) {
	jobID := uuid.New()
	worker := &JobWorker{
		jobID:  jobID,
		active: false,
	}

	if worker.active {
		t.Error("worker should not be active initially")
	}

	worker.active = true

	if !worker.active {
		t.Error("worker should be active after setting")
	}
}
