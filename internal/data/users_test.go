package data

import (
	"encoding/json"
	"testing"

	"github.com/ch1kulya/kappalib/internal/models"
)

func TestMergeProgressCookies_DeleteNovel(t *testing.T) {
	existing := models.CookieValue{
		Value:     `{"novels":{"nvl_12345678":{"chapterId":"chp_12345678","chapterNum":10,"readAt":1000},"nvl_87654321":{"chapterId":"chp_87654321","chapterNum":5,"readAt":2000}},"lastRead":{"novelId":"nvl_12345678","title":"Test Novel","author":"Author","coverUrl":"","chapterId":"chp_12345678","chapterNum":10,"totalChapters":15,"readAt":1000}}`,
		UpdatedAt: 1000,
	}

	incoming := models.CookieValue{
		Value:     `{"novels":{"nvl_12345678":{"chapterId":"chp_12345678","chapterNum":10,"readAt":1000}},"lastRead":{"novelId":"nvl_12345678","title":"Test Novel","author":"Author","coverUrl":"","chapterId":"chp_12345678","chapterNum":10,"totalChapters":15,"readAt":1000}}`,
		UpdatedAt: 2000,
	}

	result := mergeProgressCookies(existing, incoming)

	var schema progressCookieSchema
	if err := json.Unmarshal([]byte(result.Value), &schema); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := schema.Novels["nvl_87654321"]; ok {
		t.Error("deleted novel should not be in merged result")
	}

	if len(schema.Novels) != 1 {
		t.Errorf("expected 1 novel, got %d", len(schema.Novels))
	}
}

func TestMergeProgressCookies_KeepDeletionWhenServerIsNewer(t *testing.T) {
	existing := models.CookieValue{
		Value:     `{"novels":{"nvl_12345678":{"chapterId":"chp_12345678","chapterNum":10,"readAt":1000}},"lastRead":null}`,
		UpdatedAt: 2000,
	}

	incoming := models.CookieValue{
		Value:     `{"novels":{"nvl_12345678":{"chapterId":"chp_12345678","chapterNum":10,"readAt":1000},"nvl_87654321":{"chapterId":"chp_87654321","chapterNum":5,"readAt":500}},"lastRead":{"novelId":"nvl_87654321","title":"Old Novel","author":"Author","coverUrl":"","chapterId":"chp_87654321","chapterNum":5,"totalChapters":10,"readAt":500}}`,
		UpdatedAt: 1000,
	}

	result := mergeProgressCookies(existing, incoming)

	var schema progressCookieSchema
	if err := json.Unmarshal([]byte(result.Value), &schema); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := schema.Novels["nvl_87654321"]; ok {
		t.Error("deleted novel should not be added back when existing is newer")
	}

	if len(schema.Novels) != 1 {
		t.Errorf("expected 1 novel, got %d", len(schema.Novels))
	}
}

func TestMergeProgressCookies_MergeChapterProgress(t *testing.T) {
	existing := models.CookieValue{
		Value:     `{"novels":{"nvl_12345678":{"chapterId":"chp_12345678","chapterNum":5,"readAt":1000}},"lastRead":{"novelId":"nvl_12345678","title":"Test Novel","author":"Author","coverUrl":"","chapterId":"chp_12345678","chapterNum":5,"totalChapters":15,"readAt":1000}}`,
		UpdatedAt: 1000,
	}

	incoming := models.CookieValue{
		Value:     `{"novels":{"nvl_12345678":{"chapterId":"chp_87654321","chapterNum":10,"readAt":2000}},"lastRead":{"novelId":"nvl_12345678","title":"Test Novel","author":"Author","coverUrl":"","chapterId":"chp_87654321","chapterNum":10,"totalChapters":15,"readAt":2000}}`,
		UpdatedAt: 500,
	}

	result := mergeProgressCookies(existing, incoming)

	var schema progressCookieSchema
	if err := json.Unmarshal([]byte(result.Value), &schema); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	novel := schema.Novels["nvl_12345678"]
	if novel.ChapterNum != 10 {
		t.Errorf("expected chapterNum 10, got %d", novel.ChapterNum)
	}

	if schema.LastRead.ChapterNum != 10 {
		t.Errorf("expected lastRead chapterNum 10, got %d", schema.LastRead.ChapterNum)
	}
}

