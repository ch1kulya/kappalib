package data

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
	"github.com/jackc/pgx/v5"
)

//go:embed sql/user_profile_stats.sql
var queryUserProfileStats string

//go:embed sql/user_badges_get.sql
var queryUserBadgesGet string

//go:embed sql/user_activity_year.sql
var queryUserActivityYear string

//go:embed sql/user_streak_get.sql
var queryUserStreakGet string

const userProfileCacheTTL = 5 * time.Minute

type achievementTier struct {
	threshold   int
	description string
}

type achievementDef struct {
	id    string
	title string
	icon  string
	tiers []achievementTier
	value func(stats models.UserProfileStats, accountDays int) int
}

var achievementDefs = []achievementDef{
	{
		id:    "registered",
		title: "Начало пути",
		icon:  "cake",
		tiers: []achievementTier{
			{0, "Регистрация на kappalib"},
		},
		value: func(_ models.UserProfileStats, accountDays int) int { return accountDays },
	},
	{
		id:    "reader",
		title: "Книжный червь",
		icon:  "book",
		tiers: []achievementTier{
			{10 * 3600, "Больше 10 часов за чтением"},
			{50 * 3600, "Больше 50 часов за чтением"},
			{150 * 3600, "Больше 150 часов за чтением"},
			{500 * 3600, "Больше 500 часов за чтением"},
		},
		value: func(stats models.UserProfileStats, _ int) int { return stats.TotalSeconds },
	},
	{
		id:    "streak",
		title: "Постоянство",
		icon:  "flame",
		tiers: []achievementTier{
			{7, "7 дней активности подряд"},
			{30, "30 дней активности подряд"},
			{100, "100 дней активности подряд"},
			{365, "Целый год активности без перерыва"},
		},
		value: func(stats models.UserProfileStats, _ int) int { return stats.LongestStreak },
	},
	{
		id:    "commentator",
		title: "Красноречие",
		icon:  "message",
		tiers: []achievementTier{
			{1, "Первый комментарий"},
			{10, "10 комментариев"},
			{50, "50 комментариев"},
			{200, "200 комментариев"},
		},
		value: func(stats models.UserProfileStats, _ int) int { return stats.Comments },
	},
	{
		id:    "rating",
		title: "Признание",
		icon:  "thumbs-up",
		tiers: []achievementTier{
			{10, "Рейтинг комментариев 10+"},
			{50, "Рейтинг комментариев 50+"},
			{200, "Рейтинг комментариев 200+"},
			{1000, "Рейтинг комментариев 1000+"},
		},
		value: func(stats models.UserProfileStats, _ int) int { return stats.Rating },
	},
	{
		id:    "veteran",
		title: "Верность",
		icon:  "hourglass",
		tiers: []achievementTier{
			{30, "На kappalib больше месяца"},
			{180, "На kappalib больше полугода"},
			{365, "На kappalib больше года"},
			{730, "На kappalib больше двух лет"},
		},
		value: func(_ models.UserProfileStats, accountDays int) int { return accountDays },
	},
}

func BuildAchievements(stats models.UserProfileStats, createdAt, now time.Time) []models.Achievement {
	accountDays := 0
	if !createdAt.IsZero() && now.After(createdAt) {
		accountDays = int(now.Sub(createdAt).Hours() / 24)
	}

	achievements := make([]models.Achievement, 0, len(achievementDefs))
	for _, def := range achievementDefs {
		value := def.value(stats, accountDays)
		tier := 0
		for i, t := range def.tiers {
			if value >= t.threshold {
				tier = i + 1
			}
		}
		if tier == 0 {
			continue
		}
		achievements = append(achievements, models.Achievement{
			ID:          def.id,
			Title:       def.title,
			Description: def.tiers[tier-1].description,
			Icon:        def.icon,
			Tier:        tier,
			MaxTier:     len(def.tiers),
		})
	}
	return achievements
}

