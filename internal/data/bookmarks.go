package data

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/logger"
)

//go:embed sql/bookmark_chapter_info.sql
var queryBookmarkChapterInfo string

//go:embed sql/bookmarks_enrich.sql
var queryBookmarksEnrich string

//go:embed sql/user_bookmarks_get.sql
var queryUserBookmarksGet string

//go:embed sql/user_bookmarks_get_for_update.sql
var queryUserBookmarksGetForUpdate string

//go:embed sql/user_bookmarks_update.sql
var queryUserBookmarksUpdate string

const (
	defaultBookmarkCategory = "Избранное"
	maxBookmarkValueLen     = 100
	maxCategoryNameLen      = 15
	maxCategories           = 20
	maxBookmarksPerCategory = 100
)

func randomBookmarkID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "bkm_" + hex.EncodeToString(b)
}

func sanitizeBookmarkField(value, fallback string, maxLen int) string {
	value = strictPolicy.Sanitize(value)
	value = html.UnescapeString(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
	value = multiSpaceRegex.ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if runes := []rune(value); len(runes) > maxLen {
		if maxLen > 3 {
			value = string(runes[:maxLen-3]) + "..."
		} else {
			value = string(runes[:maxLen])
		}
	}
	return value
}

func sanitizeCategoryName(value, fallback string) string {
	value = strictPolicy.Sanitize(value)
	value = html.UnescapeString(value)
	value = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || (unicode.IsControl(r) && !unicode.IsSpace(r)) {
			return ' '
		}
		return r
	}, value)
	return sanitizeBookmarkField(value, fallback, maxCategoryNameLen)
}

func loadBookmarks(ctx context.Context, userID string) (map[string]models.BookmarkCategory, error) {
	var raw []byte
	err := database.DB.QueryRow(ctx, queryUserBookmarksGet, userID).Scan(&raw)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	var bookmarks map[string]models.BookmarkCategory
	if err := json.Unmarshal(raw, &bookmarks); err != nil {
		return make(map[string]models.BookmarkCategory), nil
	}
	if bookmarks == nil {
		bookmarks = make(map[string]models.BookmarkCategory)
	}
	return bookmarks, nil
}

func getBookmarkChapterInfo(ctx context.Context, chapterID string) (novelID, novelTitle, chapterTitle string, chapterNum int, err error) {
	err = database.DB.QueryRow(ctx, queryBookmarkChapterInfo, chapterID).Scan(&novelID, &novelTitle, &chapterNum, &chapterTitle)
	if err != nil {
		return "", "", "", 0, ErrChapterNotFound
	}
	return
}

func withBookmarkTx(ctx context.Context, userID string, fn func(bookmarks map[string]models.BookmarkCategory) (map[string]models.BookmarkCategory, error)) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var raw []byte
	err = tx.QueryRow(ctx, queryUserBookmarksGetForUpdate, userID).Scan(&raw)
	if err != nil {
		return ErrProfileNotFound
	}

	var bookmarks map[string]models.BookmarkCategory
	if err := json.Unmarshal(raw, &bookmarks); err != nil {
		bookmarks = make(map[string]models.BookmarkCategory)
	}
	if bookmarks == nil {
		bookmarks = make(map[string]models.BookmarkCategory)
	}

	result, err := fn(bookmarks)
	if err != nil {
		return err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, queryUserBookmarksUpdate, data, userID)
	if err != nil {
		logger.Error("Failed to save bookmarks: %v", err)
		return err
	}

	return tx.Commit(ctx)
}

func insertBookmark(bookmarks map[string]models.BookmarkCategory, categoryName string, bookmark models.Bookmark) map[string]models.BookmarkCategory {
	now := time.Now().Unix()
	cat, exists := bookmarks[categoryName]
	if !exists {
		cat = models.BookmarkCategory{
			CreatedAt: now,
			UpdatedAt: now,
			Bookmarks: []models.Bookmark{bookmark},
		}
	} else {
		cat.UpdatedAt = now
		cat.Bookmarks = append([]models.Bookmark{bookmark}, cat.Bookmarks...)
	}
	bookmarks[categoryName] = cat
	return bookmarks
}

func removeBookmarkByID(bookmarks map[string]models.BookmarkCategory, bookmarkID string) (*models.Bookmark, map[string]models.BookmarkCategory) {
	now := time.Now().Unix()
	var removed *models.Bookmark
	for catName, cat := range bookmarks {
		kept := make([]models.Bookmark, 0, len(cat.Bookmarks))
		var modified bool
		for _, b := range cat.Bookmarks {
			if b.ID == bookmarkID {
				bCopy := b
				removed = &bCopy
				modified = true
				continue
			}
			kept = append(kept, b)
		}
		if modified {
			if len(kept) == 0 {
				delete(bookmarks, catName)
			} else {
				cat.UpdatedAt = now
				cat.Bookmarks = kept
				bookmarks[catName] = cat
			}
		}
	}
	return removed, bookmarks
}

