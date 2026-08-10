package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/auth"
	"github.com/ch1kulya/kappalib/internal/data"
	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/logger"

	"github.com/danielgtaylor/huma/v2"
)

const (
	SessionCookieName   = "kpl_session"
	MaxAvatarSize       = 1 << 20
	MaxCommentImageSize = 5 << 20
)

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

func requireAuth(ctx context.Context) (string, error) {
	userID := auth.GetUserIDFromContext(ctx)
	if userID == "" {
		return "", huma.Error401Unauthorized("Authentication required")
	}
	return userID, nil
}

func requireOwner(userID, resourceID string) error {
	if userID != resourceID {
		return huma.Error403Forbidden("Access denied")
	}
	return nil
}

type GetNovelsInput struct {
	Page int    `query:"page" default:"1" minimum:"1" maximum:"9999"`
	Sort string `query:"sort" default:"popular" enum:"newest,oldest,large,small,alphabet,created,popular"`
}

type SearchNovelsInput struct {
	Query string `query:"q" required:"true" maxLength:"50"`
}

type NovelIDInput struct {
	ID string `path:"id" pattern:"^nvl_[a-z0-9]{8}$"`
}

type ChapterIDInput struct {
	ID string `path:"id" pattern:"^chp_[a-z0-9]{8}$"`
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
	ProfileID string `path:"id" pattern:"^usr_[a-z0-9]{8}$"`
}

type GetCommentsInput struct {
	ChapterID string `path:"chapterId" pattern:"^chp_[a-z0-9]{8}$"`
	Page      int    `query:"page" default:"1" minimum:"1" maximum:"9999"`
}

type CreateCommentInput struct {
	ChapterID string `path:"chapterId" pattern:"^chp_[a-z0-9]{8}$"`
	Body      struct {
		Content        string `json:"content" minLength:"1" maxLength:"12000"`
		TurnstileToken string `json:"turnstile_token" minLength:"1"`
	}
}

type TelegramWebhookInput struct {
	WebhookSecret string `header:"X-Telegram-Bot-Api-Secret-Token"`
	Body          json.RawMessage
}

type UpdateDisplayNameInput struct {
	ProfileID string `path:"id" pattern:"^usr_[a-z0-9]{8}$"`
	Body      struct {
		DisplayName string `json:"display_name" minLength:"1" maxLength:"15"`
	}
}

type UploadAvatarInput struct {
	ProfileID string `path:"id" pattern:"^usr_[a-z0-9]{8}$"`
	Body      struct {
		Image string `json:"image" minLength:"1"`
	}
}

type UploadCommentImageInput struct {
	Body struct {
		Image string `json:"image" minLength:"1"`
	}
}

type CommentImageResponse struct {
	Body struct {
		URL string `json:"url"`
	}
}

type VoteCommentInput struct {
	CommentID string `path:"commentId" pattern:"^cmt_[a-z0-9]{8}$"`
	Body      struct {
		Value int `json:"value" minimum:"-1" maximum:"1"`
	}
}

type VoteCommentResponse struct {
	Body struct {
		Score    int `json:"score"`
		UserVote int `json:"user_vote"`
	}
}

type APIStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

type StatusResponse struct {
	Body APIStatus
}

type NovelResponse struct {
	Body models.Novel
}

type NovelsPageResponse struct {
	Body models.NovelsPage
}

type ChapterResponse struct {
	Body models.Chapter
}

type ChaptersListResponse struct {
	Body models.ChaptersList
}

type ProfileResponse struct {
	Body models.ProfilePublic
}

type CommentResponse struct {
	Body models.Comment
}

type CommentsPageResponse struct {
	Body models.CommentsPage
}

type UserCommentsPageResponse struct {
	Body models.UserCommentsPage
}

type GetUserCommentsInput struct {
	Page int `query:"page" default:"1" minimum:"1" maximum:"9999"`
}

type CreateCommentAnswerInput struct {
	CommentID string `path:"commentId" pattern:"^cmt_[a-z0-9]{8}$"`
	Body      struct {
		Content        string `json:"content" minLength:"1" maxLength:"2000"`
		TurnstileToken string `json:"turnstile_token" minLength:"1"`
	}
}

type DeleteCommentAnswerInput struct {
	AnswerID string `path:"answerId" pattern:"^can_[a-z0-9]{8}$"`
}

