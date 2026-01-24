package models

import "time"

type Novel struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	TitleEn      string    `json:"title_en"`
	Author       string    `json:"author"`
	YearStart    int       `json:"year_start"`
	YearEnd      *int      `json:"year_end"`
	Status       string    `json:"status"`
	Description  string    `json:"description"`
	AgeRating    *string   `json:"age_rating"`
	CoverURL     *string   `json:"cover_url"`
	CreatedAt    time.Time `json:"created_at"`
	ChapterCount int       `json:"chapter_count"`
}

type NovelsPage struct {
	Novels     []Novel `json:"novels"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalCount int     `json:"total_count"`
	TotalPages int     `json:"total_pages"`
}

type Source struct {
	Name         string  `json:"name"`
	LogoURL      *string `json:"logo_url"`
	IsTranslator bool    `json:"is_translator"`
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
	ID              string    `json:"id"`
	DisplayName     string    `json:"display_name"`
	AvatarSeed      string    `json:"avatar_seed"`
	HasCustomAvatar bool      `json:"has_custom_avatar"`
	AvatarUpdatedAt int64     `json:"avatar_updated_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type Comment struct {
	ID                  string    `json:"id"`
	ChapterID           string    `json:"chapter_id"`
	UserID              string    `json:"user_id"`
	ContentHTML         string    `json:"content_html"`
	Status              string    `json:"status"`
	TelegramMessageID   *int64    `json:"telegram_message_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UserDisplayName     string    `json:"user_display_name,omitempty"`
	UserAvatarSeed      string    `json:"user_avatar_seed,omitempty"`
	UserHasCustomAvatar bool      `json:"user_has_custom_avatar,omitempty"`
	UserAvatarUpdatedAt int64     `json:"user_avatar_updated_at,omitempty"`
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

type UpdateProfileInput struct {
	DisplayName *string `json:"display_name,omitempty"`
}

type NovelUpdate struct {
	NovelID       string    `json:"novel_id"`
	NovelTitle    string    `json:"novel_title"`
	NovelCoverURL *string   `json:"novel_cover_url"`
	ChapterMin    int       `json:"chapter_min"`
	ChapterMax    int       `json:"chapter_max"`
	ChapterCount  int       `json:"chapter_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AppUpdate struct {
	PRNumber int       `json:"pr_number"`
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	URL      string    `json:"url"`
	MergedAt time.Time `json:"merged_at"`
}

type NovelAddition struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	YearStart   int       `json:"year_start"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CoverURL    *string   `json:"cover_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type HomeUpdateItem struct {
	Type          string
	ChapterUpdate *NovelUpdate
	AppUpdate     *AppUpdate
	NovelAddition *NovelAddition
	UpdatedAt     time.Time
}
