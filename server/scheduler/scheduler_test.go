package scheduler

import (
	"context"
	"gearr/model"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSendUpdateJobsNotification_NonBlocking(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	notification := &model.JobUpdateNotification{
		Id:        uuid.New(),
		Status:    model.QueuedNotificationStatus,
		EventTime: time.Now(),
	}

	start := time.Now()
	rs.sendUpdateJobsNotification(notification)
	elapsed := time.Since(start)

	if elapsed > notificationSendTimeout*2 {
		t.Errorf("sendUpdateJobsNotification took too long with no consumers: %v", elapsed)
	}
}

func TestSendUpdateJobsNotification_DropsOnBlockedChannel(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	id := uuid.New()
	sub := &jobSubscription{
		notifyChan: make(chan *model.JobUpdateNotification, 0),
		closed:     make(chan struct{}),
	}
	rs.updateJobsChannels[id] = sub

	notification := &model.JobUpdateNotification{
		Id:        uuid.New(),
		Status:    model.QueuedNotificationStatus,
		EventTime: time.Now(),
	}

	done := make(chan bool)
	go func() {
		rs.sendUpdateJobsNotification(notification)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(notificationSendTimeout * 2):
		t.Error("sendUpdateJobsNotification blocked on unbuffered channel with no consumer")
	}
}

func TestSendUpdateJobsNotification_MultipleChannels(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	numChannels := 5
	var wg sync.WaitGroup

	for i := 0; i < numChannels; i++ {
		id, ch := rs.GetUpdateJobsChan(context.Background())
		wg.Add(1)
		go func(channelID uuid.UUID, channel chan *model.JobUpdateNotification) {
			defer wg.Done()
			select {
			case <-channel:
			case <-time.After(2 * time.Second):
				t.Errorf("channel %s did not receive notification", channelID)
			}
		}(id, ch)
	}

	notification := &model.JobUpdateNotification{
		Id:        uuid.New(),
		Status:    model.QueuedNotificationStatus,
		EventTime: time.Now(),
	}

	rs.sendUpdateJobsNotification(notification)
	wg.Wait()
}

func TestGetUpdateJobsChan_BufferedChannel(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	id, ch := rs.GetUpdateJobsChan(context.Background())

	if ch == nil {
		t.Error("GetUpdateJobsChan returned nil channel")
	}

	rs.jobChannelsMutex.Lock()
	_, exists := rs.updateJobsChannels[id]
	rs.jobChannelsMutex.Unlock()

	if !exists {
		t.Error("channel not registered in updateJobsChannels map")
	}
}

func TestCloseUpdateJobsChan_ClosesChannel(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	id, ch := rs.GetUpdateJobsChan(context.Background())

	rs.CloseUpdateJobsChan(id)

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed")
		}
	default:
		select {
		case _, ok := <-ch:
			if ok {
				t.Error("channel should be closed after CloseUpdateJobsChan")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("channel should be closed immediately")
		}
	}

	rs.jobChannelsMutex.Lock()
	_, exists := rs.updateJobsChannels[id]
	rs.jobChannelsMutex.Unlock()

	if exists {
		t.Error("channel should be removed from map after CloseUpdateJobsChan")
	}
}

func TestCloseUpdateJobsChan_Idempotent(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	id, _ := rs.GetUpdateJobsChan(context.Background())

	rs.CloseUpdateJobsChan(id)

	rs.CloseUpdateJobsChan(id)
}

func TestSendUpdateJobsNotification_ConcurrentSafety(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var sendCount int64
	var closeCount int64

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					id, ch := rs.GetUpdateJobsChan(ctx)
					go func(ch chan *model.JobUpdateNotification) {
						for {
							select {
							case <-ch:
							case <-ctx.Done():
								return
							}
						}
					}(ch)
					time.Sleep(20 * time.Millisecond)
					rs.CloseUpdateJobsChan(id)
					atomic.AddInt64(&closeCount, 1)
				}
			}
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					notification := &model.JobUpdateNotification{
						Id:        uuid.New(),
						Status:    model.QueuedNotificationStatus,
						EventTime: time.Now(),
					}
					rs.sendUpdateJobsNotification(notification)
					atomic.AddInt64(&sendCount, 1)
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("completed %d sends and %d closes", atomic.LoadInt64(&sendCount), atomic.LoadInt64(&closeCount))
	case <-time.After(5 * time.Second):
		t.Error("concurrent test timed out, potential deadlock")
	}
}

func TestSendUpdateJobsNotification_SlowConsumer(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	fastChanID, fastChan := rs.GetUpdateJobsChan(context.Background())
	slowChanID, slowChan := rs.GetUpdateJobsChan(context.Background())

	var slowWg sync.WaitGroup
	slowWg.Add(1)
	go func() {
		defer slowWg.Done()
		time.Sleep(notificationSendTimeout + 100*time.Millisecond)
		<-slowChan
	}()

	notification := &model.JobUpdateNotification{
		Id:        uuid.New(),
		Status:    model.QueuedNotificationStatus,
		EventTime: time.Now(),
	}

	done := make(chan bool, 1)
	go func() {
		rs.sendUpdateJobsNotification(notification)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(notificationSendTimeout * 3):
		t.Error("sendUpdateJobsNotification blocked due to slow consumer")
	}

	select {
	case <-fastChan:
	default:
		t.Error("fast channel should have received notification")
	}

	slowWg.Wait()
	rs.CloseUpdateJobsChan(fastChanID)
	rs.CloseUpdateJobsChan(slowChanID)
}

func TestSendUpdateJobsNotification_ChannelBufferFull(t *testing.T) {
	rs := &RuntimeScheduler{
		updateJobsChannels: make(map[uuid.UUID]*jobSubscription),
	}

	id, ch := rs.GetUpdateJobsChan(context.Background())

	for i := 0; i < 100; i++ {
		ch <- &model.JobUpdateNotification{
			Id:        uuid.New(),
			Status:    model.QueuedNotificationStatus,
			EventTime: time.Now(),
		}
	}

	notification := &model.JobUpdateNotification{
		Id:        uuid.New(),
		Status:    model.QueuedNotificationStatus,
		EventTime: time.Now(),
	}

	start := time.Now()
	rs.sendUpdateJobsNotification(notification)
	elapsed := time.Since(start)

	if elapsed > notificationSendTimeout*2 {
		t.Errorf("sendUpdateJobsNotification blocked too long on full channel: %v", elapsed)
	}

	rs.CloseUpdateJobsChan(id)
}
