package data

import (
	"context"
	_ "embed"
	"errors"
	"sync"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/logger"
)

//go:embed sql/user_daily_time_upsert_batch.sql
var queryUserDailyTimeUpsertBatch string

type timeBucket struct {
	tokens     float64
	lastUpdate time.Time
}

type UserTimeTracker struct {
	mu      sync.Mutex
	pending map[string]int
	buckets map[string]*timeBucket
	stopCh  chan struct{}
	doneCh  chan struct{}
}

var TimeTracker = NewUserTimeTracker()

func NewUserTimeTracker() *UserTimeTracker {
	return &UserTimeTracker{
		pending: make(map[string]int),
		buckets: make(map[string]*timeBucket),
	}
}

func (t *UserTimeTracker) RecordTime(userID string, seconds int) {
	if userID == "" || seconds <= 0 || seconds > 300 {
		return
	}

	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	b, exists := t.buckets[userID]
	if !exists {
		initialTokens := min(float64(seconds), 75.0)
		b = &timeBucket{
			tokens:     initialTokens,
			lastUpdate: now,
		}
		t.buckets[userID] = b
	} else {
		elapsed := now.Sub(b.lastUpdate).Seconds()
		b.tokens += elapsed
		if b.tokens > 90.0 {
			b.tokens = 90.0
		}
		b.lastUpdate = now
	}

	toGrant := min(seconds, int(b.tokens))
	if toGrant <= 0 {
		return
	}

	b.tokens -= float64(toGrant)
	t.pending[userID] += toGrant
}

func (t *UserTimeTracker) Flush(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	t.mu.Lock()
	now := time.Now()
	for uid, b := range t.buckets {
		if now.Sub(b.lastUpdate) > 10*time.Minute {
			delete(t.buckets, uid)
		}
	}

	if len(t.pending) == 0 {
		t.mu.Unlock()
		return nil
	}

	batch := t.pending
	t.pending = make(map[string]int)
	t.mu.Unlock()

	userIDs := make([]string, 0, len(batch))
	secondsList := make([]int32, 0, len(batch))

	for uid, sec := range batch {
		if sec > 0 {
			userIDs = append(userIDs, uid)
			secondsList = append(secondsList, int32(sec))
		}
	}

	if len(userIDs) == 0 {
		return nil
	}

	if database.DB == nil {
		t.mu.Lock()
		for uid, sec := range batch {
			t.pending[uid] += sec
		}
		t.mu.Unlock()
		return errors.New("database not initialized")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := database.DB.Exec(dbCtx, queryUserDailyTimeUpsertBatch, userIDs, secondsList)
	if err != nil {
		t.mu.Lock()
		for uid, sec := range batch {
			t.pending[uid] += sec
		}
		t.mu.Unlock()
		logger.Error("Failed to flush user daily time batch: %v", err)
		return err
	}

	return nil
}

func (t *UserTimeTracker) Start(interval time.Duration) {
	t.mu.Lock()
	stopCh := make(chan struct{})
	t.stopCh = stopCh
	doneCh := make(chan struct{})
	t.doneCh = doneCh
	t.mu.Unlock()

	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = t.Flush(context.Background())
			case <-stopCh:
				return
			}
		}
	}()
}

func (t *UserTimeTracker) Stop(ctx context.Context) error {
	t.mu.Lock()
	if t.stopCh != nil {
		close(t.stopCh)
		t.stopCh = nil
	}
	done := t.doneCh
	t.mu.Unlock()

	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	return t.Flush(ctx)
}
