package data

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestUserTimeTracker_RecordTime(t *testing.T) {
	tracker := NewUserTimeTracker()

	tracker.RecordTime("usr_1", 30)
	tracker.RecordTime("usr_2", 15)

	tracker.RecordTime("", 30)
	tracker.RecordTime("usr_1", -5)
	tracker.RecordTime("usr_1", 0)
	tracker.RecordTime("usr_1", 500)

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if tracker.pending["usr_1"] != 30 {
		t.Errorf("expected usr_1 to have 30 seconds, got %d", tracker.pending["usr_1"])
	}

	if tracker.pending["usr_2"] != 15 {
		t.Errorf("expected usr_2 to have 15 seconds, got %d", tracker.pending["usr_2"])
	}
}

func TestUserTimeTracker_TokenBucketMultiTabSpam(t *testing.T) {
	tracker := NewUserTimeTracker()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.RecordTime("usr_multitab", 60)
		}()
	}
	wg.Wait()

	tracker.mu.Lock()
	total := tracker.pending["usr_multitab"]
	tracker.mu.Unlock()

	if total > 75 {
		t.Errorf("expected total granted to be capped by token bucket (<=75), got %d", total)
	}
}

func TestUserTimeTracker_FlushPruning(t *testing.T) {
	tracker := NewUserTimeTracker()

	tracker.mu.Lock()
	tracker.buckets["usr_old"] = &timeBucket{
		tokens:     10,
		lastUpdate: time.Now().Add(-15 * time.Minute),
	}
	tracker.buckets["usr_recent"] = &timeBucket{
		tokens:     10,
		lastUpdate: time.Now(),
	}
	tracker.mu.Unlock()

	_ = tracker.Flush(context.Background())

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if _, ok := tracker.buckets["usr_old"]; ok {
		t.Errorf("expected usr_old to be pruned")
	}
	if _, ok := tracker.buckets["usr_recent"]; !ok {
		t.Errorf("expected usr_recent to be preserved")
	}
}

func TestUserTimeTracker_FlushEmpty(t *testing.T) {
	tracker := NewUserTimeTracker()
	err := tracker.Flush(context.Background())
	if err != nil {
		t.Errorf("expected nil error on empty flush, got %v", err)
	}
}

func TestUserTimeTracker_BatchRestoredOnDBError(t *testing.T) {
	tracker := NewUserTimeTracker()

	tracker.RecordTime("usr_retry", 30)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tracker.Flush(cancelledCtx)
	if err == nil {
		t.Errorf("expected error with cancelled context")
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if tracker.pending["usr_retry"] != 30 {
		t.Errorf("expected usr_retry to retain 30 seconds after failed flush, got %d", tracker.pending["usr_retry"])
	}
}

func TestUserTimeTracker_LifecycleStop(t *testing.T) {
	tracker := NewUserTimeTracker()

	tracker.RecordTime("usr_stop", 20)
	tracker.Start(50 * time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	flushCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = tracker.Stop(flushCtx)

	tracker.mu.Lock()
	stopCh := tracker.stopCh
	tracker.mu.Unlock()

	if stopCh != nil {
		t.Errorf("expected stopCh to be nil after Stop")
	}
}
