package data

import (
	"testing"
	"time"
)

func TestOfflineRateLimiter_Burst(t *testing.T) {
	limiter := newOfflineRateLimiter()
	now := time.Now()

	for i := range offlineBurst {
		if !limiter.allow("usr_1", now) {
			t.Fatalf("request %d within burst was rejected", i+1)
		}
	}
	if limiter.allow("usr_1", now) {
		t.Error("request beyond burst was allowed")
	}
	if !limiter.allow("usr_2", now) {
		t.Error("other user was rejected by shared bucket")
	}
	if !limiter.allow("usr_1", now.Add(time.Second)) {
		t.Error("request after refill interval was rejected")
	}
}

func TestOfflineRateLimiter_Cleanup(t *testing.T) {
	limiter := newOfflineRateLimiter()
	now := time.Now()

	limiter.allow("usr_stale", now)
	limiter.allow("usr_fresh", now.Add(offlineVisitorExpiry))
	limiter.allow("usr_fresh", now.Add(offlineVisitorExpiry+time.Minute))

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if _, exists := limiter.visitors["usr_stale"]; exists {
		t.Error("stale visitor was not cleaned up")
	}
	if _, exists := limiter.visitors["usr_fresh"]; !exists {
		t.Error("active visitor was cleaned up")
	}
}