type CommentAnswerResponse struct {
	Body models.CommentAnswer
}

type CommentStatsResponse struct {
	Body models.UserCommentStats
}

type SitemapResponse struct {
	Body []models.SitemapItem
}

type SearchNovelsResponse struct {
	Body struct {
		Novels []models.NovelSummary `json:"novels"`
		Query  string                `json:"query"`
	}
}

type BatchNovelsResponse struct {
	Body []models.NovelSummary
}

type CookieSyncResponse struct {
	Body map[string]models.CookieValue
}

type LogoutResponse struct {
	Status     int           `json:"-" default:"204"`
	SetCookies []http.Cookie `header:"Set-Cookie"`
}

type DeleteProfileResponse struct {
	Status     int           `json:"-" default:"204"`
	SetCookies []http.Cookie `header:"Set-Cookie"`
}

type EmptyResponse struct {
	Status int `json:"-" default:"204"`
}

func HandleStatus(ctx context.Context, input *struct{}) (*StatusResponse, error) {
	dbStatus := "connected"
	if err := database.DB.Ping(ctx); err != nil {
		dbStatus = "disconnected"
	}

	return &StatusResponse{
		Body: APIStatus{Status: "ok", Database: dbStatus},
	}, nil
}

func HandleGetSitemapData(ctx context.Context, input *struct{}) (*SitemapResponse, error) {
	items, err := data.GetSitemapData(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch data")
	}
	return &SitemapResponse{Body: items}, nil
}

func HandleGetNovels(ctx context.Context, input *GetNovelsInput) (*NovelsPageResponse, error) {
	novels, err := data.GetNovels(ctx, input.Page, input.Sort)
	if err != nil {
		return nil, huma.Error500InternalServerError("Database error")
	}
	return &NovelsPageResponse{Body: *novels}, nil
}

func HandleSearchNovels(ctx context.Context, input *SearchNovelsInput) (*SearchNovelsResponse, error) {
	novels, err := data.SearchNovels(ctx, input.Query)
	if err != nil {
		return nil, huma.Error500InternalServerError("Search failed")
	}

	return &SearchNovelsResponse{
		Body: struct {
			Novels []models.NovelSummary `json:"novels"`
			Query  string                `json:"query"`
		}{
			Novels: novels,
			Query:  input.Query,
		},
	}, nil
}

func HandleGetNovel(ctx context.Context, input *NovelIDInput) (*NovelResponse, error) {
	novel, err := data.GetNovel(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Novel not found")
	}
	return &NovelResponse{Body: *novel}, nil
}

func HandleGetNovelsBatch(ctx context.Context, input *BatchNovelsInput) (*BatchNovelsResponse, error) {
	if len(input.Body.IDs) == 0 {
		return &BatchNovelsResponse{Body: []models.NovelSummary{}}, nil
	}
	novels, err := data.GetNovelsByIDs(ctx, input.Body.IDs)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch novels")
	}
	return &BatchNovelsResponse{Body: novels}, nil
}

func HandleGetChaptersList(ctx context.Context, input *NovelIDInput) (*ChaptersListResponse, error) {
	chapters, err := data.GetChapters(ctx, input.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch chapters")
	}
	return &ChaptersListResponse{Body: *chapters}, nil
}

func HandleGetChapter(ctx context.Context, input *ChapterIDInput) (*ChapterResponse, error) {
	chapter, err := data.GetChapter(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Chapter not found")
	}
	return &ChapterResponse{Body: *chapter}, nil
}

func HandleGetProfile(ctx context.Context, input *ProfileIDInput) (*ProfileResponse, error) {
	profile, err := data.GetProfile(ctx, input.ProfileID)
	if err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}
	return &ProfileResponse{Body: *profile}, nil
}

func HandleSyncCookies(ctx context.Context, input *SyncCookiesInput) (*CookieSyncResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	result, err := data.SyncCookies(ctx, userID, input.Body.Cookies)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to sync cookies")
	}

	return &CookieSyncResponse{Body: result}, nil
}

func HandleDeleteProfile(ctx context.Context, input *ProfileIDInput) (*DeleteProfileResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if err := requireOwner(userID, input.ProfileID); err != nil {
		return nil, err
	}

	if err := data.DeleteProfile(ctx, userID); err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}

	return &DeleteProfileResponse{
		Status:     http.StatusNoContent,
		SetCookies: []http.Cookie{clearSessionCookie()},
	}, nil
}

