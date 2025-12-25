package data

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"kappalib/internal/cache"
	"kappalib/internal/database"
	"kappalib/internal/models"

	logger "github.com/ch1kulya/simple-logger"
)

//go:embed sql/chapters_get_list.sql
var queryChaptersGetList string

//go:embed sql/chapters_get_one.sql
var queryChaptersGetOne string

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
		var sourceName, sourceLogo *string

		err := database.DB.QueryRow(dbCtx, queryChaptersGetOne, id).Scan(
			&c.ID, &c.NovelID, &c.ChapterNum,
			&c.Title, &c.TitleEn, &c.Content, &c.CreatedAt,
			&sourceName, &sourceLogo,
		)
		if err != nil {
			return nil, err
		}

		if sourceName != nil {
			c.Source = &models.Source{Name: *sourceName, LogoURL: sourceLogo}
		}

		return &c, nil
	})

	if err != nil {
		return nil, err
	}
	return value.(*models.Chapter), nil
}
