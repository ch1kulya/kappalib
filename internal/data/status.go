package data

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ch1kulya/kappalib/internal/cache"
)

var (
	phareToken = os.Getenv("PHARE_TOKEN")
	phareID    = os.Getenv("PHARE_ID")
)

type SystemStatus struct {
	Impact       string  `json:"impact"`
	Availability float64 `json:"availability"`
	UpdatedAt    string  `json:"updated_at"`
}

func GetSystemStatus() (SystemStatus, error) {
	key := "system_status"

	value, err := cache.C.GetOrFetch(key, 60*time.Second, func() (any, error) {
		if phareToken == "" || phareID == "" {
			return SystemStatus{Impact: "unknown"}, nil
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.phare.io/uptime/status-pages/%s/current-status", phareID), nil)
		if err != nil {
			return SystemStatus{Impact: "unknown"}, err
		}

		req.Header.Set("Authorization", "Bearer "+phareToken)

		resp, err := client.Do(req)
		if err != nil {
			return SystemStatus{Impact: "unknown"}, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusUnauthorized {
			return SystemStatus{Impact: "unknown"}, fmt.Errorf("Phare API error: unauthorized")
		}
		if resp.StatusCode == http.StatusForbidden {
			return SystemStatus{Impact: "unknown"}, fmt.Errorf("Phare API error: forbidden")
		}
		if resp.StatusCode != http.StatusOK {
			return SystemStatus{Impact: "unknown"}, fmt.Errorf("Phare API error: %d", resp.StatusCode)
		}

		var phareResponse struct {
			CurrentIncidentImpact string  `json:"current_incident_impact"`
			Availability          float64 `json:"availability"`
			UpdatedAt             string  `json:"updated_at"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&phareResponse); err != nil {
			return SystemStatus{Impact: "unknown"}, err
		}

		return SystemStatus{
			Impact:       phareResponse.CurrentIncidentImpact,
			Availability: phareResponse.Availability,
			UpdatedAt:    phareResponse.UpdatedAt,
		}, nil
	})

	if err != nil {
		return SystemStatus{Impact: "unknown"}, err
	}
	return value.(SystemStatus), nil
}