func findBookmark(bookmarks map[string]models.BookmarkCategory, bookmarkID string) (*models.Bookmark, string) {
	for catName, cat := range bookmarks {
		for _, b := range cat.Bookmarks {
			if b.ID == bookmarkID {
				bCopy := b
				return &bCopy, catName
			}
		}
	}
	return nil, ""
}

type chapterEnrichInfo struct {
	novelID    string
	novelTitle string
	chapterNum int
}

func enrichBookmarksWithDB(ctx context.Context, bookmarks map[string]models.BookmarkCategory) (map[string]models.EnrichedBookmarkCategory, error) {
	chapterIDs := make([]string, 0)
	for _, cat := range bookmarks {
		for _, b := range cat.Bookmarks {
			chapterIDs = append(chapterIDs, b.ChapterID)
		}
	}

	result := make(map[string]models.EnrichedBookmarkCategory, len(bookmarks))
	for catName, cat := range bookmarks {
		result[html.UnescapeString(catName)] = models.EnrichedBookmarkCategory{
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
			Bookmarks: make([]models.EnrichedBookmark, 0, len(cat.Bookmarks)),
		}
	}

	if len(chapterIDs) == 0 {
		return result, nil
	}

	rows, err := database.DB.Query(ctx, queryBookmarksEnrich, chapterIDs)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	infoMap := make(map[string]chapterEnrichInfo)
	for rows.Next() {
		var chID, nID, nTitle string
		var chNum int
		if err := rows.Scan(&chID, &nID, &nTitle, &chNum); err != nil {
			return result, err
		}
		infoMap[chID] = chapterEnrichInfo{
			novelID:    nID,
			novelTitle: nTitle,
			chapterNum: chNum,
		}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	for catName, cat := range bookmarks {
		enrichedList := make([]models.EnrichedBookmark, 0, len(cat.Bookmarks))
		for _, b := range cat.Bookmarks {
			eb := models.EnrichedBookmark{
				ID:        b.ID,
				ChapterID: b.ChapterID,
				Value:     html.UnescapeString(b.Value),
				CreatedAt: b.CreatedAt,
				UpdatedAt: b.UpdatedAt,
			}
			if info, ok := infoMap[b.ChapterID]; ok {
				eb.NovelID = info.novelID
				eb.NovelTitle = info.novelTitle
				eb.ChapterNum = info.chapterNum
			}
			enrichedList = append(enrichedList, eb)
		}
		result[html.UnescapeString(catName)] = models.EnrichedBookmarkCategory{
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
			Bookmarks: enrichedList,
		}
	}
	return result, nil
}

type AddBookmarkInput struct {
	ChapterID string
	Category  string
	Value     string
}

func GetUserBookmarks(ctx context.Context, userID string) (map[string]models.EnrichedBookmarkCategory, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bookmarks, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	for catName, cat := range bookmarks {
		if len(cat.Bookmarks) == 0 {
			delete(bookmarks, catName)
		}
	}

	enriched, err := enrichBookmarksWithDB(dbCtx, bookmarks)
	if err != nil {
		logger.Error("Failed to enrich bookmarks: %v", err)
		return nil, err
	}

	return enriched, nil
}

func AddBookmark(ctx context.Context, userID string, input AddBookmarkInput) (*models.EnrichedBookmark, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	novelID, novelTitle, chapterTitle, chapterNum, err := getBookmarkChapterInfo(dbCtx, input.ChapterID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	defaultValue := chapterTitle
	if defaultValue == "" {
		defaultValue = fmt.Sprintf("Глава %d", chapterNum)
	}
	category := sanitizeCategoryName(input.Category, defaultBookmarkCategory)
	value := sanitizeBookmarkField(input.Value, defaultValue, maxBookmarkValueLen)

	bookmark := models.Bookmark{
		ID:        randomBookmarkID(),
		ChapterID: input.ChapterID,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = withBookmarkTx(dbCtx, userID, func(bookmarks map[string]models.BookmarkCategory) (map[string]models.BookmarkCategory, error) {
		for _, cat := range bookmarks {
			for _, b := range cat.Bookmarks {
				if b.ChapterID == input.ChapterID {
					return nil, ErrBookmarkDuplicate
				}
			}
		}

		if _, exists := bookmarks[category]; !exists && len(bookmarks) >= maxCategories {
			return nil, ErrTooManyCategories
		}

		if cat, exists := bookmarks[category]; exists && len(cat.Bookmarks) >= maxBookmarksPerCategory {
			return nil, ErrTooManyBookmarks
		}

		return insertBookmark(bookmarks, category, bookmark), nil
	})
	if err != nil {
		return nil, err
	}

	return &models.EnrichedBookmark{
		ID:         bookmark.ID,
		NovelID:    novelID,
		ChapterID:  bookmark.ChapterID,
		ChapterNum: chapterNum,
		NovelTitle: novelTitle,
		Value:      bookmark.Value,
		CreatedAt:  bookmark.CreatedAt,
		UpdatedAt:  bookmark.UpdatedAt,
	}, nil
}

func DeleteBookmark(ctx context.Context, userID, bookmarkID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return withBookmarkTx(dbCtx, userID, func(bookmarks map[string]models.BookmarkCategory) (map[string]models.BookmarkCategory, error) {
		removed, remaining := removeBookmarkByID(bookmarks, bookmarkID)
		if removed == nil {
			return nil, ErrBookmarkNotFound
		}
		return remaining, nil
	})
}

func UpdateBookmark(ctx context.Context, userID, bookmarkID, newValue, newCategory string) (*models.EnrichedBookmark, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var updated models.Bookmark
	err := withBookmarkTx(dbCtx, userID, func(bookmarks map[string]models.BookmarkCategory) (map[string]models.BookmarkCategory, error) {
		existing, currentCategory := findBookmark(bookmarks, bookmarkID)
		if existing == nil {
			return nil, ErrBookmarkNotFound
		}

		now := time.Now().Unix()
		updated = *existing
		updated.UpdatedAt = now
		if newValue != "" {
			updated.Value = sanitizeBookmarkField(newValue, existing.Value, maxBookmarkValueLen)
		}
		targetCategory := currentCategory
		if newCategory != "" {
			targetCategory = sanitizeCategoryName(newCategory, currentCategory)
		}

		if targetCategory == currentCategory {
			cat := bookmarks[currentCategory]
			cat.UpdatedAt = now
			for j := range cat.Bookmarks {
				if cat.Bookmarks[j].ID == bookmarkID {
					cat.Bookmarks[j] = updated
				}
			}
			bookmarks[currentCategory] = cat
		} else {
			_, remaining := removeBookmarkByID(bookmarks, bookmarkID)
			if _, exists := remaining[targetCategory]; !exists && len(remaining) >= maxCategories {
				return nil, ErrTooManyCategories
			}
			if targetCat, exists := remaining[targetCategory]; exists && len(targetCat.Bookmarks) >= maxBookmarksPerCategory {
				return nil, ErrTooManyBookmarks
			}
			bookmarks = insertBookmark(remaining, targetCategory, updated)
		}

		return bookmarks, nil
	})
	if err != nil {
		return nil, err
	}

	novelID, novelTitle, _, chapterNum, err := getBookmarkChapterInfo(dbCtx, updated.ChapterID)
	if err != nil {
		return nil, err
	}

	return &models.EnrichedBookmark{
		ID:         updated.ID,
		NovelID:    novelID,
		ChapterID:  updated.ChapterID,
		ChapterNum: chapterNum,
		NovelTitle: novelTitle,
		Value:      updated.Value,
		CreatedAt:  updated.CreatedAt,
		UpdatedAt:  updated.UpdatedAt,
	}, nil
}

func resolveCategoryKey(bookmarks map[string]models.BookmarkCategory, name string) (string, bool) {
	if _, ok := bookmarks[name]; ok {
		return name, true
	}
	for key := range bookmarks {
		if html.UnescapeString(key) == name {
			return key, true
		}
	}
	return "", false
}

func DeleteCategory(ctx context.Context, userID, categoryName string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return withBookmarkTx(dbCtx, userID, func(bookmarks map[string]models.BookmarkCategory) (map[string]models.BookmarkCategory, error) {
		key, exists := resolveCategoryKey(bookmarks, categoryName)
		if !exists {
			return nil, ErrBookmarkNotFound
		}
		delete(bookmarks, key)
		return bookmarks, nil
	})
}

func RenameCategory(ctx context.Context, userID, oldName, newName string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return withBookmarkTx(dbCtx, userID, func(bookmarks map[string]models.BookmarkCategory) (map[string]models.BookmarkCategory, error) {
		key, exists := resolveCategoryKey(bookmarks, oldName)
		if !exists {
			return nil, ErrBookmarkNotFound
		}
		cat := bookmarks[key]

		sanitized := sanitizeCategoryName(newName, oldName)
		if sanitized == oldName {
			return bookmarks, nil
		}

		now := time.Now().Unix()
		delete(bookmarks, key)
		if existingTarget, ok := bookmarks[sanitized]; ok {
			if len(existingTarget.Bookmarks)+len(cat.Bookmarks) > maxBookmarksPerCategory {
				return nil, ErrTooManyBookmarks
			}
			existingTarget.UpdatedAt = now
			existingTarget.Bookmarks = append(existingTarget.Bookmarks, cat.Bookmarks...)
			bookmarks[sanitized] = existingTarget
		} else {
			cat.UpdatedAt = now
			bookmarks[sanitized] = cat
		}

		return bookmarks, nil
	})
}
