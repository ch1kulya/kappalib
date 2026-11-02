package data

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
	"github.com/jackc/pgx/v5"
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

//go:embed sql/novels_catalog_filter.sql
var queryNovelsCatalogFilter string

//go:embed sql/novels_catalog_search.sql
var queryNovelsCatalogSearch string

//go:embed sql/catalog_stats.sql
var queryCatalogStats string

//go:embed sql/catalog_tags.sql
var queryCatalogTags string

const novelsCountCacheKey = "novels:count"

func isQueryableText(text string) bool {
	return utf8.ValidString(text) && !strings.ContainsRune(text, 0)
}

func GetNovel(ctx context.Context, id string) (*models.Novel, error) {
	if !isQueryableText(id) {
		return nil, ErrNovelNotFound
	}

	key := fmt.Sprintf("novel:%s", id)

	value, err := cache.C.GetOrFetch(key, 10*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		var n models.Novel
		var altTitlesJSON []byte
		err := database.DB.QueryRow(dbCtx, queryNovelsGetOne, id).Scan(
			&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount, &n.ViewsCount,
			&n.HasSelfHarm, &n.HasDrugUsage, &n.HasSexualViolence, &n.HasGraphicSex, &n.HasProfanity,
			&altTitlesJSON,
			&n.LastChapterAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNovelNotFound
			}
			return nil, err
		}

		if len(altTitlesJSON) > 0 {
			_ = json.Unmarshal(altTitlesJSON, &n.AltTitles)
		}
		if n.AltTitles == nil {
			n.AltTitles = []string{}
		}

		tags, err := getNovelTags(dbCtx, id)
		if err != nil {
			logger.Error("GetNovel: Failed to fetch tags for %q: %v", id, err)
			return nil, err
		}
		n.Tags = tags

		return &n, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.Novel), nil
}