func userProfileCacheKey(userID string) string {
	return "user_profile:" + userID
}

func InvalidateUserProfile(userID string) {
	cache.C.Delete(userProfileCacheKey(userID))
}

func GetPublicProfile(ctx context.Context, userID string) (*models.ProfilePublic, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var profile models.ProfilePublic
	var avatarUpdatedAt time.Time
	err := database.DB.QueryRow(dbCtx, queryUsersGet, userID).Scan(
		&profile.ID, &profile.DisplayName, &profile.AvatarSeed,
		&profile.HasCustomAvatar, &avatarUpdatedAt, &profile.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		logger.Error("Failed to get public profile %s: %v", userID, err)
		return nil, err
	}

	profile.AvatarUpdatedAt = avatarUpdatedAt.Unix()
	return &profile, nil
}

func GetUserProfilePage(ctx context.Context, userID string) (*models.UserProfilePage, error) {
	value, err := cache.C.GetOrFetch(userProfileCacheKey(userID), userProfileCacheTTL, func() (any, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return fetchUserProfilePage(fetchCtx, userID)
	})
	if err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	badges, err := getUserBadges(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	streak, err := getUserStreak(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	page := *value.(*models.UserProfilePage)
	page.Badges = badges
	page.Streak = streak
	return &page, nil
}

func fetchUserProfilePage(ctx context.Context, userID string) (*models.UserProfilePage, error) {
	profile, err := GetPublicProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var stats models.UserProfileStats
	if err := database.DB.QueryRow(dbCtx, queryUserProfileStats, userID).Scan(
		&stats.TotalSeconds, &stats.LongestStreak, &stats.Comments, &stats.Rating,
	); err != nil {
		logger.Error("Failed to get profile stats for %s: %v", userID, err)
		return nil, err
	}

	activity, err := getUserActivityYear(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	return &models.UserProfilePage{
		Profile:      *profile,
		Stats:        stats,
		Achievements: BuildAchievements(stats, profile.CreatedAt, time.Now()),
		Activity:     activity,
	}, nil
}

func getUserBadges(ctx context.Context, userID string) ([]models.UserBadge, error) {
	rows, err := database.DB.Query(ctx, queryUserBadgesGet, userID)
	if err != nil {
		logger.Error("Failed to get badges for %s: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	badges := make([]models.UserBadge, 0)
	for rows.Next() {
		var b models.UserBadge
		if err := rows.Scan(&b.ID, &b.Title, &b.Description, &b.Icon, &b.AwardedAt); err != nil {
			logger.Warn("Badge row scan error: %v", err)
			continue
		}
		badges = append(badges, b)
	}
	if err := rows.Err(); err != nil {
		logger.Error("Failed to iterate badges for %s: %v", userID, err)
		return nil, err
	}
	return badges, nil
}

func getUserActivityYear(ctx context.Context, userID string) ([]models.ActivityDay, error) {
	rows, err := database.DB.Query(ctx, queryUserActivityYear, userID)
	if err != nil {
		logger.Error("Failed to get activity for %s: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	days := make([]models.ActivityDay, 0, 371)
	for rows.Next() {
		var d models.ActivityDay
		if err := rows.Scan(&d.Date, &d.Seconds); err != nil {
			logger.Warn("Activity row scan error: %v", err)
			continue
		}
		days = append(days, d)
	}
	if err := rows.Err(); err != nil {
		logger.Error("Failed to iterate activity for %s: %v", userID, err)
		return nil, err
	}
	return days, nil
}

func getUserStreak(ctx context.Context, userID string) (models.UserStreak, error) {
	var streak models.UserStreak
	if err := database.DB.QueryRow(ctx, queryUserStreakGet, userID).Scan(&streak.Current, &streak.ActiveToday); err != nil {
		logger.Error("Failed to get streak for %s: %v", userID, err)
		return models.UserStreak{}, err
	}
	return streak, nil
}
