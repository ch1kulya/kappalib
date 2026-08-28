package data

import (
	"testing"

	"github.com/ch1kulya/kappalib/internal/models"
)

func TestSanitizeBookmarkField(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		maxLen   int
		want     string
	}{
		{"empty uses fallback", "", "default", 50, "default"},
		{"whitespace uses fallback", "   ", "default", 50, "default"},
		{"html stripped", "<b>Hello</b>", "default", 50, "Hello"},
		{"spaces normalized", "a   b   c", "default", 50, "a b c"},
		{"truncated to maxLen", "abcdefghij", "default", 5, "abcde"},
		{"unicode length truncated", "Привет Мир", "default", 6, "Привет"},
		{"category maxLen 15", "ОченьДлинноеНазваниеКатегории", "default", maxCategoryNameLen, "ОченьДлинноеНаз"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBookmarkField(tt.value, tt.fallback, tt.maxLen)
			if got != tt.want {
				t.Errorf("sanitizeBookmarkField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInsertBookmark(t *testing.T) {
	bookmarks := make(map[string]models.BookmarkCategory)
	bm1 := models.Bookmark{ID: "bkm_1", ChapterID: "chp_1", Value: "Bookmark 1", CreatedAt: 100, UpdatedAt: 100}
	bm2 := models.Bookmark{ID: "bkm_2", ChapterID: "chp_2", Value: "Bookmark 2", CreatedAt: 200, UpdatedAt: 200}

	bookmarks = insertBookmark(bookmarks, "Category A", bm1)
	catA, ok := bookmarks["Category A"]
	if !ok || len(catA.Bookmarks) != 1 {
		t.Fatalf("expected 1 item, got %v", catA)
	}
	if catA.CreatedAt == 0 || catA.UpdatedAt == 0 {
		t.Errorf("expected timestamps to be set on category creation, got %d, %d", catA.CreatedAt, catA.UpdatedAt)
	}

	bookmarks = insertBookmark(bookmarks, "Category A", bm2)
	catA = bookmarks["Category A"]
	if len(catA.Bookmarks) != 2 {
		t.Fatalf("expected 2 items, got %d", len(catA.Bookmarks))
	}
	if catA.Bookmarks[0].ID != "bkm_2" {
		t.Errorf("expected bm2 to be prepended, got %s", catA.Bookmarks[0].ID)
	}
}

func TestRemoveBookmarkByID(t *testing.T) {
	bm1 := models.Bookmark{ID: "bkm_1", ChapterID: "chp_1", CreatedAt: 100, UpdatedAt: 100}
	bm2 := models.Bookmark{ID: "bkm_2", ChapterID: "chp_2", CreatedAt: 200, UpdatedAt: 200}
	bookmarks := map[string]models.BookmarkCategory{
		"Cat1": {CreatedAt: 100, UpdatedAt: 100, Bookmarks: []models.Bookmark{bm1, bm2}},
		"Cat2": {CreatedAt: 100, UpdatedAt: 100, Bookmarks: []models.Bookmark{}},
	}

	removed, remaining := removeBookmarkByID(bookmarks, "bkm_1")
	if removed == nil || removed.ID != "bkm_1" {
		t.Fatalf("expected removed bookmark bkm_1, got %v", removed)
	}
	cat1 := remaining["Cat1"]
	if len(cat1.Bookmarks) != 1 || cat1.Bookmarks[0].ID != "bkm_2" {
		t.Errorf("unexpected remaining bookmarks: %v", remaining)
	}

	removedNone, _ := removeBookmarkByID(remaining, "bkm_nonexistent")
	if removedNone != nil {
		t.Errorf("expected nil for nonexistent bookmark, got %v", removedNone)
	}

	removedLast, remainingAfterLast := removeBookmarkByID(remaining, "bkm_2")
	if removedLast == nil || removedLast.ID != "bkm_2" {
		t.Fatalf("expected removed bookmark bkm_2, got %v", removedLast)
	}
	if _, exists := remainingAfterLast["Cat1"]; exists {
		t.Errorf("expected Cat1 to be deleted after removing last bookmark, but it still exists")
	}
}

func TestFindBookmark(t *testing.T) {
	bm1 := models.Bookmark{ID: "bkm_1", ChapterID: "chp_1", CreatedAt: 100, UpdatedAt: 100}
	bookmarks := map[string]models.BookmarkCategory{
		"Cat1": {CreatedAt: 100, UpdatedAt: 100, Bookmarks: []models.Bookmark{bm1}},
	}

	found, cat := findBookmark(bookmarks, "bkm_1")
	if found == nil || found.ID != "bkm_1" {
		t.Fatalf("expected to find bkm_1, got %v", found)
	}
	if cat != "Cat1" {
		t.Errorf("expected category Cat1, got %s", cat)
	}

	notFound, catNotFound := findBookmark(bookmarks, "bkm_unknown")
	if notFound != nil || catNotFound != "" {
		t.Errorf("expected nil for unknown bookmark, got %v, %s", notFound, catNotFound)
	}
}

func TestSanitizeCategoryName(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{"empty uses fallback", "", "Избранное", "Избранное"},
		{"whitespace uses fallback", "   \t\n  ", "Избранное", "Избранное"},
		{"slashes converted to spaces", "Sci/Fi\\Fantasy", "Избранное", "Sci Fi Fantasy"},
		{"multiple slashes normalized", "a///b\\\\\\c", "Избранное", "a b c"},
		{"only slashes uses fallback", "///\\\\\\", "Избранное", "Избранное"},
		{"control characters stripped", "Cat\x00\x07\x1bName", "Избранное", "Cat Name"},
		{"html stripped", "<b>Reading</b>", "Избранное", "Reading"},
		{"truncated to maxCategoryNameLen", "ОченьДлинноеНазваниеКатегории", "Избранное", "ОченьДлинноеНаз"},
		{"unicode with spaces and slashes", "Лайт / Новеллы", "Избранное", "Лайт Новеллы"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCategoryName(tt.value, tt.fallback)
			if got != tt.want {
				t.Errorf("sanitizeCategoryName(%q, %q) = %q, want %q", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}
