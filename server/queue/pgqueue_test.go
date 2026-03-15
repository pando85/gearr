package queue

import (
	"gearr/model"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPostgresBrokerServer_Channels(t *testing.T) {
	newTask := make(chan *model.ControlEvent, 100)
	newWorkerEvent := make(chan *model.JobEventQueue, 100)

	if cap(newTask) != 100 {
		t.Errorf("newTask channel capacity = %d, want 100", cap(newTask))
	}

	if cap(newWorkerEvent) != 100 {
		t.Errorf("newWorkerEvent channel capacity = %d, want 100", cap(newWorkerEvent))
	}
}

func TestPostgresBrokerServer_ReceiveJobEvent(t *testing.T) {
	taskEventConsumers := []chan *model.TaskEvent{}

	tc := make(chan *model.TaskEvent, 100)
	taskEventConsumers = append(taskEventConsumers, tc)

	tc2 := make(chan *model.TaskEvent, 100)
	taskEventConsumers = append(taskEventConsumers, tc2)

	if len(taskEventConsumers) != 2 {
		t.Errorf("taskEventConsumers count = %d, want 2", len(taskEventConsumers))
	}

	for i, consumer := range taskEventConsumers {
		if cap(consumer) != 100 {
			t.Errorf("consumer %d channel capacity = %d, want 100", i, cap(consumer))
		}
	}
}

func TestPostgresBrokerServer_TaskPublisher(t *testing.T) {
	newTask := make(chan *model.ControlEvent, 100)

	task := &model.TaskEncode{
		Id:          uuid.New(),
		DownloadURL: "http://example.com/video.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     1,
	}

	controlChan := make(chan interface{})
	newTask <- &model.ControlEvent{
		Event:       task,
		ControlChan: controlChan,
	}

	select {
	case event := <-newTask:
		if event.Event.Id != task.Id {
			t.Errorf("event.Event.Id = %v, want %v", event.Event.Id, task.Id)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for task on newTask channel")
	}
}

func TestPostgresBrokerServer_WorkerEvent(t *testing.T) {
	newWorkerEvent := make(chan *model.JobEventQueue, 100)

	jobEvent := &model.JobEvent{
		Id:     uuid.New(),
		Action: "cancel",
	}

	jobEventQueue := &model.JobEventQueue{
		Queue:    "worker-queue",
		JobEvent: jobEvent,
	}

	newWorkerEvent <- jobEventQueue

	select {
	case event := <-newWorkerEvent:
		if event.Queue != "worker-queue" {
			t.Errorf("event.Queue = %q, want %q", event.Queue, "worker-queue")
		}
		if event.JobEvent.Action != "cancel" {
			t.Errorf("event.JobEvent.Action = %q, want %q", event.JobEvent.Action, "cancel")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event on newWorkerEvent channel")
	}
}

func TestPostgresBrokerServer_TaskEventProcessor(t *testing.T) {
	taskEventConsumers := []chan *model.TaskEvent{}

	tc := make(chan *model.TaskEvent, 100)
	taskEventConsumers = append(taskEventConsumers, tc)

	event := &model.TaskEvent{
		Id:               uuid.New(),
		EventID:          1,
		EventType:        model.NotificationEvent,
		NotificationType: model.JobNotification,
		Status:           model.ProgressingNotificationStatus,
	}

	for _, consumer := range taskEventConsumers {
		select {
		case consumer <- event:
		default:
		}
	}

	select {
	case received := <-tc:
		if received.Id != event.Id {
			t.Errorf("received.Id = %v, want %v", received.Id, event.Id)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event on consumer channel")
	}
}
