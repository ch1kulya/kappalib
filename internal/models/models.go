package models

import "time"

type Novel struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	TitleEn     string    `json:"title_en"`
	Author      string    `json:"author"`
	YearStart   int       `json:"year_start"`
	YearEnd     *int      `json:"year_end"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	AgeRating   *string   `json:"age_rating"`
	CoverURL    *string   `json:"cover_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type NovelsPage struct {
	Novels     []Novel `json:"novels"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalCount int     `json:"total_count"`
	TotalPages int     `json:"total_pages"`
}

type Source struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}

type ChapterSummary struct {
	ID         string  `json:"id"`
	ChapterNum int     `json:"chapter_num"`
	Title      string  `json:"title"`
	TitleEn    *string `json:"title_en"`
}

type Chapter struct {
	ID         string    `json:"id"`
	NovelID    string    `json:"novel_id"`
	ChapterNum int       `json:"chapter_num"`
	Title      string    `json:"title"`
	TitleEn    *string   `json:"title_en"`
	Content    string    `json:"content"`
	Source     *Source   `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
}

type ChaptersList struct {
	Chapters []ChapterSummary `json:"chapters"`
	NovelID  string           `json:"novel_id"`
	Count    int              `json:"count"`
}

type SitemapItem struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type CookieValue struct {
	Value     string `json:"value"`
	UpdatedAt int64  `json:"updated_at"`
}

type ProfilePublic struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	AvatarSeed  string    `json:"avatar_seed"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProfileWithToken struct {
	ID          string    `json:"id"`
	SecretToken string    `json:"secret_token"`
	DisplayName string    `json:"display_name"`
	AvatarSeed  string    `json:"avatar_seed"`
	CreatedAt   time.Time `json:"created_at"`
}

type SyncCodeResponse struct {
	SyncCode  string `json:"sync_code"`
	ExpiresAt string `json:"expires_at"`
}

type LoginResponse struct {
	Profile     ProfilePublic          `json:"profile"`
	SecretToken string                 `json:"secret_token"`
	Cookies     map[string]CookieValue `json:"cookies"`
}

type Comment struct {
	ID                string    `json:"id"`
	ChapterID         string    `json:"chapter_id"`
	UserID            string    `json:"user_id"`
	ContentHTML       string    `json:"content_html"`
	Status            string    `json:"status"`
	TelegramMessageID *int64    `json:"telegram_message_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UserDisplayName   string    `json:"user_display_name,omitempty"`
	UserAvatarSeed    string    `json:"user_avatar_seed,omitempty"`
}

type CommentsPage struct {
	Comments   []Comment `json:"comments"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalCount int       `json:"total_count"`
	TotalPages int       `json:"total_pages"`
}

type CreateCommentInput struct {
	ChapterID      string `json:"chapter_id"`
	Content        string `json:"content"`
	TurnstileToken string `json:"turnstile_token"`
}
