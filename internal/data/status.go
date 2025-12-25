package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"kappalib/internal/cache"
)

var betterStackToken = os.Getenv("BETTERSTACK_TOKEN")

func GetSystemStatus() (string, error) {
	key := "system_status"

	value, err := cache.C.GetOrFetch(key, 60*time.Second, func() (any, error) {
		if betterStackToken == "" {
			return "unknown", nil
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", "https://uptime.betterstack.com/api/v2/status-pages", nil)
		if err != nil {
			return "", err
		}

		req.Header.Set("Authorization", "Bearer "+betterStackToken)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("BetterStack API error: %d", resp.StatusCode)
		}

		var bsResponse struct {
			Data []struct {
				Attributes struct {
					AggregateState string `json:"aggregate_state"`
				} `json:"attributes"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&bsResponse); err != nil {
			return "", err
		}

		if len(bsResponse.Data) == 0 {
			return "unknown", nil
		}

		return bsResponse.Data[0].Attributes.AggregateState, nil
	})

	if err != nil {
		return "unknown", err
	}
	return value.(string), nil
}
