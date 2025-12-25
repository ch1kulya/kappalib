package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"kappalib/internal/data"
	"kappalib/internal/models"
	"kappalib/internal/web/views"

	"github.com/a-h/templ"
	logger "github.com/ch1kulya/simple-logger"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	assetVersion int64
}

func NewHandler() *Handler {
	return &Handler{
		assetVersion: time.Now().Unix(),
	}
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component.Render(r.Context(), w)
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, code int, title, message string) {
	w.WriteHeader(code)
	props := views.ErrorProps{
		BaseProps: views.BaseProps{
			Title:       fmt.Sprintf("%d - %s", code, title),
			Description: message,
			Favicon:     "https://s3.kappalib.ru/favicon.ico",
			Version:     h.assetVersion,
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
	robots := `User-agent: Googlebot
Disallow: /*/chapter/*

User-agent: *
Allow: /

Sitemap: https://kappalib.ru/sitemap.xml`

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(robots))
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	items, err := data.GetSitemapData(r.Context())
	if err != nil {
		logger.Error("Sitemap generation failed: %v", err)
		http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://kappalib.ru/</loc>
		<changefreq>daily</changefreq>
		<priority>1.0</priority>
	</url>
	<url>
		<loc>https://kappalib.ru/dmca</loc>
		<changefreq>monthly</changefreq>
		<priority>0.3</priority>
	</url>
	<url>
		<loc>https://kappalib.ru/privacy</loc>
		<changefreq>monthly</changefreq>
		<priority>0.3</priority>
	</url>
	<url>
		<loc>https://kappalib.ru/copyright</loc>
		<changefreq>monthly</changefreq>
		<priority>0.3</priority>
	</url>`)

	for _, item := range items {
		dateStr := item.CreatedAt.Format("2006-01-02")
		buf.WriteString(fmt.Sprintf(`
	<url>
		<loc>https://kappalib.ru/%s</loc>
		<lastmod>%s</lastmod>
		<changefreq>weekly</changefreq>
		<priority>0.8</priority>
	</url>`, item.ID, dateStr))
	}

	buf.WriteString("\n</urlset>")

	w.Header().Set("Content-Type", "application/xml")
	w.Write(buf.Bytes())
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	sortOrder := "oldest"
	if cookie, err := r.Cookie("kappalib_catalog_sort"); err == nil {
		sortOrder = cookie.Value
	}

	dataResp, err := data.GetNovels(r.Context(), page, sortOrder)
	if err != nil {
		h.renderError(w, r, http.StatusServiceUnavailable, "Сервис временно недоступен", "Не удалось загрузить список новелл. Пожалуйста, попробуйте позже.")
		logger.Error("Failed to fetch novels home page: %v", err)
		return
	}

	canonical := "https://kappalib.ru"
	if page > 1 {
		canonical = fmt.Sprintf("https://kappalib.ru/?page=%d", page)
	}

	description := "Бесплатная библиотека веб-новелл и ранобэ. Читайте популярные веб-новеллы онлайн в хорошем переводе."

	var lastReadWidget *views.LastReadWidgetData

	if cookie, err := r.Cookie("kappalib_last_read"); err == nil {
		lastNovelID := cookie.Value
		if novel, err := data.GetNovel(r.Context(), lastNovelID); err == nil {
			progCookieName := fmt.Sprintf("kappalib_prog_%s", lastNovelID)
			lastChapterID := ""
			if progCookie, err := r.Cookie(progCookieName); err == nil {
				lastChapterID = progCookie.Value
			}

			if lastChapterID != "" {
				if chapters, err := data.GetChapters(r.Context(), lastNovelID); err == nil && len(chapters.Chapters) > 0 {
					currentChapterNum := 0
					totalChapters := chapters.Count
					if totalChapters == 0 {
						totalChapters = len(chapters.Chapters)
					}

					for _, ch := range chapters.Chapters {
						if ch.ID == lastChapterID {
							currentChapterNum = ch.ChapterNum
							break
						}
					}

					if currentChapterNum > 0 {
						progressPercent := int((float64(currentChapterNum) / float64(totalChapters)) * 100)
						if progressPercent == 0 {
							progressPercent = 1
						}
						if progressPercent > 100 {
							progressPercent = 100
						}

						lastReadWidget = &views.LastReadWidgetData{
							Novel:           novel,
							LastChapterID:   lastChapterID,
							NextChapterNum:  currentChapterNum,
							TotalChapters:   totalChapters,
							ProgressPercent: progressPercent,
						}
					}
				}
			}
		}
	}

	props := views.HomeProps{
		BaseProps: views.BaseProps{
			Title:       "Свободная библиотека веб-новелл — kappalib",
			Description: description,
			Canonical:   canonical,
			Favicon:     "https://s3.kappalib.ru/favicon.ico",
			Version:     h.assetVersion,
		},
		Novels:     dataResp.Novels,
		Page:       page,
		TotalPages: dataResp.TotalPages,
		SortOrder:  sortOrder,
		LastRead:   lastReadWidget,
	}

	h.render(w, r, views.Home(props))
}

func (h *Handler) Novel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	novel, err := data.GetNovel(r.Context(), id)
	if err != nil {
		h.renderError(w, r, http.StatusNotFound, "Новелла не найдена", "Мы не смогли найти запрашиваемую новеллу.")
		return
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

	sortOrder := "asc"
	if cookie, err := r.Cookie("kappalib_chapter_sort"); err == nil {
		if cookie.Value == "desc" {
			sortOrder = "desc"
		}
	}

	if len(chapters.Chapters) > 0 {
		if sortOrder == "desc" {
			sort.Slice(chapters.Chapters, func(i, j int) bool {
				return chapters.Chapters[i].ChapterNum > chapters.Chapters[j].ChapterNum
			})
		} else {
			sort.Slice(chapters.Chapters, func(i, j int) bool {
				return chapters.Chapters[i].ChapterNum < chapters.Chapters[j].ChapterNum
			})
		}
	}

	lastChapterID := ""
	progressPercent := 0
	nextChapterNum := 1
	totalChapters := chapters.Count

	cookieName := fmt.Sprintf("kappalib_prog_%s", id)
	if cookie, err := r.Cookie(cookieName); err == nil {
		lastChapterID = cookie.Value
	}

	if lastChapterID != "" && len(chapters.Chapters) > 0 {
		if totalChapters == 0 {
			totalChapters = len(chapters.Chapters)
		}

		currentChapterNum := 0
		for _, ch := range chapters.Chapters {
			if ch.ID == lastChapterID {
				currentChapterNum = ch.ChapterNum
				break
			}
		}

		if currentChapterNum > 0 && totalChapters > 0 {
			rawPercent := (float64(currentChapterNum) / float64(totalChapters)) * 100
			progressPercent = int(rawPercent)
			if rawPercent > 0 && progressPercent == 0 {
				progressPercent = 1
			}
			progressPercent = min(progressPercent, 100)
			nextChapterNum = currentChapterNum
		}
	}

	desc := novel.Description
	if len(desc) > 155 {
		runes := []rune(desc)
		if len(runes) > 155 {
			desc = string(runes[:155]) + "..."
		}
	}

	isAdult := false
	if novel.AgeRating != nil && *novel.AgeRating == "18+" && !isBot(r.UserAgent()) {
		isAdult = true
	}

	var ogImage string
	if novel.CoverURL != nil && *novel.CoverURL != "" {
		ogImage = *novel.CoverURL
	}

	props := views.NovelProps{
		BaseProps: views.BaseProps{
			Title:       fmt.Sprintf("%s / %s — kappalib", novel.Title, novel.TitleEn),
			Description: desc,
			Canonical:   fmt.Sprintf("https://kappalib.ru/%s", id),
			Favicon:     "https://s3.kappalib.ru/favicon.ico",
			OGImage:     ogImage,
			Version:     h.assetVersion,
			IsAdult:     isAdult,
		},
		Novel:           novel,
		Chapters:        chapters.Chapters,
		SortOrder:       sortOrder,
		LastChapterID:   lastChapterID,
		FirstChapterID:  firstChapterID,
		ProgressPercent: progressPercent,
		NextChapterNum:  nextChapterNum,
		TotalChapters:   totalChapters,
	}

	h.render(w, r, views.Novel(props))
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

	allChapters, _ := data.GetChapters(r.Context(), novelID)
	var prevID, nextID string

	if allChapters != nil && len(allChapters.Chapters) > 0 {
		// Sort by chapter number ascending
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

	title := fmt.Sprintf("Глава %d", chapter.ChapterNum)
	if chapter.Title != "Без названия" {
		title += ": " + chapter.Title
	}

	isAdult := false
	if novel.AgeRating != nil && *novel.AgeRating == "18+" && !isBot(r.UserAgent()) {
		isAdult = true
	}

	props := views.ChapterProps{
		BaseProps: views.BaseProps{
			Title:         title,
			Description:   fmt.Sprintf("Читайте %s главу новеллы %s / %s бесплатно", strconv.Itoa(chapter.ChapterNum), novel.Title, novel.TitleEn),
			Canonical:     fmt.Sprintf("https://kappalib.ru/%s/chapter/%s", novelID, chapterID),
			Favicon:       "https://s3.kappalib.ru/favicon.ico",
			Version:       h.assetVersion,
			IsChapterPage: true,
			IsAdult:       isAdult,
			Novel:         novel,
		},
		Novel:   novel,
		Chapter: chapter,
		PrevID:  prevID,
		NextID:  nextID,
	}

	h.render(w, r, views.Chapter(props))
}

func (h *Handler) StaticPage(name, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const DOCS_URL = "https://s3.kappalib.ru"

		resp, err := http.Get(fmt.Sprintf("%s/%s.html", DOCS_URL, name))
		var content string

		if err != nil || resp.StatusCode != 200 {
			content = "<div class='error'>Не удалось загрузить документ с сервера.</div>"
		} else {
			defer resp.Body.Close()
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
				Title:       title,
				Description: title,
				Canonical:   fmt.Sprintf("https://kappalib.ru/%s", name),
				Favicon:     "https://s3.kappalib.ru/favicon.ico",
				Version:     h.assetVersion,
			},
			Content: content,
		}

		h.render(w, r, views.Document(props))
	}
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	state, err := data.GetSystemStatus()

	indicator := "none"
	description := "Все системы в норме"

	if err != nil {
		logger.Warn("Failed to fetch system status: %v", err)
		indicator = "unknown"
		description = "Не удалось получить статус"
	} else {
		switch state {
		case "operational":
			indicator = "none"
			description = "Все системы в норме"
		case "degraded":
			indicator = "minor"
			description = "Наблюдаются сбои"
		case "outage":
			indicator = "major"
			description = "Серьезный сбой"
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
	}

	json.NewEncoder(w).Encode(response)
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
