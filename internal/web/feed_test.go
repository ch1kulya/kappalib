package web

import (
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/kappalib/internal/web/views"
)

func TestFormatGroupedUpdatesHTML(t *testing.T) {
	tests := []struct {
		chapters int
		novels   int
		expected string
	}{
		{
			chapters: 1,
			novels:   1,
			expected: `Добавлена <span class="update-log-num">1</span> глава для <span class="update-log-num">1</span> новеллы`,
		},
		{
			chapters: 2,
			novels:   2,
			expected: `Добавлено <span class="update-log-num">2</span> главы для <span class="update-log-num">2</span> новелл`,
		},
		{
			chapters: 5,
			novels:   3,
			expected: `Добавлено <span class="update-log-num">5</span> глав для <span class="update-log-num">3</span> новелл`,
		},
		{
			chapters: 21,
			novels:   21,
			expected: `Добавлена <span class="update-log-num">21</span> глава для <span class="update-log-num">21</span> новеллы`,
		},
	}

	for _, tt := range tests {
		actual := views.FormatGroupedUpdatesHTML(tt.chapters, tt.novels)
		if actual != tt.expected {
			t.Errorf("FormatGroupedUpdatesHTML(%d, %d) = %q; want %q", tt.chapters, tt.novels, actual, tt.expected)
		}
	}
}

func TestFormatNovelAdditionChaptersHTML(t *testing.T) {
	tests := []struct {
		min      int
		max      int
		count    int
		expected string
	}{
		{
			min:      1,
			max:      1,
			count:    1,
			expected: `Новая новелла, добавлена <span class="update-log-num">1</span> гл.`,
		},
		{
			min:      1,
			max:      50,
			count:    50,
			expected: `Новая новелла, добавлены <span class="update-log-num">1-50</span> гл.`,
		},
	}

	for _, tt := range tests {
		actual := views.FormatNovelAdditionChaptersHTML(tt.min, tt.max, tt.count)
		if actual != tt.expected {
			t.Errorf("FormatNovelAdditionChaptersHTML(%d, %d, %d) = %q; want %q", tt.min, tt.max, tt.count, actual, tt.expected)
		}
	}
}

func TestProcessUnpopularCluster_SingleItem(t *testing.T) {
	now := time.Now()
	cluster := []models.NovelUpdate{
		{
			NovelID:      "nvl_1",
			NovelTitle:   "Unpopular Novel",
			ChapterMin:   1,
			ChapterMax:   5,
			ChapterCount: 5,
			UpdatedAt:    now,
		},
	}

	var items []models.HomeUpdateItem
	processUnpopularCluster(cluster, &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != "chapter" {
		t.Errorf("expected Type 'chapter', got %s", items[0].Type)
	}
	if items[0].ChapterUpdate == nil || items[0].ChapterUpdate.NovelID != "nvl_1" {
		t.Errorf("expected ChapterUpdate with NovelID 'nvl_1'")
	}
}

func TestProcessUnpopularCluster_MultipleItems(t *testing.T) {
	now := time.Now()
	cluster := []models.NovelUpdate{
		{
			NovelID:      "nvl_1",
			NovelTitle:   "Unpopular Novel 1",
			ChapterMin:   1,
			ChapterMax:   5,
			ChapterCount: 5,
			UpdatedAt:    now,
		},
		{
			NovelID:      "nvl_2",
			NovelTitle:   "Unpopular Novel 2",
			ChapterMin:   10,
			ChapterMax:   12,
			ChapterCount: 3,
			UpdatedAt:    now.Add(-10 * time.Minute),
		},
		{
			NovelID:      "nvl_1",
			NovelTitle:   "Unpopular Novel 1",
			ChapterMin:   6,
			ChapterMax:   7,
			ChapterCount: 2,
			UpdatedAt:    now.Add(-20 * time.Minute),
		},
	}

	var items []models.HomeUpdateItem
	processUnpopularCluster(cluster, &items)

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Type != "grouped" {
		t.Errorf("expected Type 'grouped', got %s", items[0].Type)
	}
	if items[0].GroupedUpdates == nil {
		t.Fatalf("expected GroupedUpdates not to be nil")
	}
	if items[0].GroupedUpdates.ChapterCount != 10 {
		t.Errorf("expected ChapterCount 10, got %d", items[0].GroupedUpdates.ChapterCount)
	}
	if items[0].GroupedUpdates.NovelCount != 2 {
		t.Errorf("expected NovelCount 2, got %d", items[0].GroupedUpdates.NovelCount)
	}
	if !items[0].GroupedUpdates.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, items[0].GroupedUpdates.UpdatedAt)
	}
}

