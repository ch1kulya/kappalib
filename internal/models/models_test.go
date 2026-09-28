package models

import (
	"testing"
	"time"
)

func TestNovel_IsAbandoned(t *testing.T) {
	now := time.Now()
	fourMonthsAgo := now.AddDate(0, -4, 0)
	oneMonthAgo := now.AddDate(0, -1, 0)

	tests := []struct {
		name     string
		novel    Novel
		expected bool
	}{
		{
			name: "ongoing with last chapter 4 months ago",
			novel: Novel{
				Status:        "ongoing",
				LastChapterAt: &fourMonthsAgo,
			},
			expected: true,
		},
		{
			name: "ongoing with last chapter 1 month ago",
			novel: Novel{
				Status:        "ongoing",
				LastChapterAt: &oneMonthAgo,
			},
			expected: false,
		},
		{
			name: "ongoing with no chapters created 4 months ago",
			novel: Novel{
				Status:    "ongoing",
				CreatedAt: fourMonthsAgo,
			},
			expected: true,
		},
		{
			name: "ongoing with no chapters created 1 month ago",
			novel: Novel{
				Status:    "ongoing",
				CreatedAt: oneMonthAgo,
			},
			expected: false,
		},
		{
			name: "completed novel with last chapter 4 months ago",
			novel: Novel{
				Status:        "completed",
				LastChapterAt: &fourMonthsAgo,
			},
			expected: false,
		},
		{
			name: "announced novel with last chapter 4 months ago",
			novel: Novel{
				Status:        "announced",
				LastChapterAt: &fourMonthsAgo,
			},
			expected: false,
		},
		{
			name: "case insensitive status Ongoing",
			novel: Novel{
				Status:        "Ongoing",
				LastChapterAt: &fourMonthsAgo,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.novel.IsAbandoned()
			if got != tt.expected {
				t.Errorf("IsAbandoned() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSetAuthor(t *testing.T) {
	author := ProfilePublic{DisplayName: "Мудрый Лис", AvatarSeed: "seed", HasCustomAvatar: true, AvatarUpdatedAt: 42}

	var c Comment
	c.SetAuthor(author)
	if c.UserDisplayName != "Мудрый Лис" || c.UserAvatarSeed != "seed" || !c.UserHasCustomAvatar || c.UserAvatarUpdatedAt != 42 {
		t.Errorf("Comment.SetAuthor did not copy fields: %+v", c)
	}

	var a CommentAnswer
	a.SetAuthor(author)
	if a.UserDisplayName != "Мудрый Лис" || a.UserAvatarSeed != "seed" || !a.UserHasCustomAvatar || a.UserAvatarUpdatedAt != 42 {
		t.Errorf("CommentAnswer.SetAuthor did not copy fields: %+v", a)
	}
}
