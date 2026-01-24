package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/ch1kulya/kappalib/internal/auth"
	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/logger"

	"github.com/danielgtaylor/huma/v2"
)

const SessionCookieName = "kpl_session"

func clearSessionCookie() http.Cookie {
	return http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

type GetNovelsInput struct {
	Page int    `query:"page" default:"1" minimum:"1" maximum:"9999"`
	Sort string `query:"sort" default:"oldest" enum:"newest,oldest,large,small,alphabet,created"`
}

type SearchNovelsInput struct {
	Query string `query:"q" required:"true" maxLength:"50"`
}

type IDInput struct {
	ID string `path:"id"`
}

type BatchNovelsInput struct {
	Body struct {
		IDs []string `json:"ids" maxItems:"50"`
	}
}

type SyncCookiesInput struct {
	Body struct {
		Cookies map[string]models.CookieValue `json:"cookies"`
	}
}

type ProfileIDInput struct {
	ProfileID string `path:"id"`
}

type APIStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

type GetCommentsInput struct {
	ChapterID string `path:"chapterId"`
	Page      int    `query:"page" default:"1" minimum:"1" maximum:"9999"`
}

type CreateCommentAPIInput struct {
	ChapterID string `path:"chapterId"`
	Body      struct {
		Content        string `json:"content" minLength:"1" maxLength:"1000"`
		TurnstileToken string `json:"turnstile_token" minLength:"1"`
	}
}

type TelegramWebhookInput struct {
	WebhookSecret string `header:"X-Telegram-Bot-Api-Secret-Token"`
	Body          json.RawMessage
}

type UpdateDisplayNameInput struct {
	ProfileID string `path:"id"`
	Body      struct {
		DisplayName string `json:"display_name" minLength:"1" maxLength:"15"`
	}
}

type UploadAvatarInput struct {
	ProfileID string `path:"id"`
	Body      struct {
		Image string `json:"image" minLength:"1"`
	}
}

type GetCurrentUserInput struct{}

type LogoutInput struct{}

func HandleStatus(ctx context.Context, input *struct{}) (*struct{ Body APIStatus }, error) {
	dbStatus := "connected"
	if err := database.DB.Ping(ctx); err != nil {
		dbStatus = "disconnected"
	}

	return &struct{ Body APIStatus }{
		Body: APIStatus{Status: "ok", Database: dbStatus},
	}, nil
}

func HandleGetSitemapData(ctx context.Context, input *struct{}) (*struct{ Body any }, error) {
	items, err := data.GetSitemapData(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch data")
	}
	return &struct{ Body any }{Body: items}, nil
}

func HandleGetNovels(ctx context.Context, input *GetNovelsInput) (*struct{ Body any }, error) {
	novels, err := data.GetNovels(ctx, input.Page, input.Sort)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error")
	}
	return &struct{ Body any }{Body: novels}, nil
}

func HandleSearchNovels(ctx context.Context, input *SearchNovelsInput) (*struct{ Body any }, error) {
	if input.Query == "" {
		return nil, huma.Error400BadRequest("Search query is required")
	}

	novels, err := data.SearchNovels(ctx, input.Query)
	if err != nil {
		return nil, huma.Error500InternalServerError("Search failed")
	}

	return &struct{ Body any }{
		Body: map[string]any{
			"novels": novels,
			"query":  input.Query,
		},
	}, nil
}

func HandleGetNovel(ctx context.Context, input *IDInput) (*struct{ Body any }, error) {
	novel, err := data.GetNovel(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Novel not found")
	}
	return &struct{ Body any }{Body: novel}, nil
}

func HandleGetNovelsBatch(ctx context.Context, input *BatchNovelsInput) (*struct{ Body any }, error) {
	if len(input.Body.IDs) == 0 {
		return &struct{ Body any }{Body: []models.Novel{}}, nil
	}
	novels, err := data.GetNovelsByIDs(ctx, input.Body.IDs)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch novels")
	}
	return &struct{ Body any }{Body: novels}, nil
}

