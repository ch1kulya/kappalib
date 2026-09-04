package data

import (
	"context"
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/models"
)

func TestGetGlobalAnnouncement_NilDB(t *testing.T) {
	cache.C.Delete("global_announcement:random")

	ann, err := GetGlobalAnnouncement(context.Background())
	if err != nil {
		t.Fatalf("expected nil error when DB is nil, got %v", err)
	}
	if ann != nil {
		t.Fatalf("expected nil announcement when DB is nil, got %+v", ann)
	}
}

func TestGetGlobalAnnouncement_FromCache(t *testing.T) {
	expected := &models.GlobalAnnouncement{
		ID:   1,
		Text: "Test announcement",
	}
	cache.C.Set("global_announcement:random", expected, time.Minute)
	defer cache.C.Delete("global_announcement:random")

	ann, err := GetGlobalAnnouncement(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ann == nil || ann.ID != expected.ID || ann.Text != expected.Text {
		t.Fatalf("expected cached announcement %+v, got %+v", expected, ann)
	}
}
