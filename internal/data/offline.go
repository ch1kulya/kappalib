package data

import (
	"context"
	_ "embed"
	"errors"
	"sync"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
	"github.com/jackc/pgx/v5"
	"golang.org/x/time/rate"
)

//go:embed sql/chapters_get_range.sql
var queryChaptersGetRange string

const (
	OfflineChaptersMaxLimit  = 50
	offlineRequestsPerSecond = 1
	offlineBurst             = 4
	offlineVisitorExpiry     = 10 * time.Minute
)

type offlineVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type offlineRateLimiter struct {
	mu          sync.Mutex
	visitors    map[string]*offlineVisitor
	lastCleanup time.Time
}

var offlineLimiter = newOfflineRateLimiter()

func newOfflineRateLimiter() *offlineRateLimiter {
	return &offlineRateLimiter{visitors: make(map[string]*offlineVisitor)}
}

func (l *offlineRateLimiter) allow(userID string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastCleanup) > offlineVisitorExpiry {
		for id, v := range l.visitors {
			if now.Sub(v.lastSeen) > offlineVisitorExpiry {
				delete(l.visitors, id)
			}
		}
		l.lastCleanup = now
	}

	v, exists := l.visitors[userID]
	if !exists {
		v = &offlineVisitor{limiter: rate.NewLimiter(rate.Limit(offlineRequestsPerSecond), offlineBurst)}
		l.visitors[userID] = v
	}
	v.lastSeen = now
	return v.limiter.AllowN(now, 1)
}

func GetOfflineChapters(ctx context.Context, userID, novelID string, after, limit int) (*models.OfflineChaptersPage, error) {
	if limit < 1 || limit > OfflineChaptersMaxLimit {
		limit = OfflineChaptersMaxLimit
	}

	if !offlineLimiter.allow(userID, time.Now()) {
		return nil, ErrRateLimitExceeded
	}

	if _, err := GetNovel(ctx, novelID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNovelNotFound
		}
		logger.Error("GetOfflineChapters: Failed to load novel %s: %v", novelID, err)
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := database.DB.Query(dbCtx, queryChaptersGetRange, novelID, after, limit+1)
	if err != nil {
		logger.Error("GetOfflineChapters: Failed to fetch chapters for novel %s: %v", novelID, err)
		return nil, err
	}
	defer rows.Close()

	chapters := make([]models.Chapter, 0, limit+1)
	for rows.Next() {
		var c models.Chapter
		var sourceName, sourceLogo, sourceLabel *string
		if err := rows.Scan(
			&c.ID, &c.NovelID, &c.ChapterNum,
			&c.Title, &c.TitleEn, &c.Content, &c.CreatedAt,
			&sourceName, &sourceLogo, &sourceLabel,
		); err != nil {
			logger.Error("GetOfflineChapters: Failed to scan chapter for novel %s: %v", novelID, err)
			return nil, err
		}
		c.Source = buildChapterSource(sourceName, sourceLogo, sourceLabel)
		chapters = append(chapters, c)
	}
	if err := rows.Err(); err != nil {
		logger.Error("GetOfflineChapters: Rows iteration failed for novel %s: %v", novelID, err)
		return nil, err
	}

	return paginateOfflineChapters(novelID, chapters, limit), nil
}

func paginateOfflineChapters(novelID string, chapters []models.Chapter, limit int) *models.OfflineChaptersPage {
	page := &models.OfflineChaptersPage{NovelID: novelID, Chapters: chapters}
	if len(chapters) > limit {
		page.Chapters = chapters[:limit]
		next := page.Chapters[limit-1].ChapterNum
		page.NextAfter = &next
	}
	return page
}
