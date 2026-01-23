package data

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"maps"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/microcosm-cc/bluemonday"
	"github.com/minio/minio-go/v7"
	"golang.org/x/image/draw"

	"github.com/ch1kulya/logger"
)

//go:embed sql/users_get.sql
var queryUsersGet string

//go:embed sql/users_update_avatar.sql
var queryUsersUpdateAvatar string

//go:embed sql/users_update_display_name.sql
var queryUsersUpdateDisplayName string

var (
	cookieNameRegex  = regexp.MustCompile(`^kappalib_[a-z0-9_]{1,50}$`)
	simpleValueRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,100}$`)
	novelIDRegex     = regexp.MustCompile(`^nvl_[a-z0-9]{8}$`)
	chapterIDRegex   = regexp.MustCompile(`^chp_[a-z0-9]{8}$`)
)

var (
	minioClient        *minio.Client
	s3Bucket           = os.Getenv("S3_BUCKET")
	imageProcessingSem = make(chan struct{}, 5)
	displayNameRegex   = regexp.MustCompile(`^[\p{L}\p{N} ]+$`)
	multiSpaceRegex    = regexp.MustCompile(`\s+`)
	strictPolicy       = bluemonday.StrictPolicy()
)

var ErrUnsupportedFormat = fmt.Errorf("unsupported image format")

type progressNovel struct {
	ChapterID  string `json:"chapterId"`
	ChapterNum int    `json:"chapterNum"`
	ReadAt     int64  `json:"readAt"`
}

type progressLastRead struct {
	NovelID       string `json:"novelId"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	CoverURL      string `json:"coverUrl"`
	ChapterID     string `json:"chapterId"`
	ChapterNum    int    `json:"chapterNum"`
	TotalChapters int    `json:"totalChapters"`
	ReadAt        int64  `json:"readAt"`
}

type progressCookieSchema struct {
	Novels   map[string]progressNovel `json:"novels"`
	LastRead *progressLastRead        `json:"lastRead"`
}

type readerSettingsSchema struct {
	Theme       string `json:"theme"`
	ColorScheme string `json:"colorScheme"`
	FontSize    int    `json:"fontSize"`
	FontFamily  string `json:"fontFamily"`
	Indent      int    `json:"indent"`
	Density     string `json:"density"`
	Justify     bool   `json:"justify"`
}

var cookieValidators = map[string]func(string) bool{
	"kappalib_progress": func(v string) bool {
		var p progressCookieSchema
		if err := json.Unmarshal([]byte(v), &p); err != nil {
			return false
		}
		if p.Novels == nil {
			return false
		}
		for id, novel := range p.Novels {
			if !novelIDRegex.MatchString(id) {
				return false
			}
			if !chapterIDRegex.MatchString(novel.ChapterID) {
				return false
			}
			if novel.ChapterNum < 1 || novel.ReadAt < 0 {
				return false
			}
		}
		if p.LastRead != nil {
			if !novelIDRegex.MatchString(p.LastRead.NovelID) {
				return false
			}
			if !chapterIDRegex.MatchString(p.LastRead.ChapterID) {
				return false
			}
			if p.LastRead.ChapterNum < 1 || p.LastRead.TotalChapters < 1 {
				return false
			}
		}
		return true
	},
	"kappalib_reader_settings": func(v string) bool {
		var s readerSettingsSchema
		return json.Unmarshal([]byte(v), &s) == nil
	},
	"kappalib_chapter_sort": func(v string) bool {
		return v == "asc" || v == "desc"
	},
	"kappalib_catalog_sort": func(v string) bool {
		valid := []string{"popular", "oldest", "newest", "created", "large", "small", "alphabet", "relevance"}
		for _, opt := range valid {
			if v == opt {
				return true
			}
		}
		return false
	},
}

func validateCookies(cookies map[string]models.CookieValue) map[string]models.CookieValue {
	valid := make(map[string]models.CookieValue)
	for name, cv := range cookies {
		if !cookieNameRegex.MatchString(name) {
			continue
		}
		if validator, ok := cookieValidators[name]; ok {
			if !validator(cv.Value) {
				continue
			}
		} else if !simpleValueRegex.MatchString(cv.Value) {
			continue
		}
		valid[name] = cv
	}
	return valid
}

func mergeCookies(existing, incoming map[string]models.CookieValue) map[string]models.CookieValue {
	result := make(map[string]models.CookieValue)
	maps.Copy(result, existing)

	for name, incomingCv := range incoming {
		if existingCv, exists := result[name]; exists {
			if incomingCv.UpdatedAt > existingCv.UpdatedAt {
				result[name] = incomingCv
			}
		} else {
			result[name] = incomingCv
		}
	}

	return result
}

