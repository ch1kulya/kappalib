package views

import "github.com/ch1kulya/kappalib/internal/models"

type BaseProps struct {
	Title         string
	Description   string
	Canonical     string
	Version       int64
	Schema        string
	OGImage       string
	IsChapterPage bool
	IsAdult       bool
	Novel         *models.Novel
}

type LastReadWidgetData struct {
	Novel           *models.Novel
	LastChapterID   string
	NextChapterNum  int
	TotalChapters   int
	ProgressPercent int
}

type HomeProps struct {
	BaseProps
	Novels     []models.Novel
	Page       int
	TotalPages int
	SortOrder  string
	LastRead   *LastReadWidgetData
}

type NovelProps struct {
	BaseProps
	Novel           *models.Novel
	Chapters        []models.ChapterSummary
	SortOrder       string
	LastChapterID   string
	FirstChapterID  string
	ProgressPercent int
	NextChapterNum  int
	TotalChapters   int
}

type ChapterProps struct {
	BaseProps
	Novel   *models.Novel
	Chapter *models.Chapter
	PrevID  string
	NextID  string
}

type DocumentProps struct {
	BaseProps
	Content string
}

type ErrorProps struct {
	BaseProps
	ErrorCode    int
	ErrorTitle   string
	ErrorMessage string
}
