package api

import (
	"context"

	"kappalib/internal/data"
	"kappalib/internal/database"
	"kappalib/internal/models"

	"github.com/danielgtaylor/huma/v2"
)

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

type CreateProfileInput struct {
	Body struct {
		TurnstileToken string `json:"turnstile_token" minLength:"1"`
	}
}

type LoginInput struct {
	Body struct {
		SyncCode string `json:"sync_code" minLength:"8" maxLength:"8"`
	}
}

type SyncCookiesInput struct {
	ProfileID   string `header:"X-Profile-ID" required:"true"`
	SecretToken string `header:"X-Secret-Token" required:"true"`
	Body        struct {
		Cookies map[string]models.CookieValue `json:"cookies"`
	}
}

type ProfileIDInput struct {
	ProfileID string `path:"id"`
}

type AuthenticatedProfileInput struct {
	ProfileID   string `path:"id"`
	SecretToken string `header:"X-Secret-Token" required:"true"`
}

type APIStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

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

func HandleCreateProfile(ctx context.Context, input *CreateProfileInput) (*struct{ Body any }, error) {
	profile, err := data.CreateProfile(ctx, input.Body.TurnstileToken)
	if err != nil {
		return nil, huma.Error400BadRequest("Captcha verification failed")
	}
	return &struct{ Body any }{Body: profile}, nil
}

func HandleGetProfile(ctx context.Context, input *ProfileIDInput) (*struct{ Body any }, error) {
	profile, err := data.GetProfile(ctx, input.ProfileID)
	if err != nil {
		return nil, huma.Error404NotFound("Profile not found")
	}
	return &struct{ Body any }{Body: profile}, nil
}

func HandleGenerateSyncCode(ctx context.Context, input *AuthenticatedProfileInput) (*struct{ Body any }, error) {
	result, err := data.GenerateSyncCode(ctx, input.ProfileID, input.SecretToken)
	if err != nil {
		return nil, huma.Error403Forbidden("Invalid secret token")
	}
	return &struct{ Body any }{Body: result}, nil
}

func HandleLogin(ctx context.Context, input *LoginInput) (*struct{ Body any }, error) {
	result, err := data.LoginWithSyncCode(ctx, input.Body.SyncCode)
	if err != nil {
		return nil, huma.Error404NotFound("Invalid or expired sync code")
	}
	return &struct{ Body any }{Body: result}, nil
}

func HandleSyncCookies(ctx context.Context, input *SyncCookiesInput) (*struct{ Body any }, error) {
	if input.ProfileID == "" {
		return nil, huma.Error401Unauthorized("X-Profile-ID header required")
	}

	result, err := data.SyncCookies(ctx, input.ProfileID, input.SecretToken, input.Body.Cookies)
	if err != nil {
		return nil, huma.Error403Forbidden("Invalid secret token")
	}
	return &struct{ Body any }{Body: result}, nil
}

func HandleDeleteProfile(ctx context.Context, input *AuthenticatedProfileInput) (*struct{}, error) {
	err := data.DeleteProfile(ctx, input.ProfileID, input.SecretToken)
	if err != nil {
		if err.Error() == "invalid secret token" {
			return nil, huma.Error403Forbidden("Invalid secret token")
		}
		return nil, huma.Error404NotFound("Profile not found")
	}
	return &struct{}{}, nil
}
