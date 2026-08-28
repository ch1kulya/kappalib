package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	if limit < 0 {
		limit = 0
	}
	key := fmt.Sprintf("github:prs:%d", limit)
	ttl := 3 * time.Minute
	if limit == 0 {
		ttl = 10 * time.Minute
	}

	value, err := cache.C.GetOrFetch(key, ttl, func() (any, error) {
		reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var updates []models.AppUpdate
		page := 1
		perPage := 100
		if limit > 0 && limit*2 < 100 {
			perPage = limit * 2
		}

		for {
			url := fmt.Sprintf(
				"https://api.github.com/repos/%s/%s/pulls?state=closed&sort=updated&direction=desc&per_page=%d&page=%d",
				githubOwner, githubRepo, perPage, page,
			)

			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
			if err != nil {
				return nil, fmt.Errorf("create request: %w", err)
			}
			req.Header.Set("Accept", "application/vnd.github.v3+json")
			req.Header.Set("User-Agent", "kappalib-server")
			if token := os.Getenv("GITHUB_TOKEN"); token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.Error("GetAppUpdates: Failed to fetch PRs: %v", err)
				if len(updates) > 0 {
					return updates, nil
				}
				return nil, err
			}

			if resp.StatusCode != http.StatusOK {
				_ = resp.Body.Close()
				logger.Error("GetAppUpdates: GitHub API returned status %d", resp.StatusCode)
				if len(updates) > 0 {
					return updates, nil
				}
				return nil, fmt.Errorf("github api status: %d", resp.StatusCode)
			}

			var prs []githubPR
			if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
				_ = resp.Body.Close()
				logger.Error("GetAppUpdates: Failed to decode response: %v", err)
				if len(updates) > 0 {
					return updates, nil
				}
				return nil, err
			}
			_ = resp.Body.Close()

			if len(prs) == 0 {
				break
			}

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
					Body:     pr.Body,
					Author:   pr.User.Login,
					URL:      pr.HTMLURL,
					MergedAt: mergedAt,
				})

				if limit > 0 && len(updates) >= limit {
					break
				}
			}

			if limit > 0 && len(updates) >= limit {
				break
			}

			if len(prs) < perPage {
				break
			}

			page++
		}

		return updates, nil
	})
	if err != nil {
		return nil, err
	}
	return value.([]models.AppUpdate), nil
}
