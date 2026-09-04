package data

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
	"github.com/jackc/pgx/v5"
)

//go:embed sql/global_announcements_get.sql
var queryGlobalAnnouncementsGet string

func GetGlobalAnnouncement(ctx context.Context) (*models.GlobalAnnouncement, error) {
	value, err := cache.C.GetOrFetch("global_announcement:random", 3*time.Minute, func() (any, error) {
		if database.DB == nil {
			return (*models.GlobalAnnouncement)(nil), nil
		}

		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var a models.GlobalAnnouncement
		err := database.DB.QueryRow(dbCtx, queryGlobalAnnouncementsGet).Scan(
			&a.ID, &a.Text, &a.URL,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				return (*models.GlobalAnnouncement)(nil), nil
			}
			return nil, err
		}

		if strings.TrimSpace(a.Text) == "" {
			return (*models.GlobalAnnouncement)(nil), nil
		}

		return &a, nil
	})
	if err != nil {
		logger.Debug("GetGlobalAnnouncement: no active global announcements: %v", err)
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	ann, ok := value.(*models.GlobalAnnouncement)
	if !ok || ann == nil {
		return nil, nil
	}
	return ann, nil
}
