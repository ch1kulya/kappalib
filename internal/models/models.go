package models

import "time"

type Novel struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	TitleEn           string    `json:"title_en"`
	Author            string    `json:"author"`
	YearStart         int       `json:"year_start"`
	YearEnd           *int      `json:"year_end"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	AgeRating         *string   `json:"age_rating"`
	CoverURL          *string   `json:"cover_url"`
	CreatedAt         time.Time `json:"created_at"`
	ChapterCount      int       `json:"chapter_count"`
	HasSelfHarm       bool      `json:"has_self_harm"`
	HasDrugUsage      bool      `json:"has_drug_usage"`
	HasSexualViolence bool      `json:"has_sexual_violence"`
	HasGraphicSex     bool      `json:"has_graphic_sex"`
	HasProfanity      bool      `json:"has_profanity"`
	AltTitles         []string  `json:"alt_titles"`
	Tags              []Tag     `json:"tags"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (n *Novel) HasContentWarnings() bool {
	return n.HasSelfHarm || n.HasSexualViolence || n.HasGraphicSex || n.HasProfanity || n.HasDrugUsage
}

type NovelSummary struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	TitleEn           string    `json:"title_en"`
	Author            string    `json:"author"`
	YearStart         int       `json:"year_start"`
	YearEnd           *int      `json:"year_end"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	AgeRating         *string   `json:"age_rating"`
	CoverURL          *string   `json:"cover_url"`
	CreatedAt         time.Time `json:"created_at"`
	ChapterCount      int       `json:"chapter_count"`
	HasSelfHarm       bool      `json:"has_self_harm"`
	HasDrugUsage      bool      `json:"has_drug_usage"`
	HasSexualViolence bool      `json:"has_sexual_violence"`
	HasGraphicSex     bool      `json:"has_graphic_sex"`
	HasProfanity      bool      `json:"has_profanity"`
}

type NovelsPage struct {
	Novels     []NovelSummary `json:"novels"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
}

type CatalogPage struct {
	Novels     []NovelSummary `json:"novels"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalCount int            `json:"total_count"`
	TotalPages int            `json:"total_pages"`
	SearchTags []string       `json:"search_tags,omitempty"`
}

type Source struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
	Label   string  `json:"label"`
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
	ID                  string    `json:"id"`
	DisplayName         string    `json:"display_name"`
	AvatarSeed          string    `json:"avatar_seed"`
	HasCustomAvatar     bool      `json:"has_custom_avatar"`
	AvatarUpdatedAt     int64     `json:"avatar_updated_at"`
	CreatedAt           time.Time `json:"created_at"`
	UnreadNotifications int       `json:"unread_notifications"`
}

type Comment struct {
	ID                  string          `json:"id"`
	ChapterID           string          `json:"chapter_id"`
	UserID              string          `json:"user_id"`
	ContentHTML         string          `json:"content_html"`
	Status              string          `json:"status"`
	TelegramMessageID   *int64          `json:"telegram_message_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UserDisplayName     string          `json:"user_display_name,omitempty"`
	UserAvatarSeed      string          `json:"user_avatar_seed,omitempty"`
	UserHasCustomAvatar bool            `json:"user_has_custom_avatar,omitempty"`
	UserAvatarUpdatedAt int64           `json:"user_avatar_updated_at,omitempty"`
	Score               int             `json:"score"`
	UserVote            int             `json:"user_vote"`
	Answers             []CommentAnswer `json:"answers,omitempty"`
	ChapterNum          int             `json:"chapter_num,omitempty"`
	NovelID             string          `json:"novel_id,omitempty"`
	NovelTitle          string          `json:"novel_title,omitempty"`
}

type CommentAnswer struct {
	ID                  string    `json:"id"`
	CommentID           string    `json:"comment_id"`
	UserID              string    `json:"user_id"`
	ContentHTML         string    `json:"content_html"`
	Status              string    `json:"status"`
	TelegramMessageID   *int64    `json:"telegram_message_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UserDisplayName     string    `json:"user_display_name,omitempty"`
	UserAvatarSeed      string    `json:"user_avatar_seed,omitempty"`
	UserHasCustomAvatar bool      `json:"user_has_custom_avatar,omitempty"`
	UserAvatarUpdatedAt int64     `json:"user_avatar_updated_at,omitempty"`
	IsNew               bool      `json:"is_new"`
}

type CommentsPage struct {
	Comments   []Comment `json:"comments"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalCount int       `json:"total_count"`
	TotalPages int       `json:"total_pages"`
}

type UserCommentsPage struct {
	Comments   []Comment `json:"comments"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalCount int       `json:"total_count"`
	TotalPages int       `json:"total_pages"`
}

type CommentStatDay struct {
	Day      string `json:"day"`
	Comments int    `json:"comments"`
	Rating   int    `json:"rating"`
	Replies  int    `json:"replies"`
}

type UserCommentStats struct {
	Days    []CommentStatDay `json:"days"`
	Total   int              `json:"total"`
	Rating  int              `json:"rating"`
	Replies int              `json:"replies"`
	Rank    int              `json:"rank"`
}

type UserAnswer struct {
	ID                    string    `json:"id"`
	CommentID             string    `json:"comment_id"`
	ContentHTML           string    `json:"content_html"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	ChapterID             string    `json:"chapter_id"`
	ChapterNum            int       `json:"chapter_num"`
	NovelID               string    `json:"novel_id"`
	NovelTitle            string    `json:"novel_title"`
	ParentUserID          string    `json:"parent_user_id"`
	ParentDisplayName     string    `json:"parent_display_name"`
	ParentAvatarSeed      string    `json:"parent_avatar_seed"`
	ParentHasCustomAvatar bool      `json:"parent_has_custom_avatar"`
	ParentAvatarUpdatedAt int64     `json:"parent_avatar_updated_at"`
	ParentContentHTML     string    `json:"parent_content_html"`
}

type CreateCommentInput struct {
	ChapterID      string `json:"chapter_id"`
	Content        string `json:"content"`
	TurnstileToken string `json:"turnstile_token"`
}

type CreateCommentAnswerInput struct {
	CommentID      string `json:"comment_id"`
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
	Author        string    `json:"author"`
	YearStart     int       `json:"year_start"`
	Status        string    `json:"status"`
	Description   string    `json:"description"`
	ChapterMin    int       `json:"chapter_min"`
	ChapterMax    int       `json:"chapter_max"`
	ChapterCount  int       `json:"chapter_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AppUpdate struct {
	PRNumber int       `json:"pr_number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
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
	Pinned        bool
}

type Announcement struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	ActionLabel *string `json:"action_label"`
	ActionURL   *string `json:"action_url"`
}
