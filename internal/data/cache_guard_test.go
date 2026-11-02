package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
)

func TestGetNovels_OutOfRangePagesStayOutOfCache(t *testing.T) {
	originalDB := database.DB
	database.DB = nil
	t.Cleanup(func() {
		database.DB = originalDB
		cache.C.Delete(novelsCountCacheKey)
	})

	tests := []struct {
		name           string
		count          int
		page           int
		wantTotalPages int
	}{
		{name: "first page past the end", count: 24, page: 3, wantTotalPages: 2},
		{name: "far past the end", count: 24, page: 9999, wantTotalPages: 2},
		{name: "largest page does not overflow", count: 24, page: math.MaxInt, wantTotalPages: 2},
		{name: "empty table", count: 0, page: 1, wantTotalPages: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.C.Set(novelsCountCacheKey, tt.count, time.Minute)

			got, err := GetNovels(context.Background(), tt.page, "popular")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Novels) != 0 || got.TotalCount != tt.count || got.TotalPages != tt.wantTotalPages || got.Page != tt.page {
				t.Errorf("got %+v", got)
			}
			if _, found := cache.C.Get(fmt.Sprintf("novels:page:%d:sort:popular", tt.page)); found {
				t.Error("out-of-range page must not create a cache entry")
			}
		})
	}
}

func TestNormalizeNovelsSort(t *testing.T) {
	tests := map[string]string{
		"newest":       "newest",
		"popular":      "popular",
		"":             "popular",
		"attacker-key": "popular",
	}

	for input, want := range tests {
		if got := normalizeNovelsSort(input); got != want {
			t.Errorf("normalizeNovelsSort(%q) = %q, want %q", input, got, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func githubPRPage(t *testing.T, count int) string {
	t.Helper()
	prs := make([]map[string]any, count)
	for i := range prs {
		prs[i] = map[string]any{
			"number":    i + 1,
			"body":      "!update: change",
			"merged_at": "2026-01-01T00:00:00Z",
		}
	}
	body, err := json.Marshal(prs)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestGetAppUpdates_FailedPaginationIsNotCached(t *testing.T) {
	const key = "github:prs:0"
	failureKey := key + ":failure"
	clear := func() {
		cache.C.Delete(key)
		cache.C.Delete(failureKey)
	}
	clear()
	t.Cleanup(clear)

	original := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = original })

	fullPage := githubPRPage(t, 100)
	var requests atomic.Int32
	var failing atomic.Bool
	failing.Store(true)

	http.DefaultClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		status, body := http.StatusOK, "[]"
		switch {
		case page == 1:
			body = fullPage
		case failing.Load():
			status = http.StatusBadGateway
		}
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})

	if _, err := GetAppUpdates(context.Background(), 0); err == nil {
		t.Fatal("expected error when a later page fails")
	}
	if _, found := cache.C.Get(key); found {
		t.Fatal("partial updates must not be cached")
	}

	seen := requests.Load()
	if _, err := GetAppUpdates(context.Background(), 0); err == nil {
		t.Fatal("expected cooldown to keep returning the error")
	}
	if requests.Load() != seen {
		t.Fatal("cooldown must not call GitHub again")
	}

	failing.Store(false)
	cache.C.Delete(failureKey)

	updates, err := GetAppUpdates(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}
	if len(updates) != 100 {
		t.Fatalf("got %d updates, want 100", len(updates))
	}
	if _, found := cache.C.Get(key); !found {
		t.Fatal("complete updates must be cached")
	}
}

func TestLookupsRejectUnqueryableIDs(t *testing.T) {
	originalDB := database.DB
	database.DB = nil
	t.Cleanup(func() { database.DB = originalDB })

	for _, id := range []string{"nvl_\x00", "nvl_\xff"} {
		if _, err := GetNovel(context.Background(), id); !errors.Is(err, ErrNovelNotFound) {
			t.Errorf("GetNovel(%q) error = %v, want ErrNovelNotFound", id, err)
		}
		if _, err := GetChapter(context.Background(), id); !errors.Is(err, ErrChapterNotFound) {
			t.Errorf("GetChapter(%q) error = %v, want ErrChapterNotFound", id, err)
		}
	}
}

func TestGetChapters_ReturnsCopyOfCachedList(t *testing.T) {
	const novelID = "nvl_copytest"
	key := "chapters:" + novelID
	cache.C.Set(key, &models.ChaptersList{
		NovelID:  novelID,
		Chapters: []models.ChapterSummary{{ID: "first"}, {ID: "second"}},
		Count:    2,
	}, time.Minute)
	t.Cleanup(func() { cache.C.Delete(key) })

	first, err := GetChapters(context.Background(), novelID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	first.Chapters[0], first.Chapters[1] = first.Chapters[1], first.Chapters[0]

	second, err := GetChapters(context.Background(), novelID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.Chapters[0].ID != "first" {
		t.Errorf("cached order changed: first chapter = %q", second.Chapters[0].ID)
	}
}
