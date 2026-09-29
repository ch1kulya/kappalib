package views

import (
	"context"
	"strings"
	"testing"

	"github.com/ch1kulya/kappalib/internal/models"
)

func renderToString(t *testing.T, render func(*strings.Builder) error) string {
	t.Helper()
	var sb strings.Builder
	if err := render(&sb); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	return sb.String()
}

func TestCatalogStatsRendersCompactValues(t *testing.T) {
	stats := models.CatalogStats{
		NovelsCount:     21,
		ChaptersCount:   1_250,
		SourcesCount:    3,
		CharactersCount: 1_234_567_890,
	}
	html := renderToString(t, func(sb *strings.Builder) error {
		return CatalogStats(stats).Render(context.Background(), sb)
	})

	for _, want := range []string{
		`<span class="catalog-stat-value">21</span> <span class="catalog-stat-label">новелла</span>`,
		`<span class="catalog-stat-value">1,2 тыс.</span> <span class="catalog-stat-label">глав</span>`,
		`<span class="catalog-stat-value">3</span> <span class="catalog-stat-label">источника</span>`,
		`<span class="catalog-stat-value">1,2 млрд</span> <span class="catalog-stat-label">знаков</span>`,
		"title=\"1 250 глав\"",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("stats html missing %q", want)
		}
	}
}

func TestCatalogFiltersRendersState(t *testing.T) {
	yearFrom := 2015
	props := CatalogProps{
		Filter: models.CatalogFilter{
			TagIDs:   []int{7},
			Statuses: []string{"completed"},
			YearFrom: &yearFrom,
		},
		FilterTags: []models.Tag{{ID: 5, Name: "Драма"}, {ID: 7, Name: "Фэнтези"}},
	}
	html := renderToString(t, func(sb *strings.Builder) error {
		return CatalogFilters(props).Render(context.Background(), sb)
	})

	for _, want := range []string{
		`data-filter="tag"`,
		`class="dropdown-item selected" data-value="7"`,
		`class="dropdown-item" data-value="5"`,
		`class="dropdown-item selected" data-value="completed"`,
		`<span class="badge catalog-filter-badge">от 2015</span>`,
		`name="year_from" inputmode="numeric" autocomplete="off" placeholder="от" aria-label="Год выхода от" data-min="1" data-max="9999" value="2015"`,
		`id="catalog-filters-reset">Сбросить</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("filters html missing %q", want)
		}
	}

	empty := renderToString(t, func(sb *strings.Builder) error {
		return CatalogFilters(CatalogProps{}).Render(context.Background(), sb)
	})
	if strings.Contains(empty, `data-filter="tag"`) {
		t.Error("tag filter should be hidden when there are no tags")
	}
	if !strings.Contains(empty, `id="catalog-filters-reset" hidden>`) {
		t.Error("reset button should be hidden without active filters")
	}
}

func TestCatalogResultsPartialRendersOnlyResults(t *testing.T) {
	props := CatalogProps{
		Novels:           []models.NovelSummary{{ID: "nvl_test", Title: "Test"}},
		Page:             1,
		TotalPages:       3,
		TotalCount:       60,
		SortOrder:        "popular",
		IsResultsPartial: true,
		Filter:           models.CatalogFilter{Statuses: []string{"ongoing"}},
	}
	html := renderToString(t, func(sb *strings.Builder) error {
		return Catalog(props).Render(context.Background(), sb)
	})

	if !strings.HasPrefix(html, `<div id="catalog-content" data-total-pages="3"`) {
		t.Errorf("results partial should start with catalog content, got %q", html[:min(len(html), 80)])
	}
	if strings.Contains(html, "<html") || strings.Contains(html, "catalog-filters") {
		t.Error("results partial should not render page layout or filters")
	}
	if !strings.Contains(html, "60 новелл по выбранным фильтрам") {
		t.Error("results partial should render filtered meta")
	}
}
