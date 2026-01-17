package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
	"github.com/ch1kulya/kappalib/internal/models"

	"github.com/ch1kulya/logger"
)

const (
	githubOwner = "ch1kulya"
	githubRepo  = "kappalib"
)

type githubPR struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	HTMLURL  string `json:"html_url"`
	MergedAt string `json:"merged_at"`
	User     struct {
		Login string `json:"login"`
	} `json:"user"`
}

func parseUpdateTag(body string) string {
	const tag = "!update:"
	idx := strings.Index(body, tag)
	if idx == -1 {
		return ""
	}
	start := idx + len(tag)
	end := strings.Index(body[start:], "\n")
	if end == -1 {
		return strings.TrimSpace(body[start:])
	}
	return strings.TrimSpace(body[start : start+end])
}

func GetAppUpdates(ctx context.Context, limit int) ([]models.AppUpdate, error) {
	key := fmt.Sprintf("github:prs:%d", limit)

	value, err := cache.C.GetOrFetch(key, 3*time.Minute, func() (any, error) {
		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		url := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/pulls?state=closed&sort=updated&direction=desc&per_page=%d",
			githubOwner, githubRepo, limit*2,
		)

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "kappalib-server")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error("GetAppUpdates: Failed to fetch PRs: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Error("GetAppUpdates: GitHub API returned status %d", resp.StatusCode)
			return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
		}

		var prs []githubPR
		if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
			logger.Error("GetAppUpdates: Failed to decode response: %v", err)
			return nil, err
		}

		updates := make([]models.AppUpdate, 0, limit)
		for _, pr := range prs {
			if pr.MergedAt == "" {
				continue
			}

			mergedAt, err := time.Parse(time.RFC3339, pr.MergedAt)
			if err != nil {
				continue
			}

			title := parseUpdateTag(pr.Body)
			if title == "" {
				continue
			}

			updates = append(updates, models.AppUpdate{
				PRNumber: pr.Number,
				Title:    title,
				Author:   pr.User.Login,
				URL:      pr.HTMLURL,
				MergedAt: mergedAt,
			})

			if len(updates) >= limit {
				break
			}
		}

		return updates, nil
	})

	if err != nil {
		return nil, err
	}
	return value.([]models.AppUpdate), nil
}
