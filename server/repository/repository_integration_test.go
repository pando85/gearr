package repository

import (
	"context"
	"gearr/model"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnqueueDequeueEncodeJob(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	task := &model.TaskEncode{
		Id:          jobID,
		DownloadURL: "http://example.com/video.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     1,
	}

	err := repo.EnqueueEncodeJob(ctx, task)
	if err != nil {
		t.Fatalf("EnqueueEncodeJob failed: %v", err)
	}

	dequeued, err := repo.DequeueEncodeJob(ctx, "test-worker")
	if err != nil {
		t.Fatalf("DequeueEncodeJob failed: %v", err)
	}

	if dequeued == nil {
		t.Fatal("Expected to dequeue a job, got nil")
	}

	if dequeued.Id != jobID {
		t.Errorf("Job ID mismatch: got %v, want %v", dequeued.Id, jobID)
	}

	if dequeued.DownloadURL != task.DownloadURL {
		t.Errorf("DownloadURL mismatch: got %v, want %v", dequeued.DownloadURL, task.DownloadURL)
	}
}

func TestDequeueEncodeJobEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	dequeued, err := repo.DequeueEncodeJob(ctx, "test-worker")
	if err != nil {
		t.Fatalf("DequeueEncodeJob failed: %v", err)
	}

	if dequeued != nil {
		t.Errorf("Expected nil for empty queue, got %+v", dequeued)
	}
}

func TestEnqueueDequeuePGSJob(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	pgs := &model.TaskPGS{
		Id:          jobID,
		PGSID:       1,
		PGSdata:     []byte("test pgs data"),
		PGSLanguage: "eng",
		ReplyTo:     "test-reply-queue",
	}

	err := repo.EnqueuePGSJob(ctx, pgs)
	if err != nil {
		t.Fatalf("EnqueuePGSJob failed: %v", err)
	}

	dequeued, err := repo.DequeuePGSJob(ctx, "test-worker")
	if err != nil {
		t.Fatalf("DequeuePGSJob failed: %v", err)
	}

	if dequeued == nil {
		t.Fatal("Expected to dequeue a PGS job, got nil")
	}

	if dequeued.Id != jobID {
		t.Errorf("Job ID mismatch: got %v, want %v", dequeued.Id, jobID)
	}

	if dequeued.PGSID != pgs.PGSID {
		t.Errorf("PGSID mismatch: got %v, want %v", dequeued.PGSID, pgs.PGSID)
	}

	if string(dequeued.PGSdata) != string(pgs.PGSdata) {
		t.Errorf("PGSdata mismatch: got %v, want %v", string(dequeued.PGSdata), string(pgs.PGSdata))
	}
}

func TestEnqueueDequeuePGSResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	resp := &model.TaskPGSResponse{
		Id:    jobID,
		PGSID: 1,
		Srt:   []byte("1\n00:00:00,000 --> 00:00:01,000\nTest\n"),
		Err:   "",
		Queue: "test-reply-queue",
	}

	err := repo.EnqueuePGSResponse(ctx, resp)
	if err != nil {
		t.Fatalf("EnqueuePGSResponse failed: %v", err)
	}

	dequeued, err := repo.DequeuePGSResponse(ctx, "test-reply-queue")
	if err != nil {
		t.Fatalf("DequeuePGSResponse failed: %v", err)
	}

	if dequeued == nil {
		t.Fatal("Expected to dequeue a PGS response, got nil")
	}

	if dequeued.Id != jobID {
		t.Errorf("Job ID mismatch: got %v, want %v", dequeued.Id, jobID)
	}

	if string(dequeued.Srt) != string(resp.Srt) {
		t.Errorf("Srt mismatch: got %v, want %v", string(dequeued.Srt), string(resp.Srt))
	}
}

func TestDequeuePGSResponseWrongQueue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	resp := &model.TaskPGSResponse{
		Id:    jobID,
		PGSID: 1,
		Srt:   []byte("test"),
		Queue: "original-queue",
	}

	err := repo.EnqueuePGSResponse(ctx, resp)
	if err != nil {
		t.Fatalf("EnqueuePGSResponse failed: %v", err)
	}

	dequeued, err := repo.DequeuePGSResponse(ctx, "different-queue")
	if err != nil {
		t.Fatalf("DequeuePGSResponse failed: %v", err)
	}

	if dequeued != nil {
		t.Errorf("Expected nil for wrong queue, got %+v", dequeued)
	}
}

func TestEnqueueDequeueTaskEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	event := &model.TaskEvent{
		Id:               jobID,
		EventID:          1,
		EventType:        model.NotificationEvent,
		WorkerName:       "test-worker",
		WorkerQueue:      "test-queue",
		EventTime:        time.Now(),
		NotificationType: model.JobNotification,
		Status:           model.QueuedNotificationStatus,
		Message:          "test message",
	}

	err := repo.EnqueueTaskEvent(ctx, event)
	if err != nil {
		t.Fatalf("EnqueueTaskEvent failed: %v", err)
	}

	events, err := repo.DequeueTaskEvents(ctx, 10)
	if err != nil {
		t.Fatalf("DequeueTaskEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Id != jobID {
		t.Errorf("Job ID mismatch: got %v, want %v", events[0].Id, jobID)
	}

	if events[0].Status != event.Status {
		t.Errorf("Status mismatch: got %v, want %v", events[0].Status, event.Status)
	}
}

func TestDequeueTaskEventsMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		jobID := uuid.New()
		event := &model.TaskEvent{
			Id:               jobID,
			EventID:          1,
			EventType:        model.NotificationEvent,
			WorkerName:       "test-worker",
			WorkerQueue:      "test-queue",
			EventTime:        time.Now(),
			NotificationType: model.JobNotification,
			Status:           model.QueuedNotificationStatus,
			Message:          "test message",
		}
		err := repo.EnqueueTaskEvent(ctx, event)
		if err != nil {
			t.Fatalf("EnqueueTaskEvent failed: %v", err)
		}
	}

	events, err := repo.DequeueTaskEvents(ctx, 3)
	if err != nil {
		t.Fatalf("DequeueTaskEvents failed: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	remaining, err := repo.DequeueTaskEvents(ctx, 10)
	if err != nil {
		t.Fatalf("DequeueTaskEvents failed: %v", err)
	}

	if len(remaining) != 2 {
		t.Errorf("Expected 2 remaining events, got %d", len(remaining))
	}
}

func TestEnqueueDequeueJobAction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()

	err := repo.EnqueueJobAction(ctx, jobID.String(), "test-worker", model.JobAction("cancel"))
	if err != nil {
		t.Fatalf("EnqueueJobAction failed: %v", err)
	}

	actions, err := repo.DequeueJobActions(ctx, "test-worker")
	if err != nil {
		t.Fatalf("DequeueJobActions failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	if actions[0].Id != jobID {
		t.Errorf("Job ID mismatch: got %v, want %v", actions[0].Id, jobID)
	}

	if actions[0].Action != "cancel" {
		t.Errorf("Action mismatch: got %v, want %v", actions[0].Action, "cancel")
	}
}

func TestDequeueJobActionsWrongWorker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()

	err := repo.EnqueueJobAction(ctx, jobID.String(), "worker-1", model.JobAction("cancel"))
	if err != nil {
		t.Fatalf("EnqueueJobAction failed: %v", err)
	}

	actions, err := repo.DequeueJobActions(ctx, "worker-2")
	if err != nil {
		t.Fatalf("DequeueJobActions failed: %v", err)
	}

	if len(actions) != 0 {
		t.Errorf("Expected 0 actions for wrong worker, got %d", len(actions))
	}
}

func TestEncodeJobSkipLocked(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	task := &model.TaskEncode{
		Id:          jobID,
		DownloadURL: "http://example.com/video.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     1,
	}

	err := repo.EnqueueEncodeJob(ctx, task)
	if err != nil {
		t.Fatalf("EnqueueEncodeJob failed: %v", err)
	}

	dequeued1, err := repo.DequeueEncodeJob(ctx, "worker-1")
	if err != nil {
		t.Fatalf("First DequeueEncodeJob failed: %v", err)
	}

	if dequeued1 == nil {
		t.Fatal("First dequeue should return the job")
	}

	dequeued2, err := repo.DequeueEncodeJob(ctx, "worker-2")
	if err != nil {
		t.Fatalf("Second DequeueEncodeJob failed: %v", err)
	}

	if dequeued2 != nil {
		t.Error("Second dequeue should return nil - job already locked")
	}
}

func TestEncodeJobFIFO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	for i, jobID := range jobIDs {
		task := &model.TaskEncode{
			Id:          jobID,
			DownloadURL: "http://example.com/video.mp4",
			UploadURL:   "http://example.com/upload",
			ChecksumURL: "http://example.com/checksum",
			EventID:     i + 1,
		}
		err := repo.EnqueueEncodeJob(ctx, task)
		if err != nil {
			t.Fatalf("EnqueueEncodeJob failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	for i, expectedID := range jobIDs {
		dequeued, err := repo.DequeueEncodeJob(ctx, "test-worker")
		if err != nil {
			t.Fatalf("DequeueEncodeJob %d failed: %v", i, err)
		}

		if dequeued == nil {
			t.Fatalf("Expected job %d, got nil", i)
		}

		if dequeued.Id != expectedID {
			t.Errorf("Job %d ID mismatch: got %v, want %v", i, dequeued.Id, expectedID)
		}
	}
}

func TestUpdateJobPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID := uuid.New()
	job := &model.Job{
		Id:              jobID,
		SourcePath:      "/test/source.mp4",
		DestinationPath: "/test/dest.mp4",
		Priority:        0,
	}

	err := repo.AddJob(ctx, job)
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	err = repo.UpdateJobPriority(ctx, jobID.String(), 10)
	if err != nil {
		t.Fatalf("UpdateJobPriority failed: %v", err)
	}

	updatedJob, err := repo.GetJob(ctx, jobID.String())
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if updatedJob.Priority != 10 {
		t.Errorf("Priority mismatch: got %v, want 10", updatedJob.Priority)
	}
}

func TestUpdateJobPriorityNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	nonExistentID := uuid.New()
	err := repo.UpdateJobPriority(ctx, nonExistentID.String(), 10)
	if err == nil {
		t.Error("Expected error for non-existent job, got nil")
	}
}

func TestDequeueEncodeJobWithPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobIDLow := uuid.New()
	jobIDHigh := uuid.New()
	jobIDMedium := uuid.New()

	db := repo.GetDB()
	db.ExecContext(ctx, "INSERT INTO jobs (id, source_path, destination_path, priority) VALUES ($1, '/test/low.mp4', '/test/low-out.mp4', 1)", jobIDLow.String())
	db.ExecContext(ctx, "INSERT INTO jobs (id, source_path, destination_path, priority) VALUES ($1, '/test/high.mp4', '/test/high-out.mp4', 10)", jobIDHigh.String())
	db.ExecContext(ctx, "INSERT INTO jobs (id, source_path, destination_path, priority) VALUES ($1, '/test/medium.mp4', '/test/medium-out.mp4', 5)", jobIDMedium.String())

	taskLow := &model.TaskEncode{
		Id:          jobIDLow,
		DownloadURL: "http://example.com/low.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     1,
	}
	taskHigh := &model.TaskEncode{
		Id:          jobIDHigh,
		DownloadURL: "http://example.com/high.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     2,
	}
	taskMedium := &model.TaskEncode{
		Id:          jobIDMedium,
		DownloadURL: "http://example.com/medium.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     3,
	}

	repo.EnqueueEncodeJob(ctx, taskLow)
	time.Sleep(10 * time.Millisecond)
	repo.EnqueueEncodeJob(ctx, taskMedium)
	time.Sleep(10 * time.Millisecond)
	repo.EnqueueEncodeJob(ctx, taskHigh)

	dequeued, err := repo.DequeueEncodeJob(ctx, "test-worker")
	if err != nil {
		t.Fatalf("DequeueEncodeJob failed: %v", err)
	}

	if dequeued == nil {
		t.Fatal("Expected to dequeue a job, got nil")
	}

	if dequeued.Id != jobIDHigh {
		t.Errorf("Expected high priority job %v, got %v", jobIDHigh, dequeued.Id)
	}
}

func TestDequeueEncodeJobPriorityThenFIFO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	jobID1 := uuid.New()
	jobID2 := uuid.New()

	db := repo.GetDB()
	db.ExecContext(ctx, "INSERT INTO jobs (id, source_path, destination_path, priority) VALUES ($1, '/test/1.mp4', '/test/1-out.mp4', 5)", jobID1.String())
	db.ExecContext(ctx, "INSERT INTO jobs (id, source_path, destination_path, priority) VALUES ($1, '/test/2.mp4', '/test/2-out.mp4', 5)", jobID2.String())

	task1 := &model.TaskEncode{
		Id:          jobID1,
		DownloadURL: "http://example.com/1.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     1,
	}
	task2 := &model.TaskEncode{
		Id:          jobID2,
		DownloadURL: "http://example.com/2.mp4",
		UploadURL:   "http://example.com/upload",
		ChecksumURL: "http://example.com/checksum",
		EventID:     2,
	}

	repo.EnqueueEncodeJob(ctx, task1)
	time.Sleep(10 * time.Millisecond)
	repo.EnqueueEncodeJob(ctx, task2)

	dequeued, err := repo.DequeueEncodeJob(ctx, "test-worker")
	if err != nil {
		t.Fatalf("DequeueEncodeJob failed: %v", err)
	}

	if dequeued == nil {
		t.Fatal("Expected to dequeue a job, got nil")
	}

	if dequeued.Id != jobID1 {
		t.Errorf("Expected first job %v (same priority, FIFO), got %v", jobID1, dequeued.Id)
	}
}

func setupTestDB(t *testing.T) (*SQLRepository, func()) {
	config := SQLServerConfig{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		Database: getEnvOrDefault("TEST_DB_NAME", "gearr_test"),
		Driver:   "pgx",
		SSLMode:  "disable",
	}

	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	ctx := context.Background()
	err = repo.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	cleanup := func() {
		ctx := context.Background()
		db := repo.GetDB()
		db.ExecContext(ctx, "DELETE FROM encode_queue")
		db.ExecContext(ctx, "DELETE FROM pgs_queue")
		db.ExecContext(ctx, "DELETE FROM pgs_responses")
		db.ExecContext(ctx, "DELETE FROM task_event_queue")
		db.ExecContext(ctx, "DELETE FROM job_actions")
		db.ExecContext(ctx, "DELETE FROM job_events")
		db.ExecContext(ctx, "DELETE FROM jobs")
		db.ExecContext(ctx, "DELETE FROM workers")
		repo.GetDB().Close()
	}

	return repo, cleanup
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := getEnv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnv(key string) string {
	return os.Getenv(key)
}
