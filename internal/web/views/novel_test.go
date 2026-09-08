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

func TestNovelRendersAbandonedBadge(t *testing.T) {
	fourMonthsAgo := time.Now().AddDate(0, -4, 0)
	oneMonthAgo := time.Now().AddDate(0, -1, 0)

	tests := []struct {
		name        string
		novel       *models.Novel
		shouldExist bool
	}{
		{
			name: "abandoned ongoing shows badge",
			novel: &models.Novel{
				ID:            "nvl_abandoned",
				Title:         "Abandoned Novel",
				Status:        "ongoing",
				LastChapterAt: &fourMonthsAgo,
			},
			shouldExist: true,
		},
		{
			name: "active ongoing does not show badge",
			novel: &models.Novel{
				ID:            "nvl_active",
				Title:         "Active Novel",
				Status:        "ongoing",
				LastChapterAt: &oneMonthAgo,
			},
			shouldExist: false,
		},
		{
			name: "completed novel does not show badge",
			novel: &models.Novel{
				ID:            "nvl_completed",
				Title:         "Completed Novel",
				Status:        "completed",
				LastChapterAt: &fourMonthsAgo,
			},
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := NovelProps{
				BaseProps: BaseProps{
					Title:          "t",
					Description:    "d",
					Version:        "test",
					ReaderSettings: DefaultReaderSettings,
				},
				Novel: tt.novel,
			}
			var sb strings.Builder
			if err := Novel(props).Render(context.Background(), &sb); err != nil {
				t.Fatalf("render failed: %v", err)
			}
			output := sb.String()
			hasBadge := strings.Contains(output, `<span class="badge badge-danger">Заброшено</span>`)
			if hasBadge != tt.shouldExist {
				t.Errorf("render abandoned badge = %v, want %v", hasBadge, tt.shouldExist)
			}
		})
	}
}
