package data

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
)

//go:embed sql/novels_sitemap.sql
var queryNovelsSitemap string

//go:embed sql/novels_count.sql
var queryNovelsCount string

//go:embed sql/novels_search.sql
var queryNovelsSearch string

//go:embed sql/novels_get_one.sql
var queryNovelsGetOne string

//go:embed sql/novels_increment_views.sql
var queryNovelsIncrementViews string

//go:embed sql/novels_catalog_search_count.sql
var queryNovelsCatalogSearchCount string

func GetNovel(ctx context.Context, id string) (*models.Novel, error) {
	key := fmt.Sprintf("novel:%s", id)

	value, err := cache.C.GetOrFetch(key, 10*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var n models.Novel
		err := database.DB.QueryRow(dbCtx, queryNovelsGetOne, id).Scan(
			&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount,
		)
		if err != nil {
			return nil, err
		}
		return &n, nil
	})

	if err != nil {
		return nil, err
	}
	return value.(*models.Novel), nil
}

func GetNovels(ctx context.Context, page int, sort string) (*models.NovelsPage, error) {
	key := fmt.Sprintf("novels:page:%d:sort:%s", page, sort)
	pageSize := 12
	offset := (page - 1) * pageSize

	value, err := cache.C.GetOrFetch(key, 5*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var totalCount int
		if err := database.DB.QueryRow(dbCtx, queryNovelsCount).Scan(&totalCount); err != nil {
			logger.Error("GetNovels: Failed to count novels: %v", err)
			return nil, err
		}

		if totalCount > 0 && offset >= totalCount {
			return &models.NovelsPage{
				Novels:     []models.Novel{},
				Page:       page,
				PageSize:   pageSize,
				TotalCount: totalCount,
				TotalPages: (totalCount + pageSize - 1) / pageSize,
			}, nil
		}

		baseQuery := `SELECT id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at FROM novels WHERE age_rating IS DISTINCT FROM '19+'`

		var orderByClause string
		switch sort {
		case "newest":
			orderByClause = "ORDER BY year_start DESC, title ASC"
		case "large":
			orderByClause = "ORDER BY chapters_count DESC, title ASC"
		case "small":
			orderByClause = "ORDER BY chapters_count ASC, title ASC"
		case "alphabet":
			orderByClause = "ORDER BY regexp_replace(lower(title), '[^а-яё]', '', 'g') ASC"
		case "created":
			orderByClause = "ORDER BY created_at DESC"
		case "oldest":
			orderByClause = "ORDER BY year_start ASC, title ASC"
		case "popular":
			fallthrough
		default:
			orderByClause = "ORDER BY views_count DESC, title ASC"
		}

		finalQuery := fmt.Sprintf("%s %s LIMIT $1 OFFSET $2", baseQuery, orderByClause)

		rows, err := database.DB.Query(dbCtx, finalQuery, pageSize, offset)
		if err != nil {
			logger.Error("GetNovels: Failed to query novels with sort '%s': %v", sort, err)
			return nil, err
		}
		defer rows.Close()

		novels := make([]models.Novel, 0)
		for rows.Next() {
			var n models.Novel
			if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
				&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
				&n.AgeRating, &n.CoverURL, &n.CreatedAt); err != nil {
				logger.Warn("GetNovels: Row scan error: %v", err)
				continue
			}
			novels = append(novels, n)
		}

		totalPages := (totalCount + pageSize - 1) / pageSize
		return &models.NovelsPage{
			Novels:     novels,
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	})

	if err != nil {
		return nil, err
	}
	return value.(*models.NovelsPage), nil
}

func SearchNovels(ctx context.Context, query string) ([]models.Novel, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []models.Novel{}, nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := database.DB.Query(dbCtx, queryNovelsSearch, query)
	if err != nil {
		logger.Error("SearchNovels: Query failed for '%s': %v", query, err)
		return nil, err
	}
	defer rows.Close()

	novels := make([]models.Novel, 0)
	for rows.Next() {
		var n models.Novel
		var relevance float64
		if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt, &relevance); err != nil {
			continue
		}
		novels = append(novels, n)
	}

	return novels, nil
}

func GetSitemapData(ctx context.Context) ([]models.SitemapItem, error) {
	key := "sitemap_data"

	value, err := cache.C.GetOrFetch(key, 1*time.Hour, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		rows, err := database.DB.Query(dbCtx, queryNovelsSitemap)
		if err != nil {
			logger.Error("Sitemap: Failed to fetch data: %v", err)
			return nil, err
		}
		defer rows.Close()

		items := make([]models.SitemapItem, 0)
		for rows.Next() {
			var item models.SitemapItem
			if err := rows.Scan(&item.ID, &item.CreatedAt); err != nil {
				logger.Warn("Sitemap: Row scan error: %v", err)
				continue
			}
			items = append(items, item)
		}

		return items, nil
	})

	if err != nil {
		return nil, err
	}
	return value.([]models.SitemapItem), nil
}

func IncrementNovelViews(ctx context.Context, novelID string) {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if _, err := database.DB.Exec(dbCtx, queryNovelsIncrementViews, novelID); err != nil {
		logger.Warn("Failed to increment views for novel %s: %v", novelID, err)
	}
}

