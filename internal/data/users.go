package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"maps"
	"math/big"
	"net/http"
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

var (
	adjectives = []string{
		"Неопознанный", "Загадочный", "Мистический", "Древний", "Теневой",
		"Странный", "Забытый", "Одинокий", "Тихий", "Быстрый",
		"Мудрый", "Храбрый", "Дикий", "Свободный", "Гордый",
	}
	animals = []string{
		"Шакал", "Волк", "Ворон", "Сокол", "Медведь",
		"Лис", "Ёж", "Барсук", "Рысь", "Сыч",
		"Филин", "Хорёк", "Енот", "Суслик", "Бобр",
	}
	cookieNameRegex  = regexp.MustCompile(`^kappalib_[a-z0-9_]{1,50}$`)
	cookieValueRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,200}$`)
	turnstileSecret  = os.Getenv("TURNSTILE_SECRET")
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

func generateRandomName() string {
	adjIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	animalIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(animals))))
	return fmt.Sprintf("%s %s", adjectives[adjIdx.Int64()], animals[animalIdx.Int64()])
}

func generateAvatarSeed() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateSecretToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateSyncCode() string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 8)
	for i := range code {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[idx.Int64()]
	}
	return string(code)
}

func verifyTurnstile(token string) bool {
	if turnstileSecret == "" {
		logger.Warn("TURNSTILE_SECRET not set")
		return false
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		map[string][]string{
			"secret":   {turnstileSecret},
			"response": {token},
		})
	if err != nil {
		logger.Error("Turnstile verification failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Success
}

func verifySecretToken(ctx context.Context, profileID, providedToken string) bool {
	var storedToken string
	err := database.DB.QueryRow(ctx, `SELECT secret_token FROM users WHERE id = $1`, profileID).Scan(&storedToken)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(providedToken)) == 1
}

func validateCookies(cookies map[string]models.CookieValue) map[string]models.CookieValue {
	valid := make(map[string]models.CookieValue)
	for name, cv := range cookies {
		if cookieNameRegex.MatchString(name) && cookieValueRegex.MatchString(cv.Value) {
			valid[name] = cv
		}
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

func CreateProfile(ctx context.Context, turnstileToken string) (*models.ProfileWithToken, error) {
	if !verifyTurnstile(turnstileToken) {
		return nil, fmt.Errorf("captcha verification failed")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	displayName := generateRandomName()
	avatarSeed := generateAvatarSeed()
	secretToken := generateSecretToken()

	var profile models.ProfileWithToken
	err := database.DB.QueryRow(dbCtx,
		`INSERT INTO users (secret_token, display_name, avatar_seed, cookies)
		VALUES ($1, $2, $3, '{}')
		RETURNING id, secret_token, display_name, avatar_seed, created_at`,
		secretToken, displayName, avatarSeed).Scan(
		&profile.ID, &profile.SecretToken, &profile.DisplayName, &profile.AvatarSeed, &profile.CreatedAt)

	if err != nil {
		logger.Error("Failed to create profile: %v", err)
		return nil, err
	}

	logger.Info("Profile created: %s (%s)", profile.DisplayName, profile.ID)
	return &profile, nil
}

func GetProfile(ctx context.Context, profileID string) (*models.ProfilePublic, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var profile models.ProfilePublic
	err := database.DB.QueryRow(dbCtx,
		`SELECT id, display_name, avatar_seed, has_custom_avatar, created_at FROM users WHERE id = $1`,
		profileID).Scan(&profile.ID, &profile.DisplayName, &profile.AvatarSeed, &profile.HasCustomAvatar, &profile.CreatedAt)

	if err != nil {
		return nil, err
	}

	database.DB.Exec(dbCtx, `UPDATE users SET last_active_at = now() WHERE id = $1`, profileID)

	return &profile, nil
}

func GenerateSyncCode(ctx context.Context, profileID, secretToken string) (*models.SyncCodeResponse, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !verifySecretToken(dbCtx, profileID, secretToken) {
		return nil, fmt.Errorf("invalid secret token")
	}

	syncCode := generateSyncCode()
	expiresAt := time.Now().Add(15 * time.Minute)

	_, err := database.DB.Exec(dbCtx,
		`UPDATE users SET sync_code = $1, sync_code_expires_at = $2, last_active_at = now() WHERE id = $3`,
		syncCode, expiresAt, profileID)

	if err != nil {
		logger.Error("Failed to generate sync code: %v", err)
		return nil, err
	}

	logger.Info("Sync code generated for %s", profileID)
	return &models.SyncCodeResponse{
		SyncCode:  syncCode,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func LoginWithSyncCode(ctx context.Context, syncCode string) (*models.LoginResponse, error) {
	syncCode = strings.ToUpper(strings.TrimSpace(syncCode))
	if len(syncCode) != 8 {
		return nil, fmt.Errorf("invalid sync code format")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var profile models.ProfilePublic
	var secretToken string
	var cookiesJSON []byte
	err := database.DB.QueryRow(dbCtx, `
		SELECT id, secret_token, display_name, avatar_seed, created_at, cookies
		FROM users
		WHERE sync_code = $1 AND sync_code_expires_at > now()`,
		syncCode).Scan(&profile.ID, &secretToken, &profile.DisplayName, &profile.AvatarSeed, &profile.CreatedAt, &cookiesJSON)

	if err != nil {
		return nil, fmt.Errorf("invalid or expired sync code")
	}

	var cookies map[string]models.CookieValue
	json.Unmarshal(cookiesJSON, &cookies)

	database.DB.Exec(dbCtx,
		`UPDATE users SET sync_code = NULL, sync_code_expires_at = NULL, last_active_at = now() WHERE id = $1`,
		profile.ID)

	logger.Info("Login via sync code: %s", profile.ID)
	return &models.LoginResponse{
		Profile:     profile,
		SecretToken: secretToken,
		Cookies:     cookies,
	}, nil
}

func SyncCookies(ctx context.Context, profileID, secretToken string, cookies map[string]models.CookieValue) (map[string]models.CookieValue, error) {
	validCookies := validateCookies(cookies)

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !verifySecretToken(dbCtx, profileID, secretToken) {
		return nil, fmt.Errorf("invalid secret token")
	}

	var existingJSON []byte
	err := database.DB.QueryRow(dbCtx, `SELECT cookies FROM users WHERE id = $1`, profileID).Scan(&existingJSON)
	if err != nil {
		return nil, fmt.Errorf("profile not found")
	}

	var existing map[string]models.CookieValue
	json.Unmarshal(existingJSON, &existing)

	merged := mergeCookies(existing, validCookies)
	mergedJSON, _ := json.Marshal(merged)

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET cookies = $1, last_active_at = now() WHERE id = $2`,
		mergedJSON, profileID)

	if err != nil {
		logger.Error("Failed to sync cookies: %v", err)
		return nil, err
	}

	return merged, nil
}