func HandleLogout(ctx context.Context, input *struct{}) (*LogoutResponse, error) {
	if _, err := requireAuth(ctx); err != nil {
		return nil, err
	}

	sessionID := auth.GetSessionIDFromContext(ctx)
	if err := auth.DeleteSession(ctx, sessionID); err != nil {
		logger.Warn("Failed to delete session on logout: %v", err)
	}

	return &LogoutResponse{
		Status:     http.StatusNoContent,
		SetCookies: []http.Cookie{clearSessionCookie()},
	}, nil
}

func HandleGetComments(ctx context.Context, input *GetCommentsInput) (*CommentsPageResponse, error) {
	userID := auth.GetUserIDFromContext(ctx)

	comments, err := data.GetVisibleComments(ctx, input.ChapterID, userID, input.Page)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch comments")
	}

	return &CommentsPageResponse{Body: *comments}, nil
}

func HandleCreateComment(ctx context.Context, input *CreateCommentInput) (*CommentResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	commentInput := models.CreateCommentInput{
		ChapterID:      input.ChapterID,
		Content:        input.Body.Content,
		TurnstileToken: input.Body.TurnstileToken,
	}

	comment, err := data.CreateComment(ctx, userID, commentInput)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRateLimitExceeded):
			return nil, huma.Error429TooManyRequests("Подождите 30 секунд перед отправкой следующего комментария")
		case errors.Is(err, data.ErrCaptchaFailed):
			return nil, huma.Error400BadRequest("Captcha verification failed")
		case errors.Is(err, data.ErrInvalidContentLength):
			return nil, huma.Error400BadRequest("Comment must be 1-3000 characters")
		case errors.Is(err, data.ErrChapterNotFound):
			return nil, huma.Error404NotFound("Chapter not found")
		default:
			return nil, huma.Error500InternalServerError("Failed to create comment")
		}
	}
	return &CommentResponse{Body: *comment}, nil
}

func HandleTelegramWebhook(ctx context.Context, input *TelegramWebhookInput) (*EmptyResponse, error) {
	expectedSecret := data.GetTelegramWebhookSecret()
	if expectedSecret == "" {
		logger.Error("Telegram webhook secret not configured")
		return nil, huma.Error500InternalServerError("Webhook not configured")
	}
	if input.WebhookSecret != expectedSecret {
		logger.Warn("Telegram Webhook: Invalid secret token")
		return nil, huma.Error403Forbidden("Invalid webhook secret")
	}

	var update struct {
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

	if err := json.Unmarshal(input.Body, &update); err != nil {
		logger.Error("Failed to unmarshal Telegram update: %v", err)
		return &EmptyResponse{Status: http.StatusNoContent}, nil
	}

	if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
		return &EmptyResponse{Status: http.StatusNoContent}, nil
	}

	callback := update.CallbackQuery

	expectedChatID := data.GetTelegramChatID()
	if expectedChatID != "" && fmt.Sprintf("%d", callback.Message.Chat.ID) != expectedChatID {
		logger.Warn("Telegram Webhook: Invalid chat ID")
		return &EmptyResponse{Status: http.StatusNoContent}, nil
	}

	parts := strings.SplitN(callback.Data, ":", 2)
	if len(parts) != 2 {
		logger.Warn("Invalid callback data format: %s", callback.Data)
		return &EmptyResponse{Status: http.StatusNoContent}, nil
	}

	action, entityID := parts[0], parts[1]

	var status, statusText string
	switch action {
	case "approve":
		status, statusText = "approved", "✅ Подтверждено"
	case "reject":
		status, statusText = "rejected", "❌ Отклонено"
	case "approve_answer":
		status, statusText = "approved", "✅ Подтверждено"
	case "reject_answer":
		status, statusText = "rejected", "❌ Отклонено"
	default:
		logger.Warn("Unknown action in callback: %s", action)
		return &EmptyResponse{Status: http.StatusNoContent}, nil
	}

	if action == "approve_answer" || action == "reject_answer" {
		if err := data.UpdateCommentAnswerStatus(ctx, entityID, status); err != nil {
			logger.Error("Failed to update comment answer via webhook: %v", err)
			return &EmptyResponse{Status: http.StatusNoContent}, nil
		}
	} else {
		if err := data.UpdateCommentStatus(ctx, entityID, status); err != nil {
			logger.Error("Failed to update comment via webhook: %v", err)
			return &EmptyResponse{Status: http.StatusNoContent}, nil
		}
	}

	if err := data.DeleteTelegramMessage(callback.Message.Chat.ID, callback.Message.MessageID); err != nil {
		logger.Warn("Failed to delete telegram message: %v", err)
	}

	answerURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", os.Getenv("TELEGRAM_BOT_TOKEN"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, answerURL, strings.NewReader(url.Values{
			"callback_query_id": {callback.ID},
			"text":              {statusText},
		}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error("Failed to answer callback query: %v", err)
			return
		}
		_ = resp.Body.Close()
	}()

	return &EmptyResponse{Status: http.StatusNoContent}, nil
}

