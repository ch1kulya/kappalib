package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestHandleTelegramWebhook(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		chatID        string
		inputSecret   string
		body          map[string]any
		wantErr       bool
		wantErrStatus int
	}{
		{
			name:          "empty secret returns 500",
			secret:        "",
			inputSecret:   "any",
			wantErr:       true,
			wantErrStatus: 500,
		},
		{
			name:          "invalid secret returns 403",
			secret:        "correct-secret",
			inputSecret:   "wrong-secret",
			wantErr:       true,
			wantErrStatus: 403,
		},
		{
			name:        "valid secret with empty body succeeds",
			secret:      "correct-secret",
			inputSecret: "correct-secret",
			body:        map[string]any{},
			wantErr:     false,
		},
		{
			name:        "valid secret with no callback_query succeeds",
			secret:      "correct-secret",
			inputSecret: "correct-secret",
			body:        map[string]any{"update_id": 123},
			wantErr:     false,
		},
		{
			name:        "wrong chat_id is ignored",
			secret:      "correct-secret",
			chatID:      "12345",
			inputSecret: "correct-secret",
			body: map[string]any{
				"update_id": 123,
				"callback_query": map[string]any{
					"id":   "callback-1",
					"data": "approve:comment-1",
					"message": map[string]any{
						"message_id": 1,
						"chat":       map[string]any{"id": 99999},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid callback data format is ignored",
			secret:      "correct-secret",
			chatID:      "12345",
			inputSecret: "correct-secret",
			body: map[string]any{
				"update_id": 123,
				"callback_query": map[string]any{
					"id":   "callback-1",
					"data": "invalid-format",
					"message": map[string]any{
						"message_id": 1,
						"chat":       map[string]any{"id": 12345},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "unknown action is ignored",
			secret:      "correct-secret",
			chatID:      "12345",
			inputSecret: "correct-secret",
			body: map[string]any{
				"update_id": 123,
				"callback_query": map[string]any{
					"id":   "callback-1",
					"data": "unknown:comment-1",
					"message": map[string]any{
						"message_id": 1,
						"chat":       map[string]any{"id": 12345},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data.SetTelegramWebhookSecret(tt.secret)
			data.SetTelegramChatID(tt.chatID)

			var body []byte
			if tt.body != nil {
				body, _ = json.Marshal(tt.body)
			}

			input := &TelegramWebhookInput{
				WebhookSecret: tt.inputSecret,
				Body:          body,
			}

			_, err := HandleTelegramWebhook(context.Background(), input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				errMsg := err.Error()
				if tt.wantErrStatus == 500 && errMsg != "Webhook not configured" {
					t.Errorf("expected 500 error, got: %v", err)
				}
				if tt.wantErrStatus == 403 && errMsg != "Invalid webhook secret" {
					t.Errorf("expected 403 error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetCommentsInputValidation(t *testing.T) {
	_, humaApi := humatest.New(t)

	type testResponse struct {
		Body string
	}

	huma.Register(humaApi, huma.Operation{
		OperationID: "get-comments",
		Method:      http.MethodGet,
		Path:        "/chapters/{chapterId}/comments",
	}, func(ctx context.Context, input *GetCommentsInput) (*testResponse, error) {
		return &testResponse{Body: "ok"}, nil
	})

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "valid chapter and no comment_id",
			path:       "/chapters/chp_12345678/comments",
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid chapter with comment_id cmt",
			path:       "/chapters/chp_12345678/comments?comment_id=cmt_abcdef12",
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid chapter with comment_id can",
			path:       "/chapters/chp_12345678/comments?comment_id=can_abcdef12",
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid chapter with page",
			path:       "/chapters/chp_12345678/comments?page=5",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid chapter format",
			path:       "/chapters/invalid_chap/comments",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid comment_id prefix",
			path:       "/chapters/chp_12345678/comments?comment_id=usr_abcdef12",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid comment_id length too short",
			path:       "/chapters/chp_12345678/comments?comment_id=cmt_123",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid comment_id length too long",
			path:       "/chapters/chp_12345678/comments?comment_id=cmt_123456789",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid comment_id uppercase",
			path:       "/chapters/chp_12345678/comments?comment_id=cmt_ABCDEF12",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid comment_id special characters",
			path:       "/chapters/chp_12345678/comments?comment_id=cmt_abcd-f12",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid page zero",
			path:       "/chapters/chp_12345678/comments?page=0",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid page negative",
			path:       "/chapters/chp_12345678/comments?page=-1",
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid page exceeds maximum",
			path:       "/chapters/chp_12345678/comments?page=10000",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := humaApi.Get(tt.path)
			if resp.Code != tt.wantStatus {
				t.Errorf("GET %s returned status %d, want %d: %s", tt.path, resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestEditCommentInputValidation(t *testing.T) {
	_, humaApi := humatest.New(t)

	type testResponse struct {
		Body string
	}

	huma.Register(humaApi, huma.Operation{
		OperationID: "edit-comment",
		Method:      http.MethodPatch,
		Path:        "/comments/{commentId}",
	}, func(ctx context.Context, input *EditCommentInput) (*testResponse, error) {
		return &testResponse{Body: "ok"}, nil
	})

	tests := []struct {
		name       string
		path       string
		body       map[string]any
		wantStatus int
	}{
		{
			name:       "valid comment id and body",
			path:       "/comments/cmt_12345678",
			body:       map[string]any{"content": "edited content"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid comment id format",
			path:       "/comments/invalid_cmt",
			body:       map[string]any{"content": "edited content"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid comment id can prefix",
			path:       "/comments/can_12345678",
			body:       map[string]any{"content": "edited content"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "empty content",
			path:       "/comments/cmt_12345678",
			body:       map[string]any{"content": ""},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := humaApi.Patch(tt.path, tt.body)
			if resp.Code != tt.wantStatus {
				t.Errorf("PATCH %s returned status %d, want %d: %s", tt.path, resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestEditCommentAnswerInputValidation(t *testing.T) {
	_, humaApi := humatest.New(t)

	type testResponse struct {
		Body string
	}

	huma.Register(humaApi, huma.Operation{
		OperationID: "edit-comment-answer",
		Method:      http.MethodPatch,
		Path:        "/comment-answers/{answerId}",
	}, func(ctx context.Context, input *EditCommentAnswerInput) (*testResponse, error) {
		return &testResponse{Body: "ok"}, nil
	})

	tests := []struct {
		name       string
		path       string
		body       map[string]any
		wantStatus int
	}{
		{
			name:       "valid answer id and body",
			path:       "/comment-answers/can_12345678",
			body:       map[string]any{"content": "edited answer content"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid answer id format",
			path:       "/comment-answers/invalid_can",
			body:       map[string]any{"content": "edited answer content"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid answer id cmt prefix",
			path:       "/comment-answers/cmt_12345678",
			body:       map[string]any{"content": "edited answer content"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "empty content",
			path:       "/comment-answers/can_12345678",
			body:       map[string]any{"content": ""},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := humaApi.Patch(tt.path, tt.body)
			if resp.Code != tt.wantStatus {
				t.Errorf("PATCH %s returned status %d, want %d: %s", tt.path, resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestRecordTimeInputValidation(t *testing.T) {
	_, humaApi := humatest.New(t)

	huma.Register(humaApi, huma.Operation{
		OperationID: "record-time",
		Method:      http.MethodPost,
		Path:        "/stats/time",
	}, func(ctx context.Context, input *RecordTimeInput) (*EmptyResponse, error) {
		return &EmptyResponse{Status: http.StatusNoContent}, nil
	})

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{
			name:       "valid seconds",
			body:       map[string]any{"seconds": 60},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "zero seconds",
			body:       map[string]any{"seconds": 0},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "negative seconds",
			body:       map[string]any{"seconds": -10},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "seconds exceeds maximum",
			body:       map[string]any{"seconds": 500},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := humaApi.Post("/stats/time", tt.body)
			if resp.Code != tt.wantStatus {
				t.Errorf("POST /stats/time returned status %d, want %d: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestHandleRecordTime(t *testing.T) {
	_, humaApi := humatest.New(t)

	huma.Register(humaApi, huma.Operation{
		OperationID: "record-time",
		Method:      http.MethodPost,
		Path:        "/stats/time",
	}, HandleRecordTime)

	resp := humaApi.Post("/stats/time", map[string]any{"seconds": 60})
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated request, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestCacheMiddleware(t *testing.T) {
	handler := CacheMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name          string
		method        string
		path          string
		headers       map[string]string
		cookies       []*http.Cookie
		wantCacheCtrl string
		wantVary      string
		wantPragma    string
		wantExpires   string
	}{
		{
			name:          "public GET request without auth",
			method:        http.MethodGet,
			path:          "/api/chapters/123/comments",
			wantCacheCtrl: "public, max-age=6000",
			wantVary:      "Cookie",
		},
		{
			name:          "authenticated GET request with session cookie",
			method:        http.MethodGet,
			path:          "/api/chapters/123/comments",
			cookies:       []*http.Cookie{{Name: SessionCookieName, Value: "valid_session"}},
			wantCacheCtrl: "private, max-age=60",
			wantVary:      "Cookie",
		},
		{
			name:          "authenticated GET request with authorization header",
			method:        http.MethodGet,
			path:          "/api/chapters/123/comments",
			headers:       map[string]string{"Authorization": "Bearer token"},
			wantCacheCtrl: "private, max-age=60",
			wantVary:      "Cookie",
		},
		{
			name:          "no-cache prefix profile path",
			method:        http.MethodGet,
			path:          "/api/profile/me",
			wantCacheCtrl: "no-store, no-cache, must-revalidate, private",
			wantPragma:    "no-cache",
			wantExpires:   "0",
		},
		{
			name:          "no-cache prefix webhook path",
			method:        http.MethodPost,
			path:          "/api/webhook/telegram",
			wantCacheCtrl: "no-store, no-cache, must-revalidate, private",
			wantPragma:    "no-cache",
			wantExpires:   "0",
		},
		{
			name:          "POST request without noCache prefix has no public cache header",
			method:        http.MethodPost,
			path:          "/api/chapters/123/comments",
			wantCacheCtrl: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			for _, c := range tt.cookies {
				req.AddCookie(c)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Cache-Control"); got != tt.wantCacheCtrl {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheCtrl)
			}
			if got := rec.Header().Get("Vary"); got != tt.wantVary {
				t.Errorf("Vary = %q, want %q", got, tt.wantVary)
			}
			if tt.wantPragma != "" {
				if got := rec.Header().Get("Pragma"); got != tt.wantPragma {
					t.Errorf("Pragma = %q, want %q", got, tt.wantPragma)
				}
			}
			if tt.wantExpires != "" {
				if got := rec.Header().Get("Expires"); got != tt.wantExpires {
					t.Errorf("Expires = %q, want %q", got, tt.wantExpires)
				}
			}
		})
	}
}
