package views

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
)

func TestNovelRendersListStatusIcon(t *testing.T) {
	novel := &models.Novel{ID: "nvl_test", Title: "Test", ChapterCount: 1}

	render := func(listStatus string) string {
		props := NovelProps{
			BaseProps: BaseProps{
				Title:          "t",
				Description:    "d",
				Version:        "test",
				ReaderSettings: DefaultReaderSettings,
			},
			Novel:      novel,
			ListStatus: listStatus,
		}
		var sb strings.Builder
		if err := Novel(props).Render(context.Background(), &sb); err != nil {
			t.Fatalf("render failed: %v", err)
		}
		return sb.String()
	}

	empty := render("")
	if !strings.Contains(empty, "ls-btn-plus") {
		t.Error("empty status should render plus icon")
	}
	if strings.Contains(empty, `class="ls-remove-wrap" style="display: block"`) {
		t.Error("empty status should hide remove wrap")
	}
	if !strings.Contains(empty, `class="ls-remove-wrap" style="display: none;"`) {
		t.Error("empty status should hide remove wrap")
	}

	withStatus := render("reading")
	if strings.Contains(withStatus, "ls-btn-plus") {
		t.Error("status set should not render plus icon")
	}
	if !strings.Contains(withStatus, `data-slug="reading"`) {
		t.Error("status set should render status icon with data-slug")
	}
	if !strings.Contains(withStatus, `class="dropdown-item selected"`) {
		t.Error("matching dropdown item should be selected")
	}
	if !strings.Contains(withStatus, `aria-selected="true"`) {
		t.Error("matching dropdown item should have aria-selected true")
	}
	if !strings.Contains(withStatus, `class="ls-remove-wrap" style="display: block;"`) {
		t.Error("status set should show remove wrap")
	}
}

func renderNovel(t *testing.T, novel *models.Novel, lastChapterID string) string {
	t.Helper()
	props := NovelProps{
		BaseProps: BaseProps{
			Title:          "t",
			Description:    "d",
			Version:        "test",
			ReaderSettings: DefaultReaderSettings,
		},
		Novel:          novel,
		Chapters:       []models.ChapterSummary{{ID: "chp_1", ChapterNum: 1, Title: "Без названия"}},
		FirstChapterID: "chp_1",
		LastChapterID:  lastChapterID,
		TotalChapters:  novel.ChapterCount,
	}
	var sb strings.Builder
	if err := Novel(props).Render(context.Background(), &sb); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return sb.String()
}

func metaBlock(t *testing.T, output string) string {
	t.Helper()
	_, after, ok := strings.Cut(output, `<div class="meta">`)
	if !ok {
		t.Fatal("meta block not found")
	}
	meta, _, _ := strings.Cut(after, "</div>")
	return meta
}

func TestNovelRendersStatsBlock(t *testing.T) {
	updatedAt := time.Date(2026, time.March, 5, 12, 0, 0, 0, time.UTC)
	novel := &models.Novel{ID: "nvl_stats", Title: "Popular Novel", ChapterCount: 21, ViewsCount: 1_234_567, LastChapterAt: &updatedAt}
	output := renderNovel(t, novel, "")

	if !strings.Contains(output, "<h4>Статистика</h4>") {
		t.Error("stats block should render even without alt titles and tags")
	}
	if !strings.Contains(output, `<span class="stat badge">21 глава</span>`) {
		t.Error("stats block should render pluralized chapters count")
	}
	if !strings.Contains(output, "<span class=\"stat badge\" title=\"1\u00a0234\u00a0567 просмотров\">1,2 млн просмотров</span>") {
		t.Error("stats block should render compact views count with full count in title")
	}
	if !strings.Contains(output, `<span class="stat badge" title="5 марта 2026 г.">обновлено `+FormatRelativeTime(updatedAt)+"</span>") {
		t.Error("stats block should render last update with full date in title")
	}
	if strings.Contains(metaBlock(t, output), "просмотр") {
		t.Error("views should not be rendered among meta badges")
	}

	noChapters := renderNovel(t, &models.Novel{ID: "nvl_empty", Title: "Empty Novel"}, "")
	if strings.Contains(noChapters, "обновлено") {
		t.Error("last update should be hidden when novel has no chapters")
	}
}

func TestNovelRendersAbandonedStatus(t *testing.T) {
	fourMonthsAgo := time.Now().AddDate(0, -4, 0)
	oneMonthAgo := time.Now().AddDate(0, -1, 0)

	tests := []struct {
		name     string
		novel    *models.Novel
		expected string
		absent   string
	}{
		{
			name: "abandoned ongoing replaces status",
			novel: &models.Novel{
				ID:            "nvl_abandoned",
				Title:         "Abandoned Novel",
				Status:        "ongoing",
				LastChapterAt: &fourMonthsAgo,
			},
			expected: `<span class="badge badge-danger">Заброшено</span>`,
			absent:   "Онгоинг",
		},
		{
			name: "active ongoing keeps status",
			novel: &models.Novel{
				ID:            "nvl_active",
				Title:         "Active Novel",
				Status:        "ongoing",
				LastChapterAt: &oneMonthAgo,
			},
			expected: `<span class="badge">Онгоинг</span>`,
			absent:   "Заброшено",
		},
		{
			name: "completed novel keeps status",
			novel: &models.Novel{
				ID:            "nvl_completed",
				Title:         "Completed Novel",
				Status:        "completed",
				LastChapterAt: &fourMonthsAgo,
			},
			expected: `<span class="badge">Завершено</span>`,
			absent:   "Заброшено",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderNovel(t, tt.novel, "")
			meta := metaBlock(t, output)
			if !strings.Contains(meta, tt.expected) {
				t.Errorf("meta should contain %q", tt.expected)
			}
			if strings.Contains(output, tt.absent) {
				t.Errorf("page should not contain %q", tt.absent)
			}
		})
	}
}
