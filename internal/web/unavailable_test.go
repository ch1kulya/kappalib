package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func useUnreachableDB(t *testing.T) {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://user:pass@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	original := database.DB
	database.DB = pool
	t.Cleanup(func() {
		database.DB = original
		pool.Close()
	})
}

func clearCacheKeys(t *testing.T, keys ...string) {
	t.Helper()
	remove := func() {
		for _, key := range keys {
			cache.C.Delete(key)
		}
	}
	remove()
	t.Cleanup(remove)
}

func TestPagesReportDatabaseFailure(t *testing.T) {
	useUnreachableDB(t)
	clearCacheKeys(t,
		"novel:nvl_down", "novel:nvl_cached", "chapters:nvl_cached", "chapter:chp_down",
		"catalog:tags", "catalog:stats", "updates:latest:0", "novels:recent:0",
		"novels:count", "novels:page:1:sort:popular", "github:prs:0", "github:prs:0:failure",
		"global_announcement:random",
	)
	cache.C.Set("novel:nvl_cached", &models.Novel{ID: "nvl_cached", Title: "Cached"}, time.Minute)

	h := NewHandler("test")
	router := chi.NewRouter()
	router.Get("/", h.Home)
	router.Get("/catalog", h.Catalog)
	router.Get("/updates", h.Updates)
	router.Get("/{id}", h.Novel)
	router.Get("/{id}/chapter/{chapterId}", h.Chapter)

	tests := []struct {
		name       string
		path       string
		absentText string
	}{
		{name: "home", path: "/"},
		{name: "catalog", path: "/catalog", absentText: "Ничего не найдено"},
		{name: "updates", path: "/updates"},
		{name: "unknown state novel is not 404", path: "/nvl_down", absentText: "Новелла не найдена"},
		{name: "novel chapters", path: "/nvl_cached", absentText: "Глав пока нет"},
		{name: "chapter", path: "/nvl_cached/chapter/chp_down", absentText: "Глава не найдена"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", got)
			}
			body := rec.Body.String()
			if !strings.Contains(body, "Сервис временно недоступен") {
				t.Error("expected unavailable message in body")
			}
			if tt.absentText != "" && strings.Contains(body, tt.absentText) {
				t.Errorf("body must not contain %q", tt.absentText)
			}
			if strings.Contains(body, "127.0.0.1") {
				t.Error("body must not leak connection details")
			}
		})
	}
}
