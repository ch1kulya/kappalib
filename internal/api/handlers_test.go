package api

import (
	"context"
	"encoding/json"
	"net/http"
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
			name:       "valid chapter with comment_id ans",
			path:       "/chapters/chp_12345678/comments?comment_id=ans_abcdef12",
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
