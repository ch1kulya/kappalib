package data

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	logger "github.com/ch1kulya/simple-logger"
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
		`SELECT id, display_name, avatar_seed, created_at FROM users WHERE id = $1`,
		profileID).Scan(&profile.ID, &profile.DisplayName, &profile.AvatarSeed, &profile.CreatedAt)

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
