package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/kappalib/internal/models"
	"github.com/ch1kulya/logger"
)

var syncCodeConfigs = map[string]struct {
	Length   int
	Duration time.Duration
	Charset  string
}{
	"15m":  {Length: 8, Duration: 15 * time.Minute, Charset: "ABCDEFGHJKLMNPQRSTUVWXYZ23456789$"},
	"1h":   {Length: 10, Duration: 1 * time.Hour, Charset: "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789$"},
	"24h":  {Length: 12, Duration: 24 * time.Hour, Charset: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789$"},
	"7d":   {Length: 14, Duration: 7 * 24 * time.Hour, Charset: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$"},
	"30d":  {Length: 16, Duration: 30 * 24 * time.Hour, Charset: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$&*"},
	"12mo": {Length: 18, Duration: 365 * 24 * time.Hour, Charset: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$&*"},
}

func generateSyncCodeWithConfig(config struct {
	Length   int
	Duration time.Duration
	Charset  string
}) string {
	code := make([]byte, config.Length)
	for i := range code {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(config.Charset))))
		code[i] = config.Charset[idx.Int64()]
	}
	return string(code)
}

func hashSyncCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

func GenerateSyncCode(ctx context.Context, userID string, ttlKey string) (*models.SyncCodeResponse, error) {
	config, exists := syncCodeConfigs[ttlKey]
	if !exists {
		config = syncCodeConfigs["15m"]
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	syncCode := generateSyncCodeWithConfig(config)
	codeHash := hashSyncCode(syncCode)
	expiresAt := time.Now().Add(config.Duration)

	_, err := database.DB.Exec(dbCtx,
		`UPDATE users SET sync_code_hash = $1, sync_code_expires_at = $2, last_active_at = now() WHERE id = $3`,
		codeHash, expiresAt, userID)

	if err != nil {
		logger.Error("Failed to generate sync code: %v", err)
		return nil, err
	}

	logger.Info("Sync code generated for %s (ttl: %s, length: %d)", userID, ttlKey, config.Length)
	return &models.SyncCodeResponse{
		SyncCode:  syncCode,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func LoginWithSyncCode(ctx context.Context, syncCode string) (*models.LoginResponse, error) {
	decoded, err := url.QueryUnescape(syncCode)
	if err != nil {
		decoded = syncCode
	}
	syncCode = strings.TrimSpace(decoded)

	if len(syncCode) < 8 || len(syncCode) > 24 {
		return nil, fmt.Errorf("invalid sync code format")
	}

	codeHash := hashSyncCode(syncCode)

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var userID string
	var cookiesJSON []byte
	err = database.DB.QueryRow(dbCtx, `
		SELECT id, cookies
		FROM users
		WHERE sync_code_hash = $1 AND sync_code_expires_at > now()`,
		codeHash).Scan(&userID, &cookiesJSON)

	if err != nil {
		return nil, fmt.Errorf("invalid or expired sync code")
	}

	newToken := generateToken()

	_, err = database.DB.Exec(dbCtx,
		`UPDATE users SET sync_code_hash = NULL, sync_code_expires_at = NULL, last_active_at = now() WHERE id = $1`,
		userID)

	if err != nil {
		logger.Error("Failed to clear sync code on login: %v", err)
		return nil, err
	}

	if err := createSession(ctx, userID, newToken); err != nil {
		return nil, err
	}

	var profile models.ProfilePublic
	database.DB.QueryRow(dbCtx,
		`SELECT id, display_name, avatar_seed, has_custom_avatar, created_at FROM users WHERE id = $1`,
		userID).Scan(&profile.ID, &profile.DisplayName, &profile.AvatarSeed, &profile.HasCustomAvatar, &profile.CreatedAt)

	var cookies map[string]models.CookieValue
	json.Unmarshal(cookiesJSON, &cookies)

	logger.Info("Login via sync code: %s", profile.ID)
	return &models.LoginResponse{
		Profile: profile,
		Token:   newToken,
		Cookies: cookies,
	}, nil
}
