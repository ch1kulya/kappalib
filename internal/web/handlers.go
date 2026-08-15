package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/kappalib/internal/templates"
	"github.com/ch1kulya/kappalib/internal/web/views"

	"github.com/a-h/templ"
	"github.com/ch1kulya/logger"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	assetVersion string
}

type NovelCookieData struct {
	SortOrder       string
	LastChapterID   string
	ProgressPercent int
	NextChapterNum  int
	ReadAt          int64
}

type HomeCookieData struct {
	SortOrder      string
	LastReadWidget *views.LastReadWidgetData
}

type novelProgress struct {
	ChapterID  string `json:"chapterId"`
	ChapterNum int    `json:"chapterNum"`
	ReadAt     int64  `json:"readAt"`
}

type lastReadData struct {
	NovelID       string `json:"novelId"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	CoverURL      string `json:"coverUrl"`
	ChapterID     string `json:"chapterId"`
	ChapterNum    int    `json:"chapterNum"`
	TotalChapters int    `json:"totalChapters"`
	ReadAt        int64  `json:"readAt"`
}

type progressCookie struct {
	Novels   map[string]novelProgress `json:"novels"`
	LastRead *lastReadData            `json:"lastRead"`
}

func (h *Handler) getProgressCookie(r *http.Request) *progressCookie {
	cookie, err := r.Cookie("kappalib_progress")
	if err != nil {
		return nil
	}

	rawValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		rawValue = cookie.Value
	}

	var prog progressCookie
	if err := json.Unmarshal([]byte(rawValue), &prog); err != nil {
		return nil
	}

	return &prog
}

func (h *Handler) hasSession(r *http.Request) bool {
	_, err := r.Cookie("kpl_session")
	return err == nil
}

func NewHandler() *Handler {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return &Handler{
		assetVersion: hex.EncodeToString(b)[:8],
	}
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = component.Render(r.Context(), w)
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, code int, title, message string) {
	w.WriteHeader(code)
	props := views.ErrorProps{
		BaseProps: views.BaseProps{
			Title:          fmt.Sprintf("%d - %s", code, title),
			Description:    message,
			Version:        h.assetVersion,
			IsLoggedIn:     h.hasSession(r),
			ReaderSettings: h.getReaderSettings(r),
		},
		ErrorCode:    code,
		ErrorTitle:   title,
		ErrorMessage: message,
	}
	h.render(w, r, views.Error(props))
}

func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, r, http.StatusNotFound, "Страница не найдена", "Запрашиваемая страница не существует.")
}

func (h *Handler) RobotsTxt(w http.ResponseWriter, r *http.Request) {
	content, err := templates.RenderRobots(templates.RobotsData{
		Domain: "https://kappalib.rip",
	})
	if err != nil {
		logger.Error("Failed to render robots.txt: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(content))
}

func (h *Handler) IndexNowKey(w http.ResponseWriter, r *http.Request) {
	content, err := templates.RenderIndexNowKey(templates.IndexNowKeyData{
		Key: os.Getenv("INDEX_NOW_KEY"),
	})
	if err != nil {
		logger.Error("Failed to render IndexNow key: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(content))
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	items, err := data.GetSitemapData(r.Context())
	if err != nil {
		logger.Error("Sitemap generation failed: %v", err)
		http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	novels := make([]templates.SitemapNovel, len(items))
	for i, item := range items {
		novels[i] = templates.SitemapNovel{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
		}
	}

	staticPages := []templates.StaticPage{
		{Path: "dmca"},
		{Path: "privacy"},
		{Path: "copyright"},
		{Path: "license"},
		{Path: "terms"},
		{Path: "markdown"},
		{Path: "api/docs"},
	}

	content, err := templates.RenderSitemap(templates.SitemapData{
		Domain:      "https://kappalib.rip",
		StaticPages: staticPages,
		Novels:      novels,
	})
	if err != nil {
		logger.Error("Failed to render sitemap: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(content))
}

func (h *Handler) getReaderSettings(r *http.Request) views.ReaderSettings {
	settings := views.DefaultReaderSettings

	cookie, err := r.Cookie("kappalib_reader_settings")
	if err != nil {
		logger.Debug("Default settings. Reason: %v", err)
		return settings
	}

	rawValue := cookie.Value
	if rawValue == "" {
		logger.Warn("Empty reader settings cookie.")
		return settings
	}

	decoded, err := url.QueryUnescape(rawValue)
	if err != nil {
		logger.Warn("QueryUnescape failed: %v", err)
		decoded = rawValue
	}

	var parsed struct {
		Theme       string `json:"theme"`
		ColorScheme string `json:"colorScheme"`
		FontSize    int    `json:"fontSize"`
		FontFamily  string `json:"fontFamily"`
		Indent      int    `json:"indent"`
		Density     string `json:"density"`
		Justify     bool   `json:"justify"`
	}

	if err := json.Unmarshal([]byte(decoded), &parsed); err != nil {
		logger.Warn("Failed to parse reader settings cookie: %v", err)
		return settings
	}

	switch parsed.Theme {
	case "light", "dark", "auto":
		settings.Theme = parsed.Theme
	default:
		logger.Warn("Invalid theme value in reader settings: %s", parsed.Theme)
		return settings
	}

	if views.IsValidColorScheme(parsed.ColorScheme) {
		settings.ColorScheme = parsed.ColorScheme
	} else if parsed.ColorScheme != "" {
		logger.Warn("Invalid colorScheme value in reader settings: %s", parsed.ColorScheme)
	}

	if parsed.FontSize < 14 || parsed.FontSize > 26 {
		logger.Warn("Invalid fontSize value in reader settings: %d", parsed.FontSize)
		return settings
	}
	settings.FontSize = parsed.FontSize

	if !h.isValidFontFamily(parsed.FontFamily) {
		logger.Warn("Invalid fontFamily value in reader settings: %s", parsed.FontFamily)
		return settings
	}
	settings.FontFamily = parsed.FontFamily

	if parsed.Indent < 0 || parsed.Indent > 3 {
		logger.Warn("Invalid indent value in reader settings: %d", parsed.Indent)
		return settings
	}
	settings.Indent = parsed.Indent

	switch parsed.Density {
	case "compact", "normal", "relaxed":
		settings.Density = parsed.Density
	default:
		logger.Warn("Invalid density value in reader settings: %s", parsed.Density)
		return settings
	}

	settings.Justify = parsed.Justify

	logger.Debug("User settings applied successful.")

	return settings
}

func (h *Handler) isValidFontFamily(family string) bool {
	for _, f := range views.FontOptions {
		if f.Value == family {
			return true
		}
	}
	return false
}

func (h *Handler) getNovelCookieData(r *http.Request, novelID string, chapters []models.ChapterSummary, totalChapters int) NovelCookieData {
	result := NovelCookieData{
		SortOrder:       "asc",
		LastChapterID:   "",
		ProgressPercent: 0,
		NextChapterNum:  1,
	}

	if cookie, err := r.Cookie("kappalib_chapter_sort"); err == nil {
		if cookie.Value == "desc" {
			result.SortOrder = "desc"
		}
	}

	prog := h.getProgressCookie(r)
	if prog == nil || prog.Novels == nil {
		return result
	}

	novelProg, ok := prog.Novels[novelID]
	if !ok {
		return result
	}

	result.LastChapterID = novelProg.ChapterID
	result.ReadAt = novelProg.ReadAt

	if result.LastChapterID != "" && len(chapters) > 0 {
		if totalChapters == 0 {
			totalChapters = len(chapters)
		}
		currentChapterNum := 0
		for _, ch := range chapters {
			if ch.ID == result.LastChapterID {
				currentChapterNum = ch.ChapterNum
				break
			}
		}
		if currentChapterNum > 0 && totalChapters > 0 {
			rawPercent := (float64(currentChapterNum) / float64(totalChapters)) * 100
			result.ProgressPercent = int(rawPercent)
			if rawPercent > 0 && result.ProgressPercent == 0 {
				result.ProgressPercent = 1
			}
			result.ProgressPercent = min(result.ProgressPercent, 100)
			result.NextChapterNum = currentChapterNum
		}
	}

	return result
}

func (h *Handler) getHomeCookieData(r *http.Request) HomeCookieData {
	result := HomeCookieData{
		SortOrder:      "popular",
		LastReadWidget: nil,
	}

	if cookie, err := r.Cookie("kappalib_catalog_sort"); err == nil {
		result.SortOrder = cookie.Value
	}

	prog := h.getProgressCookie(r)
	if prog == nil || prog.LastRead == nil {
		return result
	}

	lastRead := prog.LastRead
	if lastRead.TotalChapters <= 0 || lastRead.ChapterNum <= 0 {
		return result
	}

	progressPercent := int((float64(lastRead.ChapterNum) / float64(lastRead.TotalChapters)) * 100)
	if progressPercent == 0 {
		progressPercent = 1
	}
	if progressPercent > 100 {
		progressPercent = 100
	}

	coverURL := lastRead.CoverURL
	result.LastReadWidget = &views.LastReadWidgetData{
		Novel: &models.Novel{
			ID:       lastRead.NovelID,
			Title:    lastRead.Title,
			Author:   lastRead.Author,
			CoverURL: &coverURL,
		},
		LastChapterID:   lastRead.ChapterID,
		NextChapterNum:  lastRead.ChapterNum,
		TotalChapters:   lastRead.TotalChapters,
		ProgressPercent: progressPercent,
		ReadAt:          lastRead.ReadAt,
	}

	return result
}

func (h *Handler) Novel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	novel, err := data.GetNovel(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Новелла не найдена", "Мы не смогли найти запрашиваемую новеллу.")
		return
	}

	if r.Header.Get("Purpose") != "prefetch" && r.Header.Get("Sec-Purpose") != "prefetch" {
		go data.IncrementNovelViews(context.Background(), id)
	}

	chapters, err := data.GetChapters(r.Context(), id)
	if err != nil || chapters == nil {
		chapters = &models.ChaptersList{Chapters: []models.ChapterSummary{}}
	}

	firstChapterID := ""
	if len(chapters.Chapters) > 0 {
		minNum := chapters.Chapters[0].ChapterNum
		firstChapterID = chapters.Chapters[0].ID
		for _, ch := range chapters.Chapters {
			if ch.ChapterNum < minNum {
				minNum = ch.ChapterNum
				firstChapterID = ch.ID
			}
		}
	}

	cookieData := h.getNovelCookieData(r, id, chapters.Chapters, novel.ChapterCount)

	if len(chapters.Chapters) > 0 {
		if cookieData.SortOrder == "desc" {
			sort.Slice(chapters.Chapters, func(i, j int) bool {
				return chapters.Chapters[i].ChapterNum > chapters.Chapters[j].ChapterNum
			})
		} else {
			sort.Slice(chapters.Chapters, func(i, j int) bool {
				return chapters.Chapters[i].ChapterNum < chapters.Chapters[j].ChapterNum
			})
		}
	}

	desc := fmt.Sprintf("%d глав · %s", novel.ChapterCount, novel.Description)
	if len([]rune(desc)) > 155 {
		desc = string([]rune(desc)[:155]) + "..."
	}

	isAdult := false
	isSevere := false
	if !isBot(r.UserAgent()) {
		if novel.AgeRating != nil && *novel.AgeRating == "18+" {
			isAdult = true
		}
		if novel.HasContentWarnings() {
			isSevere = true
		}
	}

	var ogImage string
	if novel.CoverURL != nil && *novel.CoverURL != "" {
		ogImage = *novel.CoverURL
	}

	canonical := fmt.Sprintf("https://kappalib.rip/%s", id)
	title := fmt.Sprintf("%s / %s — kappalib", novel.Title, novel.TitleEn)

	schemaNovel := templates.SchemaNovel{
		ID:       novel.ID,
		Title:    novel.Title,
		TitleEn:  novel.TitleEn,
		Author:   novel.Author,
		Status:   novel.Status,
		CoverURL: views.DerefStr(novel.CoverURL),
	}

	schema, err := templates.RenderSchemaNovel(templates.SchemaNovelData{
		Domain:      "https://kappalib.rip",
		Canonical:   canonical,
		Title:       title,
		Description: desc,
		Novel:       schemaNovel,
	})
	if err != nil {
		logger.Warn("Failed to render schema for novel page: %v", err)
		schema = ""
	}

	var prefetchURL string
	if cookieData.LastChapterID != "" {
		prefetchURL = fmt.Sprintf("/%s/chapter/%s", id, cookieData.LastChapterID)
	} else if firstChapterID != "" {
		prefetchURL = fmt.Sprintf("/%s/chapter/%s", id, firstChapterID)
	}

	props := views.NovelProps{
		BaseProps: views.BaseProps{
			Title:          title,
			Description:    desc,
			Canonical:      canonical,
			OGImage:        ogImage,
			Version:        h.assetVersion,
			IsAdult:        isAdult,
			IsLoggedIn:     h.hasSession(r),
			IsSevere:       isSevere,
			Schema:         schema,
			PrefetchURL:    prefetchURL,
			ReaderSettings: h.getReaderSettings(r),
		},
		Novel:           novel,
		Chapters:        chapters.Chapters,
		SortOrder:       cookieData.SortOrder,
		LastChapterID:   cookieData.LastChapterID,
		FirstChapterID:  firstChapterID,
		ProgressPercent: cookieData.ProgressPercent,
		NextChapterNum:  cookieData.NextChapterNum,
		TotalChapters:   novel.ChapterCount,
	}

	h.render(w, r, views.Novel(props))
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	cookieData := h.getHomeCookieData(r)

	dataResp, err := data.GetNovels(r.Context(), page, cookieData.SortOrder)
	if err != nil {
		h.renderError(w, r, http.StatusServiceUnavailable, "Сервис временно недоступен", "Не удалось загрузить список новелл. Пожалуйста, попробуйте позже.")
		logger.Error("Failed to fetch novels home page: %v", err)
		return
	}

	latestUpdates, pinnedAppUpdate, _ := buildFeed(r.Context(), true)

	canonical := "https://kappalib.rip"
	if page > 1 {
		canonical = fmt.Sprintf("https://kappalib.rip/?page=%d", page)
	}

	description := "Бесплатная библиотека веб-новелл и ранобэ. Читайте популярные веб-новеллы онлайн в хорошем переводе."
	title := "Открытая коллекция веб-новелл — kappalib"

	if page > 1 {
		title = fmt.Sprintf("Открытая коллекция веб-новелл, страница %d — kappalib", page)
		description = fmt.Sprintf("Страница %d. Бесплатная библиотека веб-новелл и ранобэ. Читайте популярные веб-новеллы онлайн в хорошем переводе.", page)
	}

	schema, err := templates.RenderSchemaWebsite(templates.SchemaWebsiteData{
		Domain:      "https://kappalib.rip",
		Canonical:   canonical,
		Title:       title,
		Description: description,
	})
	if err != nil {
		logger.Warn("Failed to render schema for home page: %v", err)
		schema = ""
	}

	props := views.HomeProps{
		BaseProps: views.BaseProps{
			Title:          title,
			Description:    description,
			Canonical:      canonical,
			Version:        h.assetVersion,
			IsLoggedIn:     h.hasSession(r),
			Schema:         schema,
			ReaderSettings: h.getReaderSettings(r),
		},
		Novels:          dataResp.Novels,
		Page:            page,
		TotalPages:      dataResp.TotalPages,
		SortOrder:       cookieData.SortOrder,
		LastRead:        cookieData.LastReadWidget,
		LatestUpdates:   latestUpdates,
		PinnedAppUpdate: pinnedAppUpdate,
	}

	h.render(w, r, views.Home(props))
}

func (h *Handler) Chapter(w http.ResponseWriter, r *http.Request) {
	novelID := chi.URLParam(r, "id")
	chapterID := chi.URLParam(r, "chapterId")

	novel, err := data.GetNovel(r.Context(), novelID)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Новелла не найдена", "Не удалось найти новеллу для этой главы.")
		return
	}

	chapter, err := data.GetChapter(r.Context(), chapterID)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Глава не найдена", "Запрашиваемая глава не существует или была удалена.")
		return
	}

	if r.Header.Get("Purpose") != "prefetch" && r.Header.Get("Sec-Purpose") != "prefetch" {
		go data.IncrementNovelViews(context.Background(), novelID)
	}

	allChapters, _ := data.GetChapters(r.Context(), novelID)
	var prevID, nextID string

	if allChapters != nil && len(allChapters.Chapters) > 0 {
		sort.Slice(allChapters.Chapters, func(i, j int) bool {
			return allChapters.Chapters[i].ChapterNum < allChapters.Chapters[j].ChapterNum
		})

		for i, ch := range allChapters.Chapters {
			if ch.ID == chapterID {
				if i > 0 {
					prevID = allChapters.Chapters[i-1].ID
				}
				if i < len(allChapters.Chapters)-1 {
					nextID = allChapters.Chapters[i+1].ID
				}
				break
			}
		}
	}

	chapterTitle := fmt.Sprintf("Глава %d", chapter.ChapterNum)
	if chapter.Title != "Без названия" {
		chapterTitle += ": " + chapter.Title
	}

	isAdult := false
	isSevere := false
	if !isBot(r.UserAgent()) {
		if novel.AgeRating != nil && *novel.AgeRating == "18+" {
			isAdult = true
		}
		if novel.HasContentWarnings() {
			isSevere = true
		}
	}

	canonical := fmt.Sprintf("https://kappalib.rip/%s/chapter/%s", novelID, chapterID)
	description := fmt.Sprintf("Читайте %s главу новеллы %s / %s бесплатно", strconv.Itoa(chapter.ChapterNum), novel.Title, novel.TitleEn)

	schemaNovel := templates.SchemaNovel{
		ID:       novel.ID,
		Title:    novel.Title,
		TitleEn:  novel.TitleEn,
		Author:   novel.Author,
		CoverURL: views.DerefStr(novel.CoverURL),
	}

	schema, err := templates.RenderSchemaChapter(templates.SchemaChapterData{
		Domain:       "https://kappalib.rip",
		Canonical:    canonical,
		Description:  description,
		ChapterTitle: chapterTitle,
		ChapterNum:   chapter.ChapterNum,
		Novel:        schemaNovel,
	})
	if err != nil {
		logger.Warn("Failed to render schema for chapter page: %v", err)
		schema = ""
	}

	var prefetchURL string
	if nextID != "" {
		prefetchURL = fmt.Sprintf("/%s/chapter/%s", novelID, nextID)
	}

	announcement, _ := data.GetRandomAnnouncement(r.Context())

	props := views.ChapterProps{
		BaseProps: views.BaseProps{
			Title:          chapterTitle,
			Description:    description,
			Canonical:      canonical,
			Version:        h.assetVersion,
			IsChapterPage:  true,
			IsAdult:        isAdult,
			IsLoggedIn:     h.hasSession(r),
			IsSevere:       isSevere,
			Novel:          novel,
			Schema:         schema,
			PrefetchURL:    prefetchURL,
			ReaderSettings: h.getReaderSettings(r),
		},
		Novel:         novel,
		Chapter:       chapter,
		PrevID:        prevID,
		NextID:        nextID,
		TotalChapters: novel.ChapterCount,
		Announcement:  announcement,
	}

	h.render(w, r, views.Chapter(props))
}

func (h *Handler) Updates(w http.ResponseWriter, r *http.Request) {
	allUpdates, pinnedAppUpdate, _ := buildFeed(r.Context(), false)

	title := "Обновления — kappalib"
	description := "История обновлений новелл и сайта."

	props := views.UpdatesProps{
		BaseProps: views.BaseProps{
			Title:          title,
			Description:    description,
			Canonical:      "https://kappalib.rip/updates",
			Version:        h.assetVersion,
			IsLoggedIn:     h.hasSession(r),
			ReaderSettings: h.getReaderSettings(r),
		},
		Updates:         allUpdates,
		PinnedAppUpdate: pinnedAppUpdate,
	}

	h.render(w, r, views.Updates(props))
}

func (h *Handler) StaticPage(name, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const DOCS_URL = "https://s3.kappalib.rip"

		resp, err := http.Get(fmt.Sprintf("%s/%s.html", DOCS_URL, name))
		var content string

		if err != nil || resp.StatusCode != 200 {
			content = "<div class='error'>Не удалось загрузить документ с сервера.</div>"
		} else {
			defer func() { _ = resp.Body.Close() }()
			bodyBytes, _ := io.ReadAll(resp.Body)

			bodyStr := string(bodyBytes)
			if start := strings.Index(bodyStr, "<body>"); start != -1 {
				if end := strings.Index(bodyStr, "</body>"); end != -1 {
					bodyStr = bodyStr[start+6 : end]
				}
			}
			content = bodyStr
		}

		props := views.DocumentProps{
			BaseProps: views.BaseProps{
				Title:          title,
				Description:    title,
				Canonical:      fmt.Sprintf("https://kappalib.rip/%s", name),
				Version:        h.assetVersion,
				IsLoggedIn:     h.hasSession(r),
				ReaderSettings: h.getReaderSettings(r),
			},
			Content: content,
		}

		h.render(w, r, views.Document(props))
	}
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	status, err := data.GetSystemStatus()

	var indicator string
	var description string

	if err != nil {
		logger.Warn("Failed to fetch system status: %v", err)
		indicator = "unknown"
		description = "Не удалось получить статус"
	} else {
		switch status.Impact {
		case "operational":
			indicator = "none"
			description = "Все системы в норме"
		case "degraded_performance":
			indicator = "minor"
			description = "Наблюдаются сбои"
		case "partial_outage":
			indicator = "major"
			description = "Серьезный сбой"
		case "major_outage":
			indicator = "critical"
			description = "Критический сбой"
		case "maintenance":
			indicator = "maintenance"
			description = "Технические работы"
		default:
			indicator = "unknown"
			description = "Статус неизвестен"
		}
	}

	response := map[string]any{
		"status": map[string]string{
			"indicator":   indicator,
			"description": description,
		},
		"availability": status.Availability,
		"updated_at":   status.UpdatedAt,
	}

	_ = json.NewEncoder(w).Encode(response)
}

func isBot(ua string) bool {
	ua = strings.ToLower(ua)
	bots := []string{"googlebot", "yandex", "bingbot", "duckduckbot", "baiduspider", "slurp", "facebookexternalhit", "twitterbot"}
	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	isPartial := r.Header.Get("X-Partial") == "true"

	page := 1
	if isPartial {
		if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
			page = p
		}
	}

	searchQuery := r.URL.Query().Get("search")

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder == "" {
		if searchQuery != "" {
			sortOrder = "relevance"
		} else if cookie, err := r.Cookie("kappalib_catalog_sort"); err == nil {
			sortOrder = cookie.Value
		} else {
			sortOrder = "popular"
		}
	}

	dataResp, err := data.GetCatalogNovels(r.Context(), page, sortOrder, searchQuery)
	if err != nil {
		h.renderError(w, r, http.StatusServiceUnavailable, "Сервис временно недоступен", "Не удалось загрузить каталог. Пожалуйста, попробуйте позже.")
		logger.Error("Failed to fetch catalog: %v", err)
		return
	}

	title := "Каталог — kappalib"
	description := "Полный каталог веб-новелл и ранобэ. Читайте популярные веб-новеллы онлайн."
	if searchQuery != "" {
		title = fmt.Sprintf("Поиск: %s — kappalib", searchQuery)
		description = fmt.Sprintf("Результаты поиска по запросу «%s» в библиотеке веб-новелл.", searchQuery)
	}

	canonical := "https://kappalib.rip/catalog"
	if page > 1 {
		canonical = fmt.Sprintf("https://kappalib.rip/catalog?page=%d", page)
	}

	props := views.CatalogProps{
		BaseProps: views.BaseProps{
			Title:          title,
			Description:    description,
			Canonical:      canonical,
			Version:        h.assetVersion,
			IsLoggedIn:     h.hasSession(r),
			ReaderSettings: h.getReaderSettings(r),
		},
		Novels:      dataResp.Novels,
		Page:        page,
		TotalPages:  dataResp.TotalPages,
		TotalCount:  dataResp.TotalCount,
		SortOrder:   sortOrder,
		SearchQuery: searchQuery,
		SearchTags:  dataResp.SearchTags,
		IsPartial:   isPartial,
	}

	h.render(w, r, views.Catalog(props))
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	props := views.BaseProps{
		Title:          "История чтения — kappalib",
		Description:    "История прочитанных веб-новелл и ранобэ.",
		Canonical:      "https://kappalib.rip/history",
		Version:        h.assetVersion,
		IsLoggedIn:     h.hasSession(r),
		ReaderSettings: h.getReaderSettings(r),
	}
	h.render(w, r, views.History(props))
}

func (h *Handler) MyComments(w http.ResponseWriter, r *http.Request) {
	props := views.BaseProps{
		Title:          "Мои комментарии — kappalib",
		Description:    "Ваши комментарии к веб-новеллам.",
		Canonical:      "https://kappalib.rip/comments",
		Version:        h.assetVersion,
		IsLoggedIn:     h.hasSession(r),
		ReaderSettings: h.getReaderSettings(r),
	}
	h.render(w, r, views.MyComments(props))
}
