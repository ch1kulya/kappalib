package web

import (
	"context"
	"sort"
	"time"

	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/ch1kulya/kappalib/internal/models"
)

type matchedNovelAddition struct {
	addition *models.NovelAddition
	matched  bool
}

func buildFeed(ctx context.Context, isHome bool) ([]models.HomeUpdateItem, *models.AppUpdate, error) {
	chapterUpdates, err := data.GetLatestUpdates(ctx, 0)
	if err != nil {
		chapterUpdates = nil
	}

	appUpdates, err := data.GetAppUpdates(ctx, 0)
	if err != nil {
		appUpdates = nil
	}

	novelAdditions, err := data.GetRecentlyAddedNovels(ctx, 0)
	if err != nil {
		novelAdditions = nil
	}

	top12 := make(map[string]bool)
	popNovels, err := data.GetNovels(ctx, 1, "popular")
	if err == nil && popNovels != nil {
		for _, n := range popNovels.Novels {
			top12[n.ID] = true
		}
	}

	var pinnedAppUpdate *models.AppUpdate
	if len(appUpdates) > 0 {
		pinnedAppUpdate = &appUpdates[0]
	}

	var oldestDates []time.Time
	if len(chapterUpdates) > 0 {
		oldestDates = append(oldestDates, chapterUpdates[len(chapterUpdates)-1].UpdatedAt)
	}
	if len(novelAdditions) > 0 {
		oldestDates = append(oldestDates, novelAdditions[len(novelAdditions)-1].CreatedAt)
	}
	if len(appUpdates) > 0 {
		oldestDates = append(oldestDates, appUpdates[len(appUpdates)-1].MergedAt)
	}

	if len(oldestDates) > 0 {
		cutoff := oldestDates[0]
		for _, d := range oldestDates[1:] {
			if d.After(cutoff) {
				cutoff = d
			}
		}

		filteredChapters := make([]models.NovelUpdate, 0, len(chapterUpdates))
		for _, cu := range chapterUpdates {
			if !cu.UpdatedAt.Before(cutoff) {
				filteredChapters = append(filteredChapters, cu)
			}
		}
		chapterUpdates = filteredChapters

		filteredNovels := make([]models.NovelAddition, 0, len(novelAdditions))
		for _, na := range novelAdditions {
			if !na.CreatedAt.Before(cutoff) {
				filteredNovels = append(filteredNovels, na)
			}
		}
		novelAdditions = filteredNovels

		filteredApps := make([]models.AppUpdate, 0, len(appUpdates))
		for _, au := range appUpdates {
			if !au.MergedAt.Before(cutoff) {
				filteredApps = append(filteredApps, au)
			}
		}
		appUpdates = filteredApps
	}

	novelAddMap := make(map[string]*matchedNovelAddition)
	for i := range novelAdditions {
		novelAddMap[novelAdditions[i].ID] = &matchedNovelAddition{addition: &novelAdditions[i]}
	}

	var remainingChapters []models.NovelUpdate
	var feedItems []models.HomeUpdateItem

	for _, cu := range chapterUpdates {
		if m, exists := novelAddMap[cu.NovelID]; exists && !m.matched {
			diff := cu.UpdatedAt.Sub(m.addition.CreatedAt)
			if diff < 0 {
				diff = -diff
			}
			if diff <= time.Hour {
				m.matched = true
				updatedAt := m.addition.CreatedAt
				if cu.UpdatedAt.After(updatedAt) {
					updatedAt = cu.UpdatedAt
				}
				feedItems = append(feedItems, models.HomeUpdateItem{
					Type: "novel_with_chapters",
					NovelAdditionChapters: &models.NovelAdditionChapters{
						ID:           m.addition.ID,
						Title:        m.addition.Title,
						Author:       m.addition.Author,
						YearStart:    m.addition.YearStart,
						Status:       m.addition.Status,
						Description:  m.addition.Description,
						CoverURL:     m.addition.CoverURL,
						ChapterMin:   cu.ChapterMin,
						ChapterMax:   cu.ChapterMax,
						ChapterCount: cu.ChapterCount,
						UpdatedAt:    updatedAt,
					},
					UpdatedAt: updatedAt,
				})
				continue
			}
		}
		remainingChapters = append(remainingChapters, cu)
	}

	for _, m := range novelAddMap {
		if !m.matched {
			feedItems = append(feedItems, models.HomeUpdateItem{
				Type:          "novel",
				NovelAddition: m.addition,
				UpdatedAt:     m.addition.CreatedAt,
			})
		}
	}

	var popularChapters []models.NovelUpdate
	var unpopularChapters []models.NovelUpdate

	for _, cu := range remainingChapters {
		if top12[cu.NovelID] {
			popularChapters = append(popularChapters, cu)
		} else {
			unpopularChapters = append(unpopularChapters, cu)
		}
	}

	for _, pu := range popularChapters {
		cu := pu
		feedItems = append(feedItems, models.HomeUpdateItem{
			Type:          "chapter",
			ChapterUpdate: &cu,
			UpdatedAt:     cu.UpdatedAt,
		})
	}

	sort.Slice(unpopularChapters, func(i, j int) bool {
		return unpopularChapters[i].UpdatedAt.After(unpopularChapters[j].UpdatedAt)
	})

	var currentCluster []models.NovelUpdate
	for _, u := range unpopularChapters {
		if len(currentCluster) == 0 {
			currentCluster = append(currentCluster, u)
			continue
		}
		if currentCluster[0].UpdatedAt.Sub(u.UpdatedAt) <= time.Hour {
			currentCluster = append(currentCluster, u)
		} else {
			processUnpopularCluster(currentCluster, &feedItems)
			currentCluster = []models.NovelUpdate{u}
		}
	}
	if len(currentCluster) > 0 {
		processUnpopularCluster(currentCluster, &feedItems)
	}

	for i, au := range appUpdates {
		app := au
		item := models.HomeUpdateItem{
			Type:      "app",
			AppUpdate: &app,
			UpdatedAt: app.MergedAt,
		}
		if i == 0 {
			item.Pinned = true
		}
		feedItems = append(feedItems, item)
	}

	sort.Slice(feedItems, func(i, j int) bool {
		if feedItems[i].Pinned != feedItems[j].Pinned {
			return feedItems[i].Pinned
		}
		return feedItems[i].UpdatedAt.After(feedItems[j].UpdatedAt)
	})

	if isHome && len(feedItems) > 15 {
		feedItems = feedItems[:15]
	}

	return feedItems, pinnedAppUpdate, nil
}

func processUnpopularCluster(cluster []models.NovelUpdate, items *[]models.HomeUpdateItem) {
	if len(cluster) > 1 {
		totalChapters := 0
		uniqueNovels := make(map[string]bool)
		for _, u := range cluster {
			totalChapters += u.ChapterCount
			uniqueNovels[u.NovelID] = true
		}
		*items = append(*items, models.HomeUpdateItem{
			Type: "grouped",
			GroupedUpdates: &models.GroupedUpdates{
				NovelCount:   len(uniqueNovels),
				ChapterCount: totalChapters,
				UpdatedAt:    cluster[0].UpdatedAt,
			},
			UpdatedAt: cluster[0].UpdatedAt,
		})
	} else if len(cluster) == 1 {
		cu := cluster[0]
		*items = append(*items, models.HomeUpdateItem{
			Type:          "chapter",
			ChapterUpdate: &cu,
			UpdatedAt:     cu.UpdatedAt,
		})
	}
}