func HandleUpdateDisplayName(ctx context.Context, input *UpdateDisplayNameInput) (*ProfileResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if err := requireOwner(userID, input.ProfileID); err != nil {
		return nil, err
	}

	profile, err := data.UpdateDisplayName(ctx, userID, input.Body.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrInvalidDisplayName),
			errors.Is(err, data.ErrNameEmpty),
			errors.Is(err, data.ErrNameTooLong),
			errors.Is(err, data.ErrInvalidCharacters):
			return nil, huma.Error400BadRequest("Invalid display name")
		default:
			return nil, huma.Error500InternalServerError("Failed to update profile")
		}
	}
	return &ProfileResponse{Body: *profile}, nil
}

func HandleUploadAvatar(ctx context.Context, input *UploadAvatarInput) (*ProfileResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	if err := requireOwner(userID, input.ProfileID); err != nil {
		return nil, err
	}

	imageData, err := base64.StdEncoding.DecodeString(input.Body.Image)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid base64 image")
	}

	if len(imageData) > MaxAvatarSize {
		return nil, huma.Error400BadRequest("Image too large (max 1MB)")
	}

	profile, err := data.UpdateAvatar(ctx, userID, imageData)
	if err != nil {
		if errors.Is(err, data.ErrUnsupportedFormat) {
			return nil, huma.Error400BadRequest("Unsupported format")
		}
		return nil, huma.Error500InternalServerError("Upload failed")
	}
	return &ProfileResponse{Body: *profile}, nil
}

func HandleGetCurrentUser(ctx context.Context, input *struct{}) (*ProfileResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	profile, err := data.GetProfile(ctx, userID)
	if err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}

	return &ProfileResponse{Body: *profile}, nil
}

func HandleUploadCommentImage(ctx context.Context, input *UploadCommentImageInput) (*CommentImageResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	imageData, err := base64.StdEncoding.DecodeString(input.Body.Image)
	if err != nil {
		imageData, err = base64.RawStdEncoding.DecodeString(input.Body.Image)
		if err != nil {
			logger.Warn("Comment image upload: invalid base64 from user %s: %v", userID, err)
			return nil, huma.Error400BadRequest("Invalid base64 image")
		}
	}

	if len(imageData) > MaxCommentImageSize {
		logger.Warn("Comment image upload: file too large (%d bytes) from user %s", len(imageData), userID)
		return nil, huma.Error400BadRequest("Image too large (max 5MB)")
	}

	imageURL, err := data.UploadCommentImage(ctx, userID, imageData)
	if err != nil {
		if errors.Is(err, data.ErrUnsupportedFormat) {
			logger.Warn("Comment image upload: unsupported format from user %s", userID)
			return nil, huma.Error400BadRequest("Unsupported format (JPEG, PNG, GIF)")
		}
		if errors.Is(err, data.ErrRateLimitExceeded) {
			return nil, huma.Error429TooManyRequests("Подождите перед загрузкой следующего изображения")
		}
		if errors.Is(err, data.ErrS3NotConfigured) {
			logger.Error("Comment image upload: S3 not configured")
			return nil, huma.Error500InternalServerError("Storage not configured")
		}
		logger.Error("Comment image upload failed for user %s: %v", userID, err)
		return nil, huma.Error500InternalServerError("Upload failed")
	}

	return &CommentImageResponse{
		Body: struct {
			URL string `json:"url"`
		}{URL: imageURL},
	}, nil
}

