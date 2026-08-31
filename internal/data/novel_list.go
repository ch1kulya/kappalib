package data

import (
	"context"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/logger"
)

//go:embed sql/list_enrich.sql
var queryListEnrich string

//go:embed sql/list_novel_exists.sql
var queryListNovelExists string

//go:embed sql/user_list_get.sql
var queryUserListGet string

//go:embed sql/user_list_get_for_update.sql
var queryUserListGetForUpdate string

//go:embed sql/user_list_update.sql
var queryUserListUpdate string

var listStatuses = []string{"completed", "dropped", "on_hold", "planned", "rereading", "reading", "favorite"}

func isValidListStatus(status string) bool {
	for _, s := range listStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func loadList(ctx context.Context, userID string) (map[string]models.ListCategory, error) {
	var raw []byte
	err := database.DB.QueryRow(ctx, queryUserListGet, userID).Scan(&raw)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	var list map[string]models.ListCategory
	if err := json.Unmarshal(raw, &list); err != nil {
		return make(map[string]models.ListCategory), nil
	}
	if list == nil {
		list = make(map[string]models.ListCategory)
	}
	return list, nil
}

func withListTx(ctx context.Context, userID string, fn func(list map[string]models.ListCategory) (map[string]models.ListCategory, error)) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var raw []byte
	err = tx.QueryRow(ctx, queryUserListGetForUpdate, userID).Scan(&raw)
	if err != nil {
		return ErrProfileNotFound
	}

	var list map[string]models.ListCategory
	if err := json.Unmarshal(raw, &list); err != nil {
		list = make(map[string]models.ListCategory)
	}
	if list == nil {
		list = make(map[string]models.ListCategory)
	}

	result, err := fn(list)
	if err != nil {
		return err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, queryUserListUpdate, data, userID)
	if err != nil {
		logger.Error("Failed to save novel list: %v", err)
		return err
	}

	return tx.Commit(ctx)
}

func removeNovelFromAllCategories(list map[string]models.ListCategory, novelID string) (map[string]models.ListCategory, bool) {
	now := time.Now().Unix()
	found := false
	for slug, cat := range list {
		kept := make([]models.ListEntry, 0, len(cat.Novels))
		var modified bool
		for _, entry := range cat.Novels {
			if entry.ID == novelID {
				found = true
				modified = true
				continue
			}
			kept = append(kept, entry)
		}
		if modified {
			if len(kept) == 0 {
				delete(list, slug)
			} else {
				cat.UpdatedAt = now
				cat.Novels = kept
				list[slug] = cat
			}
		}
	}
	return list, found
}

func listNovelExists(ctx context.Context, novelID string) (bool, error) {
	var exists bool
	err := database.DB.QueryRow(ctx, queryListNovelExists, novelID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

type listNovelInfo struct {
	title    string
	author   string
	coverURL *string
}

func enrichListWithDB(ctx context.Context, list map[string]models.ListCategory) (map[string]models.EnrichedListCategory, error) {
	novelIDs := make([]string, 0)
	for _, cat := range list {
		for _, entry := range cat.Novels {
			novelIDs = append(novelIDs, entry.ID)
		}
	}

	result := make(map[string]models.EnrichedListCategory, len(list))
	for slug, cat := range list {
		result[slug] = models.EnrichedListCategory{
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
			Novels:    make([]models.EnrichedListEntry, 0, len(cat.Novels)),
		}
	}

	if len(novelIDs) == 0 {
		return result, nil
	}

	rows, err := database.DB.Query(ctx, queryListEnrich, novelIDs)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	infoMap := make(map[string]listNovelInfo, len(novelIDs))
	for rows.Next() {
		var id, title, author string
		var coverURL *string
		if err := rows.Scan(&id, &title, &author, &coverURL); err != nil {
			return result, err
		}
		infoMap[id] = listNovelInfo{title: title, author: author, coverURL: coverURL}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	for slug, cat := range list {
		enriched := make([]models.EnrichedListEntry, 0, len(cat.Novels))
		for _, entry := range cat.Novels {
			if info, ok := infoMap[entry.ID]; ok {
				enriched = append(enriched, models.EnrichedListEntry{
					ID:       entry.ID,
					AddedAt:  entry.AddedAt,
					Title:    info.title,
					Author:   info.author,
					CoverURL: info.coverURL,
				})
			}
		}
		result[slug] = models.EnrichedListCategory{
			CreatedAt: cat.CreatedAt,
			UpdatedAt: cat.UpdatedAt,
			Novels:    enriched,
		}
	}
	return result, nil
}

func GetUserNovelListStatus(ctx context.Context, userID, novelID string) (string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	list, err := loadList(dbCtx, userID)
	if err != nil {
		return "", err
	}

	for slug, cat := range list {
		for _, entry := range cat.Novels {
			if entry.ID == novelID {
				return slug, nil
			}
		}
	}
	return "", nil
}

func GetUserList(ctx context.Context, userID string) (map[string]models.EnrichedListCategory, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	list, err := loadList(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	for slug, cat := range list {
		if len(cat.Novels) == 0 {
			delete(list, slug)
		}
	}

	enriched, err := enrichListWithDB(dbCtx, list)
	if err != nil {
		logger.Error("Failed to enrich novel list: %v", err)
		return nil, err
	}

	return enriched, nil
}

func SetListItem(ctx context.Context, userID, novelID, status string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !isValidListStatus(status) {
		return ErrInvalidListStatus
	}

	exists, err := listNovelExists(dbCtx, novelID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNovelNotFound
	}

	return withListTx(dbCtx, userID, func(list map[string]models.ListCategory) (map[string]models.ListCategory, error) {
		if cat, ok := list[status]; ok {
			for _, entry := range cat.Novels {
				if entry.ID == novelID {
					return list, nil
				}
			}
		}

		list, _ = removeNovelFromAllCategories(list, novelID)

		now := time.Now().Unix()
		cat, exists := list[status]
		if !exists {
			list[status] = models.ListCategory{
				CreatedAt: now,
				UpdatedAt: now,
				Novels:    []models.ListEntry{{ID: novelID, AddedAt: now}},
			}
		} else {
			cat.UpdatedAt = now
			cat.Novels = append([]models.ListEntry{{ID: novelID, AddedAt: now}}, cat.Novels...)
			list[status] = cat
		}

		return list, nil
	})
}

func RemoveListItem(ctx context.Context, userID, novelID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return withListTx(dbCtx, userID, func(list map[string]models.ListCategory) (map[string]models.ListCategory, error) {
		remaining, found := removeNovelFromAllCategories(list, novelID)
		if !found {
			return nil, ErrListItemNotFound
		}
		return remaining, nil
	})
}
