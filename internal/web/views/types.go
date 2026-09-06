package views

import "github.com/ch1kulya/kappalib/internal/models"

type BaseProps struct {
	Title              string
	Description        string
	Canonical          string
	Version            string
	Schema             string
	OGImage            string
	IsHomePage         bool
	IsChapterPage      bool
	IsAdult            bool
	IsLoggedIn         bool
	Novel              *models.Novel
	PrefetchURL        string
	ReaderSettings     ReaderSettings
	IsSevere           bool
	IsMyCommentsPage   bool
	GlobalAnnouncement *models.GlobalAnnouncement
}

type LastReadWidgetData struct {
	Novel           *models.Novel
	LastChapterID   string
	NextChapterNum  int
	TotalChapters   int
	ProgressPercent int
	ReadAt          int64
}

type HomeProps struct {
	BaseProps
	Novels          []models.NovelSummary
	Page            int
	TotalPages      int
	SortOrder       string
	LastRead        *LastReadWidgetData
	LatestUpdates   []models.HomeUpdateItem
	PinnedAppUpdate *models.AppUpdate
	EditorsPick     []models.NovelSummary
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
	ListStatus      string
}

type ChapterProps struct {
	BaseProps
	Novel         *models.Novel
	Chapter       *models.Chapter
	PrevID        string
	NextID        string
	TotalChapters int
	Announcement  *models.Announcement
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

type ReaderSettings struct {
	Theme        string
	ColorScheme  string
	FontSize     int
	FontFamily   string
	Indent       int
	Density      string
	Justify      bool
	ShowComments bool
}

type FontOption struct {
	Value  string
	Label  string
	Family string
}

type ColorSchemeOption struct {
	Value string
	Label string
}

type ListStatusOption struct {
	Slug  string
	Label string
}

type CatalogProps struct {
	BaseProps
	Novels      []models.NovelSummary
	Page        int
	TotalPages  int
	TotalCount  int
	SortOrder   string
	SearchQuery string
	SearchTags  []string
	IsPartial   bool
}

type UpdatesProps struct {
	BaseProps
	Updates         []models.HomeUpdateItem
	PinnedAppUpdate *models.AppUpdate
}