func GetProfile(ctx context.Context, profileID string) (*models.ProfilePublic, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var profile models.ProfilePublic
	var avatarUpdatedAt time.Time
	err := database.DB.QueryRow(dbCtx, queryUsersGet, profileID).Scan(
		&profile.ID, &profile.DisplayName, &profile.AvatarSeed, &profile.HasCustomAvatar, &avatarUpdatedAt, &profile.CreatedAt)

	if err != nil {
		return nil, err
	}

	profile.AvatarUpdatedAt = avatarUpdatedAt.Unix()

	if _, err := database.DB.Exec(dbCtx, `UPDATE users SET last_active_at = now() WHERE id = $1`, profileID); err != nil {
		logger.Warn("Failed to update last_active_at for user %s: %v", profileID, err)
	}

	return &profile, nil
}

func SyncCookies(ctx context.Context, userID string, cookies map[string]models.CookieValue) (map[string]models.CookieValue, error) {
	validCookies := validateCookies(cookies)

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var existingJSON []byte
	err := database.DB.QueryRow(dbCtx, `SELECT cookies FROM users WHERE id = $1`, userID).Scan(&existingJSON)
	if err != nil {
		return nil, fmt.Errorf("profile not found")
	}

	var existing map[string]models.CookieValue
	if err := json.Unmarshal(existingJSON, &existing); err != nil {
		existing = make(map[string]models.CookieValue)
	}

	merged := mergeCookies(existing, validCookies)
	mergedJSON, _ := json.Marshal(merged)

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET cookies = $1, last_active_at = now() WHERE id = $2`,
		mergedJSON, userID)

	if err != nil {
		logger.Error("Failed to sync cookies: %v", err)
		return nil, err
	}

	logger.Debug("Synced cookies for user: %v", userID)

	return merged, nil
}

func DeleteProfile(ctx context.Context, userID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := database.DB.Exec(dbCtx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		logger.Error("Failed to delete profile: %v", err)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("profile not found")
	}

	logger.Info("Profile deleted: %s", userID)
	return nil
}

func ValidateDisplayName(name string) (string, error) {
	name = strictPolicy.Sanitize(name)
	name = multiSpaceRegex.ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	if len(name) == 0 {
		return "", fmt.Errorf("name is empty")
	}

	runeCount := len([]rune(name))
	if runeCount > 15 {
		return "", fmt.Errorf("name too long")
	}

	if !displayNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid characters")
	}

	return name, nil
}

func UpdateDisplayName(ctx context.Context, userID, newName string) (*models.ProfilePublic, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	validName, err := ValidateDisplayName(newName)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	_, err = database.DB.Exec(dbCtx, queryUsersUpdateDisplayName, validName, userID)
	if err != nil {
		return nil, err
	}

	logger.Debug("Updated display name for %s: %s", userID, validName)

	return GetProfile(ctx, userID)
}

func UpdateAvatar(ctx context.Context, userID string, imageData []byte) (*models.ProfilePublic, error) {
	if minioClient == nil {
		return nil, fmt.Errorf("s3 not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case imageProcessingSem <- struct{}{}:
		defer func() { <-imageProcessingSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	imgData, err := processAvatar(imageData)
	if err != nil {
		if errors.Is(err, ErrUnsupportedFormat) {
			return nil, fmt.Errorf("unsupported format")
		}
		return nil, fmt.Errorf("image processing failed: %w", err)
	}

	key := fmt.Sprintf("avatars/%s.jpg", userID)
	reader := bytes.NewReader(imgData)

	_, err = minioClient.PutObject(ctx, s3Bucket, key, reader, int64(len(imgData)), minio.PutObjectOptions{
		ContentType:  "image/jpeg",
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return nil, fmt.Errorf("s3 upload failed: %w", err)
	}

	_, err = database.DB.Exec(dbCtx, queryUsersUpdateAvatar, userID)
	if err != nil {
		return nil, err
	}

	return GetProfile(ctx, userID)
}

func processAvatar(data []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupportedFormat
	}

	if format != "jpeg" && format != "png" {
		return nil, ErrUnsupportedFormat
	}

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	var cropRect image.Rectangle
	if srcW > srcH {
		offset := (srcW - srcH) / 2
		cropRect = image.Rect(offset, 0, offset+srcH, srcH)
	} else {
		offset := (srcH - srcW) / 2
		cropRect = image.Rect(0, offset, srcW, offset+srcW)
	}

	cropped := image.NewRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	draw.Draw(cropped, cropped.Bounds(), img, cropRect.Min, draw.Src)

	resized := image.NewRGBA(image.Rect(0, 0, 250, 250))
	draw.CatmullRom.Scale(resized, resized.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