func getNovelTags(ctx context.Context, novelID string) ([]models.Tag, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT t.id, t.name FROM tags t JOIN novel_tags nt ON t.id = nt.tag_id WHERE nt.novel_id = $1 ORDER BY t.name`,
		novelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]models.Tag, 0)
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func GetNovelsByIDs(ctx context.Context, ids []string) ([]models.NovelSummary, error) {
	if len(ids) == 0 {
		return []models.NovelSummary{}, nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, title, title_en, author, year_start, year_end,
			   status, description, age_rating, cover_url, created_at, chapters_count,
			   has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex, has_profanity
		FROM novels
		WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := database.DB.Query(dbCtx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	novels := make([]models.NovelSummary, 0, len(ids))
	for rows.Next() {
		var n models.NovelSummary
		if err := rows.Scan(
			&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount,
			&n.HasSelfHarm, &n.HasDrugUsage, &n.HasSexualViolence, &n.HasGraphicSex, &n.HasProfanity,
		); err != nil {
			return nil, err
		}
		novels = append(novels, n)
	}
	if err := rows.Err(); err != nil {
		logger.Error("GetNovelsByIDs: Failed to read novels: %v", err)
		return nil, err
	}

	return novels, nil
}

func normalizeNovelsSort(sort string) string {
	switch sort {
	case "newest", "large", "small", "alphabet", "created", "oldest", "popular":
		return sort
	default:
		return "popular"
	}
}

func getNovelsCount(ctx context.Context) (int, error) {
	value, err := cache.C.GetOrFetchWhen(novelsCountCacheKey, 5*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		var totalCount int
		if err := database.DB.QueryRow(dbCtx, queryNovelsCount).Scan(&totalCount); err != nil {
			logger.Error("GetNovels: Failed to count novels: %v", err)
			return nil, err
		}
		return totalCount, nil
	}, func(v any) bool {
		return v.(int) > 0
	})
	if err != nil {
		return 0, err
	}
	return value.(int), nil
}

func GetNovels(ctx context.Context, page int, sort string) (*models.NovelsPage, error) {
	sort = normalizeNovelsSort(sort)
	pageSize := 12
	key := fmt.Sprintf("novels:page:%d:sort:%s", page, sort)

	if cached, found := cache.C.Get(key); found {
		return cached.(*models.NovelsPage), nil
	}

	totalCount, err := getNovelsCount(ctx)
	if err != nil {
		return nil, err
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if page > totalPages {
		return &models.NovelsPage{
			Novels:     []models.NovelSummary{},
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	}

	offset := (page - 1) * pageSize

	value, err := cache.C.GetOrFetchWhen(key, 5*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		baseQuery := `SELECT id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at, chapters_count, has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex, has_profanity FROM novels`

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
			orderByClause = "ORDER BY views_count / SQRT(GREATEST(chapters_count, 1)) DESC, title ASC"
		}

		finalQuery := fmt.Sprintf("%s %s LIMIT $1 OFFSET $2", baseQuery, orderByClause)

		rows, err := database.DB.Query(dbCtx, finalQuery, pageSize, offset)
		if err != nil {
			logger.Error("GetNovels: Failed to query novels with sort '%s': %v", sort, err)
			return nil, err
		}
		defer rows.Close()

		novels := make([]models.NovelSummary, 0)
		for rows.Next() {
			var n models.NovelSummary
			if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
				&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
				&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount,
				&n.HasSelfHarm, &n.HasDrugUsage, &n.HasSexualViolence, &n.HasGraphicSex, &n.HasProfanity); err != nil {
				logger.Error("GetNovels: Row scan error: %v", err)
				return nil, err
			}
			novels = append(novels, n)
		}
		if err := rows.Err(); err != nil {
			logger.Error("GetNovels: Failed to read novels with sort %q: %v", sort, err)
			return nil, err
		}

		return &models.NovelsPage{
			Novels:     novels,
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	}, func(v any) bool {
		return len(v.(*models.NovelsPage).Novels) > 0
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.NovelsPage), nil
}

func SearchNovels(ctx context.Context, query string) ([]models.NovelSummary, error) {
	query = strings.TrimSpace(query)
	if query == "" || !isQueryableText(query) {
		return []models.NovelSummary{}, nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := database.DB.Query(dbCtx, queryNovelsSearch, query)
	if err != nil {
		logger.Error("SearchNovels: Query failed for %q: %v", query, err)
		return nil, err
	}
	defer rows.Close()

	novels := make([]models.NovelSummary, 0)
	for rows.Next() {
		var n models.NovelSummary
		var relevance float64
		if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount,
			&n.HasSelfHarm, &n.HasDrugUsage, &n.HasSexualViolence, &n.HasGraphicSex, &n.HasProfanity,
			&relevance); err != nil {
			return nil, err
		}
		novels = append(novels, n)
	}
	if err := rows.Err(); err != nil {
		logger.Error("SearchNovels: Failed to read results for %q: %v", query, err)
		return nil, err
	}

	return novels, nil
}

func GetSitemapData(ctx context.Context) ([]models.SitemapItem, error) {
	key := "sitemap_data"

	value, err := cache.C.GetOrFetchWhen(key, 1*time.Hour, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
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
				logger.Error("Sitemap: Row scan error: %v", err)
				return nil, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			logger.Error("Sitemap: Failed to read data: %v", err)
			return nil, err
		}

		return items, nil
	}, func(v any) bool {
		return len(v.([]models.SitemapItem)) > 0
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

func GetCatalogStats(ctx context.Context) (*models.CatalogStats, error) {
	value, err := cache.C.GetOrFetchWhen("catalog:stats", 10*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		var s models.CatalogStats
		if err := database.DB.QueryRow(dbCtx, queryCatalogStats).Scan(
			&s.NovelsCount, &s.ChaptersCount, &s.SourcesCount, &s.CharactersCount,
		); err != nil {
			logger.Error("GetCatalogStats: Failed to fetch stats: %v", err)
			return nil, err
		}
		return &s, nil
	}, func(v any) bool {
		return v.(*models.CatalogStats).NovelsCount > 0
	})
	if err != nil {
		return nil, err
	}
	return value.(*models.CatalogStats), nil
}

func GetCatalogTags(ctx context.Context) ([]models.Tag, error) {
	value, err := cache.C.GetOrFetchWhen("catalog:tags", 10*time.Minute, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		rows, err := database.DB.Query(dbCtx, queryCatalogTags)
		if err != nil {
			logger.Error("GetCatalogTags: Failed to fetch tags: %v", err)
			return nil, err
		}
		defer rows.Close()

		tags := make([]models.Tag, 0)
		for rows.Next() {
			var t models.Tag
			if err := rows.Scan(&t.ID, &t.Name); err != nil {
				logger.Error("GetCatalogTags: Row scan error: %v", err)
				return nil, err
			}
			tags = append(tags, t)
		}
		return tags, rows.Err()
	}, func(v any) bool {
		return len(v.([]models.Tag)) > 0
	})
	if err != nil {
		return nil, err
	}
	return value.([]models.Tag), nil
}

func catalogFilterArgs(filter models.CatalogFilter) []any {
	return []any{filter.Statuses, filter.YearFrom, filter.YearTo, filter.ChaptersFrom, filter.ChaptersTo, filter.TagIDs}
}

func GetCatalogNovels(ctx context.Context, page int, sort string, search string, filter models.CatalogFilter) (*models.CatalogPage, error) {
	pageSize := 24

	if !isQueryableText(search) {
		return &models.CatalogPage{
			Novels:   []models.NovelSummary{},
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	filterArgs := catalogFilterArgs(filter)

	if search != "" {
		searchArgs := append(filterArgs, search)

		var totalCount int
		countQuery := fmt.Sprintf("SELECT count(*) FROM (%s) AS c WHERE c.id IN (%s)", queryNovelsCatalogSearch, queryNovelsCatalogFilter)
		if err := database.DB.QueryRow(dbCtx, countQuery, searchArgs...).Scan(&totalCount); err != nil {
			logger.Error("GetCatalogNovels: Failed to count search results: %v", err)
			return nil, err
		}

		totalPages := (totalCount + pageSize - 1) / pageSize
		if page > totalPages {
			return &models.CatalogPage{
				Novels:     []models.NovelSummary{},
				Page:       page,
				PageSize:   pageSize,
				TotalCount: totalCount,
				TotalPages: totalPages,
			}, nil
		}
		offset := (page - 1) * pageSize

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
			orderByClause = "ORDER BY views_count / SQRT(GREATEST(chapters_count, 1)) DESC, title ASC"
		default:
			orderByClause = "ORDER BY relevance DESC, created_at DESC"
		}

		searchQuery := fmt.Sprintf(
			"SELECT id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at, chapters_count, has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex, has_profanity FROM (%s) AS c WHERE c.id IN (%s) %s LIMIT $8 OFFSET $9",
			queryNovelsCatalogSearch, queryNovelsCatalogFilter, orderByClause,
		)

		rows, err := database.DB.Query(dbCtx, searchQuery, append(searchArgs, pageSize, offset)...)
		if err != nil {
			logger.Error("GetCatalogNovels: Failed to search novels: %v", err)
			return nil, err
		}
		defer rows.Close()

		novels := make([]models.NovelSummary, 0)
		for rows.Next() {
			var n models.NovelSummary
			if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
				&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
				&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount,
				&n.HasSelfHarm, &n.HasDrugUsage, &n.HasSexualViolence, &n.HasGraphicSex, &n.HasProfanity); err != nil {
				logger.Error("GetCatalogNovels: Row scan error: %v", err)
				return nil, err
			}
			novels = append(novels, n)
		}
		if err := rows.Err(); err != nil {
			logger.Error("GetCatalogNovels: Failed to read search results: %v", err)
			return nil, err
		}

		return &models.CatalogPage{
			Novels:     novels,
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	}

	var totalCount int
	countQuery := fmt.Sprintf("SELECT count(*) FROM (%s) AS filtered", queryNovelsCatalogFilter)
	if err := database.DB.QueryRow(dbCtx, countQuery, filterArgs...).Scan(&totalCount); err != nil {
		logger.Error("GetCatalogNovels: Failed to count novels: %v", err)
		return nil, err
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	if page > totalPages {
		return &models.CatalogPage{
			Novels:     []models.NovelSummary{},
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}, nil
	}
	offset := (page - 1) * pageSize

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
		orderByClause = "ORDER BY views_count / SQRT(GREATEST(chapters_count, 1)) DESC, title ASC"
	}

	selectQuery := fmt.Sprintf(
		"SELECT id, title, title_en, author, year_start, year_end, status, description, age_rating, cover_url, created_at, chapters_count, has_self_harm, has_drug_usage, has_sexual_violence, has_graphic_sex, has_profanity FROM novels WHERE id IN (%s) %s LIMIT $7 OFFSET $8",
		queryNovelsCatalogFilter, orderByClause,
	)

	rows, err := database.DB.Query(dbCtx, selectQuery, append(filterArgs, pageSize, offset)...)
	if err != nil {
		logger.Error("GetCatalogNovels: Failed to query novels: %v", err)
		return nil, err
	}
	defer rows.Close()

	novels := make([]models.NovelSummary, 0)
	for rows.Next() {
		var n models.NovelSummary
		if err := rows.Scan(&n.ID, &n.Title, &n.TitleEn, &n.Author,
			&n.YearStart, &n.YearEnd, &n.Status, &n.Description,
			&n.AgeRating, &n.CoverURL, &n.CreatedAt, &n.ChapterCount,
			&n.HasSelfHarm, &n.HasDrugUsage, &n.HasSexualViolence, &n.HasGraphicSex, &n.HasProfanity); err != nil {
			logger.Error("GetCatalogNovels: Row scan error: %v", err)
			return nil, err
		}
		novels = append(novels, n)
	}
	if err := rows.Err(); err != nil {
		logger.Error("GetCatalogNovels: Failed to read novels: %v", err)
		return nil, err
	}

	return &models.CatalogPage{
		Novels:     novels,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}
