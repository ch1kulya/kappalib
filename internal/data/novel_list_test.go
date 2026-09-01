package data

import (
	"testing"

	"github.com/ch1kulya/kappalib/internal/models"
)

func TestIsValidListStatus(t *testing.T) {
	valid := []string{"completed", "dropped", "on_hold", "planned", "rereading", "reading", "favorite"}
	for _, slug := range valid {
		if !isValidListStatus(slug) {
			t.Errorf("isValidListStatus(%q) = false, want true", slug)
		}
	}

	invalid := []string{"", "Reading", "read", "watching", "on-hold", "completed "}
	for _, slug := range invalid {
		if isValidListStatus(slug) {
			t.Errorf("isValidListStatus(%q) = true, want false", slug)
		}
	}
}

func TestRemoveNovelFromAllCategories(t *testing.T) {
	list := map[string]models.ListCategory{
		"reading": {
			CreatedAt: 100,
			UpdatedAt: 100,
			Novels: []models.ListEntry{
				{ID: "nvl_aaaa1111", AddedAt: 100},
				{ID: "nvl_bbbb2222", AddedAt: 90},
			},
		},
		"planned": {
			CreatedAt: 100,
			UpdatedAt: 100,
			Novels:    []models.ListEntry{{ID: "nvl_cccc3333", AddedAt: 80}},
		},
	}

	list, found := removeNovelFromAllCategories(list, "nvl_aaaa1111")
	if !found {
		t.Fatal("expected novel to be found in reading")
	}
	reading := list["reading"]
	if len(reading.Novels) != 1 || reading.Novels[0].ID != "nvl_bbbb2222" {
		t.Errorf("expected remaining entry nvl_bbbb2222, got %v", reading.Novels)
	}
	if reading.UpdatedAt <= 100 {
		t.Errorf("expected UpdatedAt to be refreshed, got %d", reading.UpdatedAt)
	}

	list, found = removeNovelFromAllCategories(list, "nvl_cccc3333")
	if !found {
		t.Fatal("expected novel to be found in planned")
	}
	if _, exists := list["planned"]; exists {
		t.Errorf("expected emptied category to be removed")
	}

	_, found = removeNovelFromAllCategories(list, "nvl_dddd4444")
	if found {
		t.Error("expected unknown novel not to be found")
	}
}

func TestRemoveNovelFromAllCategoriesPreservesOrder(t *testing.T) {
	list := map[string]models.ListCategory{
		"favorite": {
			CreatedAt: 100,
			UpdatedAt: 100,
			Novels: []models.ListEntry{
				{ID: "nvl_aaaa1111", AddedAt: 30},
				{ID: "nvl_bbbb2222", AddedAt: 20},
				{ID: "nvl_cccc3333", AddedAt: 10},
			},
		},
	}

	list, _ = removeNovelFromAllCategories(list, "nvl_bbbb2222")

	novels := list["favorite"].Novels
	if len(novels) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(novels))
	}
	if novels[0].ID != "nvl_aaaa1111" || novels[1].ID != "nvl_cccc3333" {
		t.Errorf("expected order to be preserved, got %v", novels)
	}
}
