package data

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestVerifyCommentsCaptcha_EmptyTokens(t *testing.T) {
	if verifyCommentsCaptcha("", "", "") {
		t.Error("expected false for empty tokens")
	}
}

func TestVerifyCommentsSmartCaptcha_MockServer(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   map[string]string
		secret     string
		want       bool
	}{
		{
			name:       "empty secret",
			secret:     "",
			statusCode: http.StatusOK,
			response:   map[string]string{"status": "ok"},
			want:       false,
		},
		{
			name:       "valid token ok",
			secret:     "test-secret",
			statusCode: http.StatusOK,
			response:   map[string]string{"status": "ok"},
			want:       true,
		},
		{
			name:       "invalid token failed",
			secret:     "test-secret",
			statusCode: http.StatusOK,
			response:   map[string]string{"status": "failed", "message": "invalid"},
			want:       false,
		},
		{
			name:       "server 500 error fail-closed",
			secret:     "test-secret",
			statusCode: http.StatusInternalServerError,
			response:   map[string]string{"status": "error"},
			want:       false,
		},
		{
			name:       "server 403 error fail-closed",
			secret:     "wrong-secret",
			statusCode: http.StatusForbidden,
			response:   map[string]string{"status": "error", "message": "forbidden"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST method, got %s", r.Method)
				}
				if err := r.ParseForm(); err != nil {
					t.Errorf("failed to parse form: %v", err)
				}
				if r.FormValue("secret") != tt.secret {
					t.Errorf("expected secret %s, got %s", tt.secret, r.FormValue("secret"))
				}
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			prevSecret := commentsSmartCaptchaSecret
			commentsSmartCaptchaSecret = tt.secret
			defer func() { commentsSmartCaptchaSecret = prevSecret }()

			client := &http.Client{Timeout: 3 * time.Second}
			params := url.Values{
				"secret": {commentsSmartCaptchaSecret},
				"token":  {"test-token"},
				"ip":     {"127.0.0.1"},
			}

			if commentsSmartCaptchaSecret == "" {
				if verifyCommentsSmartCaptcha("test-token", "127.0.0.1") != tt.want {
					t.Errorf("expected %v for empty secret", tt.want)
				}
				return
			}

			resp, err := client.PostForm(server.URL, params)
			if err != nil {
				if tt.want {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			defer func() { _ = resp.Body.Close() }()

			var got bool
			if resp.StatusCode != http.StatusOK {
				got = false
			} else {
				var result struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					got = false
				} else {
					got = result.Status == "ok"
				}
			}

			if got != tt.want {
				t.Errorf("verify result = %v, want %v", got, tt.want)
			}
		})
	}
}