func DeleteProfile(ctx context.Context, profileID, secretToken string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !verifySecretToken(dbCtx, profileID, secretToken) {
		return fmt.Errorf("invalid secret token")
	}

	result, err := database.DB.Exec(dbCtx, `DELETE FROM users WHERE id = $1`, profileID)
	if err != nil {
		logger.Error("Failed to delete profile: %v", err)
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("profile not found")
	}

	logger.Info("Profile deleted: %s", profileID)
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

func UpdateDisplayName(ctx context.Context, profileID, secretToken, newName string) (*models.ProfilePublic, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if !verifySecretToken(dbCtx, profileID, secretToken) {
		return nil, fmt.Errorf("invalid secret token")
	}

	validName, err := ValidateDisplayName(newName)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET display_name = $1, last_active_at = now() WHERE id = $2`,
		validName, profileID)
	if err != nil {
		return nil, err
	}

	logger.Debug("Updated display name for %s: %s", profileID, validName)

	return GetProfile(ctx, profileID)
}

func UpdateAvatar(ctx context.Context, profileID, secretToken string, imageData []byte) (*models.ProfilePublic, error) {
	if minioClient == nil {
		return nil, fmt.Errorf("s3 not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if !verifySecretToken(dbCtx, profileID, secretToken) {
		return nil, fmt.Errorf("invalid secret token")
	}

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

	key := fmt.Sprintf("avatars/%s.jpg", profileID)
	reader := bytes.NewReader(imgData)

	_, err = minioClient.PutObject(ctx, s3Bucket, key, reader, int64(len(imgData)), minio.PutObjectOptions{
		ContentType:  "image/jpeg",
		CacheControl: "public, max-age=3600",
	})
	if err != nil {
		return nil, fmt.Errorf("s3 upload failed: %w", err)
	}

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET has_custom_avatar = true, last_active_at = now() WHERE id = $1`,
		profileID)
	if err != nil {
		return nil, err
	}

	return GetProfile(ctx, profileID)
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