func TestTimelineCutoffCalculation(t *testing.T) {
	t1 := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	oldestDates := []time.Time{t1, t2, t3}

	cutoff := oldestDates[0]
	for _, d := range oldestDates[1:] {
		if d.After(cutoff) {
			cutoff = d
		}
	}

	if !cutoff.Equal(t1) {
		t.Errorf("expected cutoff %v, got %v", t1, cutoff)
	}
}

func TestNovelAdditionAndChaptersMatch(t *testing.T) {
	now := time.Now()
	novelAddition := models.NovelAddition{
		ID:        "nvl_new",
		Title:     "Brand New Novel",
		Author:    "Author",
		YearStart: 2026,
		Status:    "ongoing",
		CreatedAt: now.Add(-10 * time.Minute),
	}
	chapterUpdate := models.NovelUpdate{
		NovelID:      "nvl_new",
		NovelTitle:   "Brand New Novel",
		ChapterMin:   1,
		ChapterMax:   20,
		ChapterCount: 20,
		UpdatedAt:    now,
	}

	novelAdditions := []models.NovelAddition{novelAddition}
	chapterUpdates := []models.NovelUpdate{chapterUpdate}

	novelAddMap := make(map[string]*matchedNovelAddition)
	for i := range novelAdditions {
		novelAddMap[novelAdditions[i].ID] = &matchedNovelAddition{addition: &novelAdditions[i]}
	}

	var remainingChapters []models.NovelUpdate
	var feedItems []models.HomeUpdateItem

	for _, cu := range chapterUpdates {
		if m, exists := novelAddMap[cu.NovelID]; exists && !m.matched {
			diff := cu.UpdatedAt.Sub(m.addition.CreatedAt)
			if diff < 0 {
				diff = -diff
			}
			if diff <= time.Hour {
				m.matched = true
				updatedAt := m.addition.CreatedAt
				if cu.UpdatedAt.After(updatedAt) {
					updatedAt = cu.UpdatedAt
				}
				feedItems = append(feedItems, models.HomeUpdateItem{
					Type: "novel_with_chapters",
					NovelAdditionChapters: &models.NovelAdditionChapters{
						ID:           m.addition.ID,
						Title:        m.addition.Title,
						Author:       m.addition.Author,
						YearStart:    m.addition.YearStart,
						Status:       m.addition.Status,
						Description:  m.addition.Description,
						CoverURL:     m.addition.CoverURL,
						ChapterMin:   cu.ChapterMin,
						ChapterMax:   cu.ChapterMax,
						ChapterCount: cu.ChapterCount,
						UpdatedAt:    updatedAt,
					},
					UpdatedAt: updatedAt,
				})
				continue
			}
		}
		remainingChapters = append(remainingChapters, cu)
	}

	for _, m := range novelAddMap {
		if !m.matched {
			feedItems = append(feedItems, models.HomeUpdateItem{
				Type:          "novel",
				NovelAddition: m.addition,
				UpdatedAt:     m.addition.CreatedAt,
			})
		}
	}

	if len(remainingChapters) != 0 {
		t.Errorf("expected 0 remaining chapters, got %d", len(remainingChapters))
	}
	if len(feedItems) != 1 {
		t.Fatalf("expected 1 feed item, got %d", len(feedItems))
	}
	if feedItems[0].Type != "novel_with_chapters" {
		t.Errorf("expected Type 'novel_with_chapters', got %s", feedItems[0].Type)
	}
	if feedItems[0].NovelAdditionChapters.ID != "nvl_new" {
		t.Errorf("expected ID 'nvl_new', got %s", feedItems[0].NovelAdditionChapters.ID)
	}
	if feedItems[0].NovelAdditionChapters.ChapterCount != 20 {
		t.Errorf("expected ChapterCount 20, got %d", feedItems[0].NovelAdditionChapters.ChapterCount)
	}
}
