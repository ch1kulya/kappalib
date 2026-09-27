package data

import (
	"strings"
	"testing"
	"time"

	"github.com/ch1kulya/kappalib/internal/models"
)

func TestCanDeleteComment(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		authorID  string
		profileID string
		expected  bool
	}{
		{"author of chapter comment", "usr_a", "usr_a", "", true},
		{"stranger on chapter comment", "usr_b", "usr_a", "", false},
		{"author of profile comment", "usr_a", "usr_a", "usr_p", true},
		{"profile owner", "usr_p", "usr_a", "usr_p", true},
		{"stranger on profile comment", "usr_b", "usr_a", "usr_p", false},
		{"anonymous on chapter comment", "", "usr_a", "", false},
		{"empty ids never match", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canDeleteComment(tt.userID, tt.authorID, tt.profileID); got != tt.expected {
				t.Errorf("canDeleteComment(%q, %q, %q) = %v, want %v", tt.userID, tt.authorID, tt.profileID, got, tt.expected)
			}
		})
	}
}

func TestIsUnseenComment(t *testing.T) {
	seenAt := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	after := seenAt.Add(time.Minute)
	before := seenAt.Add(-time.Minute)

	tests := []struct {
		name       string
		comment    models.Comment
		approvedAt *time.Time
		expected   bool
	}{
		{"approved after last visit", models.Comment{Status: "approved", UserID: "usr_a", ProfileID: "usr_p"}, &after, true},
		{"approved before last visit", models.Comment{Status: "approved", UserID: "usr_a", ProfileID: "usr_p"}, &before, false},
		{"never approved", models.Comment{Status: "pending", UserID: "usr_a", ProfileID: "usr_p"}, nil, false},
		{"rejected after approval", models.Comment{Status: "rejected", UserID: "usr_a", ProfileID: "usr_p"}, &after, false},
		{"own comment", models.Comment{Status: "approved", UserID: "usr_p", ProfileID: "usr_p"}, &after, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnseenComment(tt.comment, "usr_p", tt.approvedAt, seenAt); got != tt.expected {
				t.Errorf("isUnseenComment() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEarliestSeen(t *testing.T) {
	stored := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		provided time.Time
		expected time.Time
	}{
		{"not provided", time.Time{}, stored},
		{"earlier provided wins", stored.Add(-time.Hour), stored.Add(-time.Hour)},
		{"later provided ignored", stored.Add(time.Hour), stored},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := earliestSeen(stored, tt.provided); !got.Equal(tt.expected) {
				t.Errorf("earliestSeen() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCommentTargetQueriesStayAligned(t *testing.T) {
	pairs := []struct {
		name    string
		profile string
		chapter string
	}{
		{"position", profileCommentQueries.position, chapterCommentQueries.position},
		{"count", profileCommentQueries.count, chapterCommentQueries.count},
		{"list", profileCommentQueries.list, chapterCommentQueries.list},
	}

	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			if p.profile == "" || p.chapter == "" {
				t.Fatal("query is not embedded")
			}
			if got := strings.ReplaceAll(p.profile, "profile_id = $1", "chapter_id = $1"); got != p.chapter {
				t.Errorf("%s queries differ beyond the target column", p.name)
			}
		})
	}
}

func TestBuildProfileCommentTelegramText(t *testing.T) {
	got := buildProfileCommentTelegramText("usr_abc12345", "Мудрый Лис", "User1", "<p>Привет! <strong>bold</strong></p>")

	if !strings.Contains(got, "<p>💬 Новый комментарий в профиле</p>") {
		t.Errorf("expected header in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Профиль</td><td><a href="https://kappalib.rip/usr_abc12345">Мудрый Лис</a></td></tr>`) {
		t.Errorf("expected profile row in %q", got)
	}
	if !strings.Contains(got, `<tr><td>Автор</td><td>User1</td></tr>`) {
		t.Errorf("expected author row in %q", got)
	}
	if strings.Contains(got, "Новелла") || strings.Contains(got, "Глава") {
		t.Errorf("unexpected chapter metadata in %q", got)
	}
	if !strings.Contains(got, "<details open><summary>Текст</summary><p>Привет! <b>bold</b></p></details>") {
		t.Errorf("expected formatted text in %q", got)
	}
}

func TestBuildEditedProfileCommentTelegramText(t *testing.T) {
	got := buildEditedProfileCommentTelegramText("usr_abc12345", "", "User1", "<p>old</p>", "<p>new</p>")

	if !strings.Contains(got, "<p>📝 Новая редакция комментария в профиле</p>") {
		t.Errorf("expected header in %q", got)
	}
	if !strings.Contains(got, `<a href="https://kappalib.rip/usr_abc12345">Профиль пользователя</a>`) {
		t.Errorf("expected fallback profile name in %q", got)
	}
	if !strings.Contains(got, "<details><summary>Старая версия</summary><p>old</p></details>") {
		t.Errorf("expected old version in %q", got)
	}
	if !strings.Contains(got, "<details open><summary>Текст</summary><p>new</p></details>") {
		t.Errorf("expected new version in %q", got)
	}
}

func TestBuildProfileTelegramMetadataTableEscapes(t *testing.T) {
	got := buildProfileTelegramMetadataTable("https://kappalib.rip/usr_1", "<b>x</b>", "a&b")
	if strings.Contains(got, "<b>x</b>") || !strings.Contains(got, "&lt;b&gt;x&lt;/b&gt;") {
		t.Errorf("expected escaped profile name in %q", got)
	}
	if !strings.Contains(got, "a&amp;b") {
		t.Errorf("expected escaped author name in %q", got)
	}
}
