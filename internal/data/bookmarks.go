package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/logger"
)

const (
	defaultBookmarkCategory = "Избранное"
	maxBookmarkNameLen      = 100
	maxCategoryNameLen      = 50
)

func randomBookmarkID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "bkm_" + hex.EncodeToString(b)
}

func sanitizeBookmarkField(value, fallback string, maxLen int) string {
	value = strictPolicy.Sanitize(value)
	value = multiSpaceRegex.ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if runes := []rune(value); len(runes) > maxLen {
		value = string(runes[:maxLen])
	}
	return value
}

func loadBookmarks(ctx context.Context, userID string) (map[string]models.BookmarkCategory, error) {
	var raw []byte
	err := database.DB.QueryRow(ctx, `SELECT bookmarks FROM users WHERE id = $1`, userID).Scan(&raw)
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

func saveBookmarks(ctx context.Context, userID string, bookmarks map[string]models.BookmarkCategory) error {
	data, err := json.Marshal(bookmarks)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(ctx, `UPDATE users SET bookmarks = $1 WHERE id = $2`, data, userID)
	if err != nil {
		logger.Error("Failed to save bookmarks: %v", err)
	}
	return err
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
			cat.UpdatedAt = now
			cat.Bookmarks = kept
			bookmarks[catName] = cat
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

type AddBookmarkInput struct {
	NovelID       string
	ChapterID     string
	ChapterNum    int
	NovelTitle    string
	NovelCoverURL string
	Category      string
	Name          string
}

func GetUserBookmarks(ctx context.Context, userID string) (map[string]models.BookmarkCategory, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return loadBookmarks(dbCtx, userID)
}

func AddBookmark(ctx context.Context, userID string, input AddBookmarkInput) (*models.Bookmark, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bookmarks, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	defaultName := fmt.Sprintf("Глава %d — %s", input.ChapterNum, input.NovelTitle)
	category := sanitizeBookmarkField(input.Category, defaultBookmarkCategory, maxCategoryNameLen)
	name := sanitizeBookmarkField(input.Name, defaultName, maxBookmarkNameLen)

	bookmark := models.Bookmark{
		ID:            randomBookmarkID(),
		NovelID:       input.NovelID,
		ChapterID:     input.ChapterID,
		ChapterNum:    input.ChapterNum,
		NovelTitle:    input.NovelTitle,
		NovelCoverURL: input.NovelCoverURL,
		Name:          name,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	bookmarks = insertBookmark(bookmarks, category, bookmark)

	if err := saveBookmarks(dbCtx, userID, bookmarks); err != nil {
		return nil, err
	}
	return &bookmark, nil
}

func DeleteBookmark(ctx context.Context, userID, bookmarkID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bookmarks, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return err
	}

	removed, remaining := removeBookmarkByID(bookmarks, bookmarkID)
	if removed == nil {
		return ErrBookmarkNotFound
	}

	return saveBookmarks(dbCtx, userID, remaining)
}

func UpdateBookmark(ctx context.Context, userID, bookmarkID, newName, newCategory string) (*models.Bookmark, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bookmarks, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	existing, currentCategory := findBookmark(bookmarks, bookmarkID)
	if existing == nil {
		return nil, ErrBookmarkNotFound
	}

	now := time.Now().Unix()
	updated := *existing
	updated.UpdatedAt = now
	if newName != "" {
		updated.Name = sanitizeBookmarkField(newName, existing.Name, maxBookmarkNameLen)
	}
	targetCategory := currentCategory
	if newCategory != "" {
		targetCategory = sanitizeBookmarkField(newCategory, currentCategory, maxCategoryNameLen)
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
		bookmarks = insertBookmark(remaining, targetCategory, updated)
	}

	if err := saveBookmarks(dbCtx, userID, bookmarks); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DeleteCategory(ctx context.Context, userID, categoryName string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bookmarks, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return err
	}

	if _, exists := bookmarks[categoryName]; !exists {
		return ErrBookmarkNotFound
	}

	delete(bookmarks, categoryName)

	return saveBookmarks(dbCtx, userID, bookmarks)
}

func RenameCategory(ctx context.Context, userID, oldName, newName string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bookmarks, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return err
	}

	cat, exists := bookmarks[oldName]
	if !exists {
		return ErrBookmarkNotFound
	}

	sanitized := sanitizeBookmarkField(newName, oldName, maxCategoryNameLen)
	if sanitized == oldName {
		return nil
	}

	now := time.Now().Unix()
	cat.UpdatedAt = now

	delete(bookmarks, oldName)
	if existingTarget, ok := bookmarks[sanitized]; ok {
		existingTarget.UpdatedAt = now
		existingTarget.Bookmarks = append(existingTarget.Bookmarks, cat.Bookmarks...)
		bookmarks[sanitized] = existingTarget
	} else {
		bookmarks[sanitized] = cat
	}

	return saveBookmarks(dbCtx, userID, bookmarks)
}
