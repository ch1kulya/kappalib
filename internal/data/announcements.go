package data

import (
	"context"
	_ "embed"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
)

//go:embed sql/announcements_get_random.sql
var queryAnnouncementsGetRandom string

func GetRandomAnnouncement(ctx context.Context) (*models.Announcement, error) {
	value, err := cache.C.GetOrFetch("announcement:random", 3*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var a models.Announcement
		err := database.DB.QueryRow(dbCtx, queryAnnouncementsGetRandom).Scan(
			&a.ID, &a.Title, &a.Description,
			&a.ActionLabel, &a.ActionURL,
		)
		if err != nil {
			return nil, err
		}

		return &a, nil
	})
	if err != nil {
		logger.Debug("GetRandomAnnouncement: no active announcements: %v", err)
		return nil, err
	}
	return value.(*models.Announcement), nil
}
