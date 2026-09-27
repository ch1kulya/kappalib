package data

import (
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
)

func TestBuildAchievements(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		stats     models.UserProfileStats
		createdAt time.Time
		expected  map[string]int
	}{
		{
			name:      "new user has only registration",
			stats:     models.UserProfileStats{},
			createdAt: now.Add(-time.Hour),
			expected:  map[string]int{"registered": 1},
		},
		{
			name:      "zero created at still counts as registered",
			stats:     models.UserProfileStats{},
			createdAt: time.Time{},
			expected:  map[string]int{"registered": 1},
		},
		{
			name:      "exact thresholds unlock tiers",
			stats:     models.UserProfileStats{TotalSeconds: 10 * 3600, LongestStreak: 30, Comments: 1, Rating: 200},
			createdAt: now.AddDate(0, 0, -365),
			expected:  map[string]int{"registered": 1, "reader": 1, "streak": 2, "commentator": 1, "rating": 3, "veteran": 3},
		},
		{
			name:      "below thresholds stay locked",
			stats:     models.UserProfileStats{TotalSeconds: 10*3600 - 1, LongestStreak: 6, Comments: 0, Rating: 9},
			createdAt: now.AddDate(0, 0, -29),
			expected:  map[string]int{"registered": 1},
		},
		{
			name:      "values above max tier cap at max",
			stats:     models.UserProfileStats{TotalSeconds: 10000 * 3600, LongestStreak: 1000, Comments: 5000, Rating: 99999},
			createdAt: now.AddDate(-5, 0, 0),
			expected:  map[string]int{"registered": 1, "reader": 4, "streak": 4, "commentator": 4, "rating": 4, "veteran": 4},
		},
		{
			name:      "negative rating unlocks nothing",
			stats:     models.UserProfileStats{Rating: -50, Comments: 12},
			createdAt: now,
			expected:  map[string]int{"registered": 1, "commentator": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAchievements(tt.stats, tt.createdAt, now)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d achievements, got %d: %+v", len(tt.expected), len(got), got)
			}
			for _, a := range got {
				tier, ok := tt.expected[a.ID]
				if !ok {
					t.Errorf("unexpected achievement %q", a.ID)
					continue
				}
				if a.Tier != tier {
					t.Errorf("achievement %q: expected tier %d, got %d", a.ID, tier, a.Tier)
				}
				expectedMax := 4
				if a.ID == "registered" {
					expectedMax = 1
				}
				if a.MaxTier != expectedMax {
					t.Errorf("achievement %q: expected max tier %d, got %d", a.ID, expectedMax, a.MaxTier)
				}
				if a.Title == "" || a.Description == "" || a.Icon == "" {
					t.Errorf("achievement %q has empty fields: %+v", a.ID, a)
				}
			}
		})
	}
}

func TestBuildAchievementsOrderIsStable(t *testing.T) {
	now := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	stats := models.UserProfileStats{TotalSeconds: 60 * 3600, LongestStreak: 10, Comments: 20, Rating: 20}

	got := BuildAchievements(stats, now.AddDate(-1, 0, -1), now)
	expected := []string{"registered", "reader", "streak", "commentator", "rating", "veteran"}
	if len(got) != len(expected) {
		t.Fatalf("expected %d achievements, got %d", len(expected), len(got))
	}
	for i, id := range expected {
		if got[i].ID != id {
			t.Errorf("position %d: expected %q, got %q", i, id, got[i].ID)
		}
	}
}

func TestUserProfileCacheKey(t *testing.T) {
	if got := userProfileCacheKey("usr_abc12345"); got != "user_profile:usr_abc12345" {
		t.Errorf("unexpected cache key %q", got)
	}
}