func GetCatalogNovels(ctx context.Context, page int, sort string, search string) (*models.NovelsPage, error) {
	pageSize := 24
	offset := (page - 1) * pageSize

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if search != "" {
		var totalCount int
		if err := database.DB.QueryRow(dbCtx, queryNovelsCatalogSearchCount, search).Scan(&totalCount); err != nil {
			logger.Error("GetCatalogNovels: Failed to count search results: %v", err)
			return nil, err
		}

		if totalCount > 0 && offset >= totalCount {
			return &models.NovelsPage{
				Novels:     []models.Novel{},
				Page:       page,
				PageSize:   pageSize,
				TotalCount: totalCount,
				TotalPages: (totalCount + pageSize - 1) / pageSize,
			}, nil
		}

		var orderByClause string
		switch sort {
		case "newest":
			orderByClause = "ORDER BY year_start DESC, title ASC"
		case "large":
			orderByClause = "ORDER BY chapters_count DESC, title ASC"
		case "small":
			orderByClause = "ORDER BY chapters_count ASC, title ASC"
		case "alphabet":
			orderByClause = "ORDER BY regexp_replace(lower(title), '[^а-яё]', '', 'g') ASC"
		case "created":
			orderByClause = "ORDER BY created_at DESC"
		case "oldest":
			orderByClause = "ORDER BY year_start ASC, title ASC"
		case "popular":
			orderByClause = "ORDER BY views_count DESC, title ASC"
		default:
			orderByClause = "ORDER BY relevance DESC, created_at DESC"
		}

		searchQuery := fmt.Sprintf(`
			WITH norm_query AS (
				SELECT
					lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) AS q,
					'%%' || lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) || '%%' AS q_like
			),
			candidates AS (
				SELECT n.*, nq.q, nq.q_like
				FROM novels AS n, norm_query AS nq
				WHERE
					n.title_norm ILIKE nq.q_like
					OR n.title_en_norm ILIKE nq.q_like
					OR n.author_norm ILIKE nq.q_like
					OR nq.q %% n.title_norm
					OR nq.q %% n.title_en_norm
					OR nq.q %% n.author_norm
					OR nq.q <%% n.title_norm
					OR nq.q <%% n.title_en_norm
			),
			scored AS (
				SELECT
					id, title, title_en, author, year_start, year_end, status,
					description, age_rating, cover_url, created_at, chapters_count, views_count,
					(
						CASE WHEN title_norm = q THEN 100 ELSE 0 END
						+ CASE WHEN title_en_norm = q THEN 100 ELSE 0 END
						+ CASE WHEN title_norm LIKE q || '%%' THEN 50 ELSE 0 END
						+ CASE WHEN title_en_norm LIKE q || '%%' THEN 50 ELSE 0 END
						+ CASE WHEN title_norm ILIKE q_like THEN 25 ELSE 0 END
						+ CASE WHEN title_en_norm ILIKE q_like THEN 20 ELSE 0 END
						+ CASE WHEN author_norm ILIKE q_like THEN 15 ELSE 0 END
						+ similarity(q, title_norm) * 30
						+ similarity(q, title_en_norm) * 25
						+ similarity(q, author_norm) * 15
						+ word_similarity(q, title_norm) * 20
						+ word_similarity(q, title_en_norm) * 15
					) AS relevance
				FROM candidates
			)
			SELECT id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at
			FROM scored
			%s
			LIMIT $2 OFFSET $3
		`, orderByClause)

		rows, err := database.DB.Query(dbCtx, searchQuery, search, pageSize, offset)
		if err != nil {
			logger.Error("GetCatalogNovels: Failed to search novels: %v", err)
			return nil, err
		}
		defer rows.Close()

		novels := make([]models.Novel, 0)
		for rows.Next() {
			var n models.Novel
			if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
				&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
				&n.AgeRating, &n.CoverURL, &n.CreatedAt); err != nil {
				logger.Warn("GetCatalogNovels: Row scan error: %v", err)
				continue
			}
			novels = append(novels, n)
		}

		totalPages := (totalCount + pageSize - 1) / pageSize
		return &models.NovelsPage{
			Novels:     novels,
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	}

	var totalCount int
	if err := database.DB.QueryRow(dbCtx, queryNovelsCount).Scan(&totalCount); err != nil {
		logger.Error("GetCatalogNovels: Failed to count novels: %v", err)
		return nil, err
	}

	if totalCount > 0 && offset >= totalCount {
		return &models.NovelsPage{
			Novels:     []models.Novel{},
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: (totalCount + pageSize - 1) / pageSize,
		}, nil
	}

	var orderByClause string
	switch sort {
	case "newest":
		orderByClause = "ORDER BY year_start DESC, title ASC"
	case "large":
		orderByClause = "ORDER BY chapters_count DESC, title ASC"
	case "small":
		orderByClause = "ORDER BY chapters_count ASC, title ASC"
	case "alphabet":
		orderByClause = "ORDER BY regexp_replace(lower(title), '[^а-яё]', '', 'g') ASC"
	case "created":
		orderByClause = "ORDER BY created_at DESC"
	case "oldest":
		orderByClause = "ORDER BY year_start ASC, title ASC"
	case "popular":
		fallthrough
	default:
		orderByClause = "ORDER BY views_count DESC, title ASC"
	}

	selectQuery := fmt.Sprintf(
		"SELECT id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at FROM novels WHERE age_rating IS DISTINCT FROM '19+' %s LIMIT $1 OFFSET $2",
		orderByClause,
	)

	rows, err := database.DB.Query(dbCtx, selectQuery, pageSize, offset)
	if err != nil {
		logger.Error("GetCatalogNovels: Failed to query novels: %v", err)
		return nil, err
	}
	defer rows.Close()

	novels := make([]models.Novel, 0)
	for rows.Next() {
		var n models.Novel
		if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt); err != nil {
			logger.Warn("GetCatalogNovels: Row scan error: %v", err)
			continue
		}
		novels = append(novels, n)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &models.NovelsPage{
		Novels:     novels,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}
