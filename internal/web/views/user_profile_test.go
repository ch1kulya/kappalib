package views

import (
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
)

func activityDays(start time.Time, count int, seconds func(i int) int) []models.ActivityDay {
	days := make([]models.ActivityDay, count)
	for i := range days {
		days[i] = models.ActivityDay{Date: start.AddDate(0, 0, i), Seconds: seconds(i)}
	}
	return days
}

func TestHeatmapLevel(t *testing.T) {
	tests := []struct {
		seconds  int
		expected int
	}{
		{0, 0},
		{1, 1},
		{599, 1},
		{600, 2},
		{1799, 2},
		{1800, 3},
		{5399, 3},
		{5400, 4},
		{86400, 4},
	}

	for _, tt := range tests {
		if got := heatmapLevel(tt.seconds); got != tt.expected {
			t.Errorf("heatmapLevel(%d) = %d, want %d", tt.seconds, got, tt.expected)
		}
	}
}

func TestBuildActivityHeatmapFullYear(t *testing.T) {
	start := time.Date(2025, 9, 29, 0, 0, 0, 0, time.UTC)
	days := activityDays(start, 364+7, func(i int) int { return i })

	heatmap := BuildActivityHeatmap(days)

	if len(heatmap.Weeks) != 53 {
		t.Fatalf("expected 53 weeks, got %d", len(heatmap.Weeks))
	}
	for i, week := range heatmap.Weeks {
		if len(week) != 7 {
			t.Errorf("week %d: expected 7 days, got %d", i, len(week))
		}
		if week[0].Empty || week[0].Date.Weekday() != time.Monday {
			t.Errorf("week %d does not start on Monday: %+v", i, week[0])
		}
	}

	expectedTotal := 0
	for i := range days {
		expectedTotal += i
	}
	if heatmap.TotalSeconds != expectedTotal {
		t.Errorf("expected total %d, got %d", expectedTotal, heatmap.TotalSeconds)
	}
}

func TestBuildActivityHeatmapPartialWeeks(t *testing.T) {
	start := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	days := activityDays(start, 6, func(int) int { return 900 })

	heatmap := BuildActivityHeatmap(days)

	if len(heatmap.Weeks) != 2 {
		t.Fatalf("expected 2 weeks, got %d", len(heatmap.Weeks))
	}

	first := heatmap.Weeks[0]
	if len(first) != 7 {
		t.Fatalf("expected padded first week of 7 cells, got %d", len(first))
	}
	for i := range 3 {
		if !first[i].Empty {
			t.Errorf("cell %d should be empty padding", i)
		}
	}
	if first[3].Empty || !first[3].Date.Equal(start) {
		t.Errorf("expected first real cell to be %v, got %+v", start, first[3])
	}
	if first[3].Level != 2 {
		t.Errorf("expected level 2, got %d", first[3].Level)
	}

	if len(heatmap.Weeks[1]) != 2 {
		t.Errorf("expected partial last week of 2 days, got %d", len(heatmap.Weeks[1]))
	}
	if heatmap.TotalSeconds != 6*900 {
		t.Errorf("expected total %d, got %d", 6*900, heatmap.TotalSeconds)
	}
}

func TestBuildActivityHeatmapEmpty(t *testing.T) {
	heatmap := BuildActivityHeatmap(nil)
	if len(heatmap.Weeks) != 0 || len(heatmap.Months) != 0 || heatmap.TotalSeconds != 0 {
		t.Errorf("expected empty heatmap, got %+v", heatmap)
	}
}

func TestBuildHeatmapMonths(t *testing.T) {
	start := time.Date(2025, 9, 29, 0, 0, 0, 0, time.UTC)
	heatmap := BuildActivityHeatmap(activityDays(start, 364+7, func(int) int { return 0 }))

	if len(heatmap.Months) == 0 {
		t.Fatal("expected month labels")
	}
	if heatmap.Months[0].Label != "Окт" || heatmap.Months[0].Column != 1 {
		t.Errorf("expected first label Окт at column 1, got %+v", heatmap.Months[0])
	}
	for i := 1; i < len(heatmap.Months); i++ {
		if heatmap.Months[i].Column-heatmap.Months[i-1].Column < 2 {
			t.Errorf("labels %d and %d overlap: %+v", i-1, i, heatmap.Months)
		}
	}
	last := heatmap.Months[len(heatmap.Months)-1]
	if last.Column > len(heatmap.Weeks)-2 {
		t.Errorf("last label too close to the edge: %+v", last)
	}
}

func TestBuildHeatmapMonthsDropsCrowdedFirstLabel(t *testing.T) {
	weeks := [][]HeatmapCell{
		{{Date: time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)}},
		{{Date: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)}},
		{{Date: time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)}},
		{{Date: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)}},
	}

	months := buildHeatmapMonths(weeks)
	if len(months) != 1 || months[0].Label != "Фев" || months[0].Column != 1 {
		t.Errorf("expected only Фев at column 1, got %+v", months)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "0 мин"},
		{-5, "0 мин"},
		{30, "меньше минуты"},
		{60, "1 мин"},
		{59 * 60, "59 мин"},
		{3600, "1 ч"},
		{3600 + 5*60, "1 ч 5 мин"},
		{125 * 3600, "125 ч"},
	}

	for _, tt := range tests {
		if got := FormatDuration(tt.seconds); got != tt.expected {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.seconds, got, tt.expected)
		}
	}
}

func TestFormatDate(t *testing.T) {
	got := FormatDate(time.Date(2025, 10, 3, 15, 0, 0, 0, time.UTC))
	if got != "3 октября 2025" {
		t.Errorf("FormatDate = %q", got)
	}
}

func TestHeatmapCellLabel(t *testing.T) {
	date := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if got := heatmapCellLabel(HeatmapCell{Date: date}); got != "Нет активности · 1 мая 2026" {
		t.Errorf("unexpected empty label %q", got)
	}
	if got := heatmapCellLabel(HeatmapCell{Date: date, Seconds: 2700}); got != "45 мин · 1 мая 2026" {
		t.Errorf("unexpected label %q", got)
	}
}

func TestAvatarURL(t *testing.T) {
	t.Setenv("S3_PUBLIC_URL", "https://cdn.example.com")

	if got := AvatarURL("usr_abc12345", true, "seed", 42); got != "https://cdn.example.com/avatars/usr_abc12345.jpg?v=42" {
		t.Errorf("unexpected custom avatar URL %q", got)
	}
	if got := AvatarURL("usr_abc12345", false, "a b&c", 0); got != "https://api.dicebear.com/9.x/bottts-neutral/svg?seed=a+b%26c&backgroundType=solid,gradientLinear" {
		t.Errorf("unexpected generated avatar URL %q", got)
	}
}

func TestAchievementTierLabel(t *testing.T) {
	if got := achievementTierLabel(3); got != "x3" {
		t.Errorf("achievementTierLabel(3) = %q, want %q", got, "x3")
	}
}