func HandleVoteComment(ctx context.Context, input *VoteCommentInput) (*VoteCommentResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	score, err := data.VoteComment(ctx, input.CommentID, userID, input.Body.Value)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrInvalidVoteValue):
			return nil, huma.Error400BadRequest("Value must be -1, 0, or 1")
		case errors.Is(err, data.ErrCommentNotFound):
			return nil, huma.Error404NotFound("Comment not found")
		default:
			return nil, huma.Error500InternalServerError("Failed to vote")
		}
	}

	return &VoteCommentResponse{
		Body: struct {
			Score    int `json:"score"`
			UserVote int `json:"user_vote"`
		}{Score: score, UserVote: input.Body.Value},
	}, nil
}

type DeleteCommentInput struct {
	CommentID string `path:"commentId" pattern:"^cmt_[a-z0-9]{8}$"`
}

func HandleDeleteComment(ctx context.Context, input *DeleteCommentInput) (*EmptyResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	err = data.DeleteComment(ctx, input.CommentID, userID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrCommentNotFound):
			return nil, huma.Error404NotFound("Comment not found")
		case errors.Is(err, data.ErrNotCommentAuthor):
			return nil, huma.Error403Forbidden("You can only delete your own comments")
		case errors.Is(err, data.ErrCannotDeleteComment):
			return nil, huma.Error400BadRequest("Only approved comments can be deleted")
		default:
			return nil, huma.Error500InternalServerError("Failed to delete comment")
		}
	}

	return &EmptyResponse{Status: 204}, nil
}

func HandleGetUserComments(ctx context.Context, input *GetUserCommentsInput) (*UserCommentsPageResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	comments, err := data.GetUserComments(ctx, userID, input.Page)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch comments")
	}

	if err := data.UpdateNotificationsLastSeen(ctx, userID); err != nil {
		logger.Warn("Failed to update notifications_last_seen for user %s: %v", userID, err)
	}

	return &UserCommentsPageResponse{Body: *comments}, nil
}

func HandleCreateCommentAnswer(ctx context.Context, input *CreateCommentAnswerInput) (*CommentAnswerResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	answerInput := models.CreateCommentAnswerInput{
		CommentID:      input.CommentID,
		Content:        input.Body.Content,
		TurnstileToken: input.Body.TurnstileToken,
	}

	answer, err := data.CreateCommentAnswer(ctx, userID, answerInput)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRateLimitExceeded):
			return nil, huma.Error429TooManyRequests("Подождите 30 секунд перед отправкой следующего ответа")
		case errors.Is(err, data.ErrCaptchaFailed):
			return nil, huma.Error400BadRequest("Captcha verification failed")
		case errors.Is(err, data.ErrInvalidAnswerLength):
			return nil, huma.Error400BadRequest("Answer must be 1-500 characters")
		case errors.Is(err, data.ErrCommentNotFound):
			return nil, huma.Error404NotFound("Comment not found")
		case errors.Is(err, data.ErrCommentNotApproved):
			return nil, huma.Error400BadRequest("Comment must be approved to answer")
		default:
			return nil, huma.Error500InternalServerError("Failed to create answer")
		}
	}
	return &CommentAnswerResponse{Body: *answer}, nil
}

func HandleDeleteCommentAnswer(ctx context.Context, input *DeleteCommentAnswerInput) (*EmptyResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	err = data.DeleteCommentAnswer(ctx, input.AnswerID, userID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrCommentAnswerNotFound):
			return nil, huma.Error404NotFound("Answer not found")
		case errors.Is(err, data.ErrNotAnswerAuthor):
			return nil, huma.Error403Forbidden("You can only delete your own answers")
		case errors.Is(err, data.ErrCannotDeleteAnswer):
			return nil, huma.Error400BadRequest("Only approved answers can be deleted")
		default:
			return nil, huma.Error500InternalServerError("Failed to delete answer")
		}
	}

	return &EmptyResponse{Status: 204}, nil
}

func HandleGetCommentStats(ctx context.Context, input *struct{}) (*CommentStatsResponse, error) {
	userID, err := requireAuth(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := data.GetUserCommentStats(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to fetch comment stats")
	}

	return &CommentStatsResponse{Body: *stats}, nil
}
