package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
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

//go:embed sql/sessions_create.sql
var querySessionsCreate string

//go:embed sql/sessions_verify.sql
var querySessionsVerify string

//go:embed sql/sessions_delete.sql
var querySessionsDelete string

//go:embed sql/sessions_cleanup.sql
var querySessionsCleanup string

const (
	TokenPrefix        = "kpl_"
	MaxSessionsPerUser = 10
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
	cookieValueRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-{}\[\]":,.\s]{1,500}$`)
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

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return TokenPrefix + hex.EncodeToString(b)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
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

func createSession(ctx context.Context, userID, token string) error {
	tokenHash := hashToken(token)

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(dbCtx, querySessionsCreate, userID, tokenHash)
	if err != nil {
		logger.Error("Failed to create session: %v", err)
		return err
	}

	go cleanupOldSessions(context.Background(), userID)

	return nil
}

func cleanupOldSessions(ctx context.Context, userID string) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(dbCtx, querySessionsCleanup, userID, MaxSessionsPerUser)
	if err != nil {
		logger.Warn("Failed to cleanup old sessions for user %s: %v", userID, err)
	}
}

func VerifyToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	if !strings.HasPrefix(token, TokenPrefix) {
		return "", fmt.Errorf("invalid token format")
	}

	tokenHash := hashToken(token)

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var userID string
	err := database.DB.QueryRow(dbCtx, querySessionsVerify, tokenHash).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	return userID, nil
}

func DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	tokenHash := hashToken(token)

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := database.DB.Exec(dbCtx, querySessionsDelete, tokenHash)
	return err
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
	token := generateToken()

	var profile models.ProfileWithToken
	err := database.DB.QueryRow(dbCtx,
		`INSERT INTO users (display_name, avatar_seed, cookies)
		VALUES ($1, $2, '{}')
		RETURNING id, display_name, avatar_seed, created_at`,
		displayName, avatarSeed).Scan(
		&profile.ID, &profile.DisplayName, &profile.AvatarSeed, &profile.CreatedAt)

	if err != nil {
		logger.Error("Failed to create profile: %v", err)
		return nil, err
	}

	if err := createSession(ctx, profile.ID, token); err != nil {
		return nil, err
	}

	profile.Token = token

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
	json.Unmarshal(existingJSON, &existing)

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

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET display_name = $1, last_active_at = now() WHERE id = $2`,
		validName, userID)
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
		CacheControl: "public, max-age=6000",
	})
	if err != nil {
		return nil, fmt.Errorf("s3 upload failed: %w", err)
	}

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET has_custom_avatar = true, last_active_at = now() WHERE id = $1`,
		userID)
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