func HandleGetChaptersList(ctx context.Context, input *IDInput) (*struct{ Body any }, error) {
	chapters, err := data.GetChapters(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch chapters")
	}
	return &struct{ Body any }{Body: chapters}, nil
}

func HandleGetChapter(ctx context.Context, input *IDInput) (*struct{ Body any }, error) {
	chapter, err := data.GetChapter(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Chapter not found")
	}
	return &struct{ Body any }{Body: chapter}, nil
}

func HandleGetProfile(ctx context.Context, input *ProfileIDInput) (*struct{ Body any }, error) {
	profile, err := data.GetProfile(ctx, input.ProfileID)
	if err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}
	return &struct{ Body any }{Body: profile}, nil
}

func HandleSyncCookies(ctx context.Context, input *SyncCookiesInput) (*struct{ Body any }, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	result, err := data.SyncCookies(ctx, userID, input.Body.Cookies)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to sync cookies")
	}

	return &struct{ Body any }{Body: result}, nil
}

func HandleDeleteProfile(ctx context.Context, input *ProfileIDInput) (*struct {
	SetCookies []http.Cookie `header:"Set-Cookie"`
}, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	if userID != input.ProfileID {
		return nil, huma.Error403Forbidden("Access denied")
	}

	err := data.DeleteProfile(ctx, userID)
	if err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}

	return &struct {
		SetCookies []http.Cookie `header:"Set-Cookie"`
	}{
		SetCookies: []http.Cookie{clearSessionCookie()},
	}, nil
}

func HandleLogout(ctx context.Context, input *LogoutInput) (*struct {
	Body       any
	SetCookies []http.Cookie `header:"Set-Cookie"`
}, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	sessionID := auth.GetSessionIDFromContext(ctx)
	if err := auth.DeleteSession(ctx, sessionID); err != nil {
		logger.Warn("Failed to delete session on logout: %v", err)
	}

	return &struct {
		Body       any
		SetCookies []http.Cookie `header:"Set-Cookie"`
	}{
		Body:       map[string]string{"status": "logged out"},
		SetCookies: []http.Cookie{clearSessionCookie()},
	}, nil
}

func HandleGetComments(ctx context.Context, input *GetCommentsInput) (*struct{ Body any }, error) {
	comments, err := data.GetApprovedComments(ctx, input.ChapterID, input.Page)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch comments")
	}
	return &struct{ Body any }{Body: comments}, nil
}

func HandleCreateComment(ctx context.Context, input *CreateCommentAPIInput) (*struct{ Body any }, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	commentInput := models.CreateCommentInput{
		ChapterID:      input.ChapterID,
		Content:        input.Body.Content,
		TurnstileToken: input.Body.TurnstileToken,
	}

	comment, err := data.CreateComment(ctx, userID, commentInput)
	if err != nil {
		switch err.Error() {
		case "rate limit exceeded":
			return nil, huma.Error429TooManyRequests("Подождите 30 секунд перед отправкой следующего комментария")
		case "captcha verification failed":
			return nil, huma.Error400BadRequest("Captcha verification failed")
		case "invalid content length":
			return nil, huma.Error400BadRequest("Comment must be 1-1000 characters")
		case "chapter not found":
			return nil, huma.Error404NotFound("Chapter not found")
		default:
			return nil, huma.Error500InternalServerError("Failed to create comment")
		}
	}
	return &struct{ Body any }{Body: comment}, nil
}

