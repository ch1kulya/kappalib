package auth

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ch1kulya/kappalib/internal/database"
	"github.com/ch1kulya/logger"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/jackc/pgx/v5"
)

//go:embed sql/users_get_by_oauth.sql
var queryUsersGetByOAuth string

//go:embed sql/users_create_oauth.sql
var queryUsersCreateOAuth string

const KplUserIDKey = "kpl_user_id"

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
)

type UserStore struct{}

func NewUserStore() *UserStore {
	return &UserStore{}
}

func (us *UserStore) Update(claims token.Claims) token.Claims {
	if claims.User == nil {
		return claims
	}

	oauthProvider, oauthID := parseOAuthID(claims.User.ID)
	if oauthProvider == "" || oauthID == "" {
		return claims
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var kplUserID string
	err := database.DB.QueryRow(ctx, queryUsersGetByOAuth, oauthProvider, oauthID).Scan(&kplUserID)

	if err == pgx.ErrNoRows {
		kplUserID, err = us.createUser(ctx, oauthProvider, oauthID, claims.User.Name)
		if err != nil {
			logger.Error("Failed to create OAuth user: %v", err)
			return claims
		}
		logger.Info("Created OAuth user: %s (provider: %s)", kplUserID, oauthProvider)
	} else if err != nil {
		logger.Error("Failed to lookup OAuth user: %v", err)
		return claims
	}

	if claims.User.Attributes == nil {
		claims.User.Attributes = make(map[string]any)
	}
	claims.User.Attributes[KplUserIDKey] = kplUserID

	return claims
}

func (us *UserStore) createUser(ctx context.Context, oauthProvider, oauthID, name string) (string, error) {
	displayName := name
	if displayName == "" {
		displayName = generateRandomName()
	}
	avatarSeed := generateAvatarSeed()

	var userID string
	err := database.DB.QueryRow(ctx, queryUsersCreateOAuth, displayName, avatarSeed, oauthProvider, oauthID).Scan(&userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func parseOAuthID(userID string) (provider, id string) {
	providers := []string{"google_", "github_", "yandex_"}
	for _, p := range providers {
		if len(userID) > len(p) && userID[:len(p)] == p {
			return p[:len(p)-1], userID[len(p):]
		}
	}
	return "", ""
}

func generateRandomName() string {
	adjIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	if err != nil {
		return "Неизвестный Зверь"
	}
	animalIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(animals))))
	if err != nil {
		return "Неизвестный Зверь"
	}
	return adjectives[adjIdx.Int64()] + " " + animals[animalIdx.Int64()]
}

func generateAvatarSeed() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "default"
	}
	return hex.EncodeToString(b)
}
