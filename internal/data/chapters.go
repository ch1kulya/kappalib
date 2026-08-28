package data

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
)

//go:embed sql/chapters_get_list.sql
var queryChaptersGetList string

//go:embed sql/chapters_get_one.sql
var queryChaptersGetOne string

//go:embed sql/updates_latest.sql
var queryUpdatesLatest string

//go:embed sql/novels_recent.sql
var queryNovelsRecent string

func GetChapters(ctx context.Context, novelID string) (*models.ChaptersList, error) {
	key := fmt.Sprintf("chapters:%s", novelID)

	value, err := cache.C.GetOrFetch(key, 5*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		rows, err := database.DB.Query(dbCtx, queryChaptersGetList, novelID)
		if err != nil {
			logger.Error("GetChapters: Failed to fetch chapters for novel %s: %v", novelID, err)
			return nil, err
		}
		defer rows.Close()

		chapters := make([]models.ChapterSummary, 0)
		for rows.Next() {
			var c models.ChapterSummary
			if err := rows.Scan(&c.ID, &c.ChapterNum, &c.Title, &c.TitleEn); err != nil {
				continue
			}
			chapters = append(chapters, c)
		}

		return &models.ChaptersList{
			Chapters: chapters,
			NovelID:  novelID,
			Count:    len(chapters),
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.ChaptersList), nil
}

func GetChapter(ctx context.Context, id string) (*models.Chapter, error) {
	key := fmt.Sprintf("chapter:%s", id)

	value, err := cache.C.GetOrFetch(key, 30*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var c models.Chapter
		var sourceName, sourceLogo, sourceLabel *string

		err := database.DB.QueryRow(dbCtx, queryChaptersGetOne, id).Scan(
			&c.ID, &c.NovelID, &c.ChapterNum,
			&c.Title, &c.TitleEn, &c.Content, &c.CreatedAt,
			&sourceName, &sourceLogo, &sourceLabel,
		)
		if err != nil {
			return nil, err
		}

		if sourceName != nil {
			label := "Источник перевода"
			if sourceLabel != nil && *sourceLabel != "" {
				label = *sourceLabel
			}
			c.Source = &models.Source{Name: *sourceName, LogoURL: sourceLogo, Label: label}
		}

		return &c, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.Chapter), nil
}

func GetLatestUpdates(ctx context.Context, limit int) ([]models.NovelUpdate, error) {
	if limit < 0 {
		limit = 0
	}
	key := fmt.Sprintf("updates:latest:%d", limit)

	value, err := cache.C.GetOrFetch(key, 3*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		rows, err := database.DB.Query(dbCtx, queryUpdatesLatest, limit)
		if err != nil {
			logger.Error("GetLatestUpdates: Failed to fetch updates: %v", err)
			return nil, err
		}
		defer rows.Close()

		updates := make([]models.NovelUpdate, 0, limit)
		for rows.Next() {
			var u models.NovelUpdate
			if err := rows.Scan(
				&u.NovelID,
				&u.NovelTitle,
				&u.NovelCoverURL,
				&u.Author,
				&u.YearStart,
				&u.Status,
				&u.Description,
				&u.ChapterMin,
				&u.ChapterMax,
				&u.ChapterCount,
				&u.UpdatedAt,
			); err != nil {
				logger.Error("GetLatestUpdates: Failed to scan row: %v", err)
				continue
			}
			updates = append(updates, u)
		}

		return updates, nil
	})
	if err != nil {
		return nil, err
	}
	return value.([]models.NovelUpdate), nil
}

func GetRecentlyAddedNovels(ctx context.Context, limit int) ([]models.NovelAddition, error) {
	if limit < 0 {
		limit = 0
	}
	key := fmt.Sprintf("novels:recent:%d", limit)

	value, err := cache.C.GetOrFetch(key, 3*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		rows, err := database.DB.Query(dbCtx, queryNovelsRecent, limit)
		if err != nil {
			logger.Error("GetRecentlyAddedNovels: Failed to fetch novels: %v", err)
			return nil, err
		}
		defer rows.Close()

		novels := make([]models.NovelAddition, 0, limit)
		for rows.Next() {
			var n models.NovelAddition
			if err := rows.Scan(
				&n.ID,
				&n.Title,
				&n.Author,
				&n.YearStart,
				&n.Status,
				&n.Description,
				&n.CoverURL,
				&n.CreatedAt,
			); err != nil {
				logger.Error("GetRecentlyAddedNovels: Failed to scan row: %v", err)
				continue
			}
			novels = append(novels, n)
		}

		return novels, nil
	})
	if err != nil {
		return nil, err
	}
	return value.([]models.NovelAddition), nil
}