func HandleTelegramWebhook(ctx context.Context, input *TelegramWebhookInput) (*struct{}, error) {
	expectedSecret := data.GetTelegramWebhookSecret()
	if expectedSecret == "" {
		logger.Error("Telegram webhook secret not configured")
		return nil, huma.Error500InternalServerError("Webhook not configured")
	}
	if input.WebhookSecret != expectedSecret {
		logger.Warn("Telegram Webhook: Invalid secret token")
		return nil, huma.Error403Forbidden("Invalid webhook secret")
	}

	bodyStr := string(input.Body)
	logger.Debug("Telegram Webhook Payload: %s", bodyStr)

	type TelegramUpdate struct {
		UpdateID      int64 `json:"update_id"`
		CallbackQuery *struct {
			ID      string `json:"id"`
			Data    string `json:"data"`
			Message *struct {
				MessageID int64 `json:"message_id"`
				Chat      struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
		} `json:"callback_query"`
	}

	var update TelegramUpdate
	if err := json.Unmarshal(input.Body, &update); err != nil {
		logger.Error("Failed to unmarshal Telegram update: %v", err)
		return &struct{}{}, nil
	}

	if update.CallbackQuery == nil {
		return &struct{}{}, nil
	}

	callback := update.CallbackQuery

	if callback.Message == nil {
		logger.Warn("CallbackQuery received without Message field")
		return &struct{}{}, nil
	}

	expectedChatID := data.GetTelegramChatID()
	if expectedChatID != "" && fmt.Sprintf("%d", callback.Message.Chat.ID) != expectedChatID {
		logger.Warn("Telegram Webhook: Invalid chat ID")
		return &struct{}{}, nil
	}

	parts := strings.SplitN(callback.Data, ":", 2)
	if len(parts) != 2 {
		logger.Warn("Invalid callback data format: %s", callback.Data)
		return &struct{}{}, nil
	}

	action := parts[0]
	commentID := parts[1]

	var status string
	var statusText string

	switch action {
	case "approve":
		status = "approved"
		statusText = "✅ Подтверждено"
	case "reject":
		status = "rejected"
		statusText = "❌ Отклонено"
	default:
		logger.Warn("Unknown action in callback: %s", action)
		return &struct{}{}, nil
	}

	if err := data.UpdateCommentStatus(ctx, commentID, status); err != nil {
		logger.Error("Failed to update comment via webhook: %v", err)
		return &struct{}{}, nil
	}

	if err := data.DeleteTelegramMessage(
		callback.Message.Chat.ID,
		callback.Message.MessageID,
	); err != nil {
		logger.Warn("Failed to delete telegram message: %v", err)
	}

	answerURL := fmt.Sprintf(
		"https://api.telegram.org/bot%s/answerCallbackQuery",
		os.Getenv("TELEGRAM_BOT_TOKEN"),
	)

	go func() {
		_, err := http.PostForm(answerURL, url.Values{
			"callback_query_id": {callback.ID},
			"text":              {statusText},
		})
		if err != nil {
			logger.Error("Failed to answer callback query: %v", err)
		}
	}()

	return &struct{}{}, nil
}

func HandleUpdateDisplayName(ctx context.Context, input *UpdateDisplayNameInput) (*struct{ Body any }, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	if userID != input.ProfileID {
		return nil, huma.Error403Forbidden("Access denied")
	}

	profile, err := data.UpdateDisplayName(ctx, userID, input.Body.DisplayName)
	if err != nil {
		if strings.Contains(err.Error(), "invalid name") {
			return nil, huma.Error400BadRequest("Invalid display name")
		}
		return nil, huma.Error500InternalServerError("Failed to update profile")
	}
	return &struct{ Body any }{Body: profile}, nil
}

func HandleUploadAvatar(ctx context.Context, input *UploadAvatarInput) (*struct{ Body any }, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	if userID != input.ProfileID {
		return nil, huma.Error403Forbidden("Access denied")
	}

	imageData, err := base64.StdEncoding.DecodeString(input.Body.Image)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid base64 image")
	}

	if len(imageData) > 1<<20 {
		return nil, huma.Error400BadRequest("Image too large (max 1MB)")
	}

	profile, err := data.UpdateAvatar(ctx, userID, imageData)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported format") {
			return nil, huma.Error400BadRequest("Unsupported format")
		}
		return nil, huma.Error500InternalServerError("Upload failed")
	}
	return &struct{ Body any }{Body: profile}, nil
}

func HandleGetCurrentUser(ctx context.Context, input *GetCurrentUserInput) (*struct{ Body any }, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	profile, err := data.GetProfile(ctx, userID)
	if err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}

	return &struct{ Body any }{Body: profile}, nil
}
