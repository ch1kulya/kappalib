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

//go:embed sql/about_stats.sql
var queryAboutStats string

//go:embed sql/about_daily_activity.sql
var queryAboutDailyActivity string

func GetAboutStats(ctx context.Context) (*models.AboutStats, error) {
	value, err := cache.C.GetOrFetch("about:stats", 15*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var s models.AboutStats

		if err := database.DB.QueryRow(dbCtx, queryAboutStats).Scan(
			&s.NovelsCount,
			&s.ChaptersCount,
			&s.SourcesCount,
			&s.CharactersCount,
			&s.TotalUsers,
		); err != nil {
			logger.Error("GetAboutStats: failed to fetch general stats: %v", err)
			return nil, err
		}

		rows, err := database.DB.Query(dbCtx, queryAboutDailyActivity)
		if err != nil {
			logger.Error("GetAboutStats: failed to fetch daily activity: %v", err)
			return nil, err
		}
		defer rows.Close()

		s.Daily = make([]models.AboutDailyPoint, 0, 31)
		for rows.Next() {
			var p models.AboutDailyPoint
			if err := rows.Scan(&p.Date, &p.TotalSeconds, &p.ActiveUsers); err != nil {
				logger.Error("GetAboutStats: failed to scan daily point: %v", err)
				continue
			}
			s.Daily = append(s.Daily, p)
		}

		return &s, nil
	})
	if err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("%+v", value))

	return value.(*models.AboutStats), nil
}