func TestMergeProgressCookies_LastReadByReadAt(t *testing.T) {
	existing := models.CookieValue{
		Value:     `{"novels":{"nvl_old":{"chapterId":"chp_old","chapterNum":5,"readAt":1000}},"lastRead":{"novelId":"nvl_old","title":"Old Novel","author":"Author","coverUrl":"","chapterId":"chp_old","chapterNum":5,"totalChapters":10,"readAt":1000}}`,
		UpdatedAt: 1000,
	}

	incoming := models.CookieValue{
		Value:     `{"novels":{"nvl_new":{"chapterId":"chp_new","chapterNum":3,"readAt":2000}},"lastRead":{"novelId":"nvl_new","title":"New Novel","author":"Author","coverUrl":"","chapterId":"chp_new","chapterNum":3,"totalChapters":10,"readAt":2000}}`,
		UpdatedAt: 2000,
	}

	result := mergeProgressCookies(existing, incoming)

	var schema progressCookieSchema
	if err := json.Unmarshal([]byte(result.Value), &schema); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if schema.LastRead.NovelID != "nvl_new" {
		t.Errorf("expected lastRead nvl_new, got %s", schema.LastRead.NovelID)
	}
}

func TestMergeProgressCookies_DeleteLastReadUpdate(t *testing.T) {
	existing := models.CookieValue{
		Value:     `{"novels":{"nvl_deleted":{"chapterId":"chp_del","chapterNum":15,"readAt":2000}},"lastRead":{"novelId":"nvl_deleted","title":"Deleted Novel","author":"Author","coverUrl":"","chapterId":"chp_del","chapterNum":15,"totalChapters":20,"readAt":2000}}`,
		UpdatedAt: 1000,
	}

	incoming := models.CookieValue{
		Value:     `{"novels":{"nvl_next":{"chapterId":"chp_next","chapterNum":10,"readAt":1000}},"lastRead":{"novelId":"nvl_next","title":"Next Novel","author":"Author","coverUrl":"","chapterId":"chp_next","chapterNum":10,"totalChapters":15,"readAt":3000}}`,
		UpdatedAt: 3000,
	}

	result := mergeProgressCookies(existing, incoming)

	var schema progressCookieSchema
	if err := json.Unmarshal([]byte(result.Value), &schema); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := schema.Novels["nvl_deleted"]; ok {
		t.Error("deleted novel should not be in result")
	}

	if schema.LastRead.NovelID != "nvl_next" {
		t.Errorf("expected lastRead nvl_next, got %s", schema.LastRead.NovelID)
	}

	if schema.LastRead.ReadAt != 3000 {
		t.Errorf("expected lastRead readAt 3000, got %d", schema.LastRead.ReadAt)
	}
}

func TestMergeProgressCookies_ExistingLastReadNewerThanIncoming(t *testing.T) {
	existing := models.CookieValue{
		Value:     `{"novels":{"nvl_a":{"chapterId":"chp_a","chapterNum":5,"readAt":2000}},"lastRead":{"novelId":"nvl_a","title":"Novel A","author":"Author","coverUrl":"","chapterId":"chp_a","chapterNum":5,"totalChapters":10,"readAt":2000}}`,
		UpdatedAt: 2000,
	}

	incoming := models.CookieValue{
		Value:     `{"novels":{"nvl_b":{"chapterId":"chp_b","chapterNum":3,"readAt":3000}},"lastRead":{"novelId":"nvl_b","title":"Novel B","author":"Author","coverUrl":"","chapterId":"chp_b","chapterNum":3,"totalChapters":10,"readAt":3000}}`,
		UpdatedAt: 1000,
	}

	result := mergeProgressCookies(existing, incoming)

	var schema progressCookieSchema
	if err := json.Unmarshal([]byte(result.Value), &schema); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if schema.LastRead.NovelID != "nvl_b" {
		t.Errorf("expected lastRead nvl_b (newer read), got %s", schema.LastRead.NovelID)
	}
}
