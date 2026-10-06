package data

import (
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
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

func TestPaginateOfflineChapters(t *testing.T) {
	chapters := func(nums ...int) []models.Chapter {
		result := make([]models.Chapter, 0, len(nums))
		for _, num := range nums {
			result = append(result, models.Chapter{ChapterNum: num})
		}
		return result
	}

	tests := []struct {
		name      string
		chapters  []models.Chapter
		limit     int
		wantCount int
		wantNext  *int
	}{
		{name: "empty page", chapters: chapters(), limit: 2, wantCount: 0},
		{name: "fewer than limit", chapters: chapters(1), limit: 2, wantCount: 1},
		{name: "exactly limit", chapters: chapters(1, 2), limit: 2, wantCount: 2},
		{name: "one extra row", chapters: chapters(1, 2, 5), limit: 2, wantCount: 2, wantNext: new(2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := paginateOfflineChapters("nvl_test", tt.chapters, tt.limit)
			if page.NovelID != "nvl_test" {
				t.Errorf("NovelID = %q, want %q", page.NovelID, "nvl_test")
			}
			if len(page.Chapters) != tt.wantCount {
				t.Fatalf("got %d chapters, want %d", len(page.Chapters), tt.wantCount)
			}
			switch {
			case tt.wantNext == nil && page.NextAfter != nil:
				t.Errorf("NextAfter = %d, want nil", *page.NextAfter)
			case tt.wantNext != nil && page.NextAfter == nil:
				t.Errorf("NextAfter = nil, want %d", *tt.wantNext)
			case tt.wantNext != nil && *page.NextAfter != *tt.wantNext:
				t.Errorf("NextAfter = %d, want %d", *page.NextAfter, *tt.wantNext)
			}
		})
	}
}
