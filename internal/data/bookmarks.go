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

func loadBookmarks(ctx context.Context, userID string) ([]models.BookmarkCategory, error) {
	var raw []byte
	err := database.DB.QueryRow(ctx, `SELECT bookmarks FROM users WHERE id = $1`, userID).Scan(&raw)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	var categories []models.BookmarkCategory
	if err := json.Unmarshal(raw, &categories); err != nil {
		return []models.BookmarkCategory{}, nil
	}
	return categories, nil
}

func saveBookmarks(ctx context.Context, userID string, categories []models.BookmarkCategory) error {
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(ctx, `UPDATE users SET bookmarks = $1 WHERE id = $2`, data, userID)
	if err != nil {
		logger.Error("Failed to save bookmarks: %v", err)
	}
	return err
}

func insertBookmark(categories []models.BookmarkCategory, categoryName string, bookmark models.Bookmark) []models.BookmarkCategory {
	for i := range categories {
		if categories[i].Name == categoryName {
			categories[i].Bookmarks = append([]models.Bookmark{bookmark}, categories[i].Bookmarks...)
			return categories
		}
	}
	return append([]models.BookmarkCategory{{Name: categoryName, Bookmarks: []models.Bookmark{bookmark}}}, categories...)
}

func removeBookmarkByID(categories []models.BookmarkCategory, bookmarkID string) (*models.Bookmark, []models.BookmarkCategory) {
	var removed *models.Bookmark
	result := make([]models.BookmarkCategory, 0, len(categories))
	for _, cat := range categories {
		kept := make([]models.Bookmark, 0, len(cat.Bookmarks))
		for _, b := range cat.Bookmarks {
			if b.ID == bookmarkID {
				bCopy := b
				removed = &bCopy
				continue
			}
			kept = append(kept, b)
		}
		cat.Bookmarks = kept
		result = append(result, cat)
	}
	return removed, result
}

func findBookmark(categories []models.BookmarkCategory, bookmarkID string) (*models.Bookmark, string) {
	for _, cat := range categories {
		for _, b := range cat.Bookmarks {
			if b.ID == bookmarkID {
				bCopy := b
				return &bCopy, cat.Name
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

func GetUserBookmarks(ctx context.Context, userID string) ([]models.BookmarkCategory, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return loadBookmarks(dbCtx, userID)
}

func AddBookmark(ctx context.Context, userID string, input AddBookmarkInput) (*models.Bookmark, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	categories, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return nil, err
	}

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
		CreatedAt:     time.Now().Unix(),
	}

	categories = insertBookmark(categories, category, bookmark)

	if err := saveBookmarks(dbCtx, userID, categories); err != nil {
		return nil, err
	}
	return &bookmark, nil
}

func DeleteBookmark(ctx context.Context, userID, bookmarkID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	categories, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return err
	}

	removed, remaining := removeBookmarkByID(categories, bookmarkID)
	if removed == nil {
		return ErrBookmarkNotFound
	}

	return saveBookmarks(dbCtx, userID, remaining)
}

func UpdateBookmark(ctx context.Context, userID, bookmarkID, newName, newCategory string) (*models.Bookmark, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	categories, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	existing, currentCategory := findBookmark(categories, bookmarkID)
	if existing == nil {
		return nil, ErrBookmarkNotFound
	}

	updated := *existing
	if newName != "" {
		updated.Name = sanitizeBookmarkField(newName, existing.Name, maxBookmarkNameLen)
	}
	targetCategory := currentCategory
	if newCategory != "" {
		targetCategory = sanitizeBookmarkField(newCategory, currentCategory, maxCategoryNameLen)
	}

	if targetCategory == currentCategory {
		for i := range categories {
			if categories[i].Name != currentCategory {
				continue
			}
			for j := range categories[i].Bookmarks {
				if categories[i].Bookmarks[j].ID == bookmarkID {
					categories[i].Bookmarks[j] = updated
				}
			}
		}
	} else {
		_, remaining := removeBookmarkByID(categories, bookmarkID)
		categories = insertBookmark(remaining, targetCategory, updated)
	}

	if err := saveBookmarks(dbCtx, userID, categories); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DeleteCategory(ctx context.Context, userID, categoryName string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	categories, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return err
	}

	found := false
	result := make([]models.BookmarkCategory, 0, len(categories))
	for _, cat := range categories {
		if cat.Name == categoryName {
			found = true
			continue
		}
		result = append(result, cat)
	}

	if !found {
		return ErrBookmarkNotFound
	}

	return saveBookmarks(dbCtx, userID, result)
}

func RenameCategory(ctx context.Context, userID, oldName, newName string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	categories, err := loadBookmarks(dbCtx, userID)
	if err != nil {
		return err
	}

	sanitized := sanitizeBookmarkField(newName, oldName, maxCategoryNameLen)

	targetIdx := -1
	for i := range categories {
		if categories[i].Name == oldName {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return ErrBookmarkNotFound
	}

	for i := range categories {
		if i == targetIdx || categories[i].Name != sanitized {
			continue
		}
		categories[i].Bookmarks = append(categories[i].Bookmarks, categories[targetIdx].Bookmarks...)
		categories = append(categories[:targetIdx], categories[targetIdx+1:]...)
		return saveBookmarks(dbCtx, userID, categories)
	}

	categories[targetIdx].Name = sanitized
	return saveBookmarks(dbCtx, userID, categories)
}
