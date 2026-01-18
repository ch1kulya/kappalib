package auth

import (
	"net/http"
	"os"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/token"
)

type Config struct {
	Secret string
	URL    string
	Secure bool
}

type Service struct {
	auth       *auth.Service
	AuthRoutes http.Handler
}

func NewService(cfg Config, userStore *UserStore) *Service {
	opts := auth.Opts{
		SecretReader: token.SecretFunc(func(string) (string, error) {
			return cfg.Secret, nil
		}),
		TokenDuration:  7 * 24 * time.Hour,
		CookieDuration: 30 * 24 * time.Hour,
		ClaimsUpd:      token.ClaimsUpdFunc(userStore.Update),
		SecureCookies:  cfg.Secure,
		DisableXSRF:    true,
		SameSiteCookie: http.SameSiteLaxMode,
		JWTCookieName:  "kpl_session",
		Issuer:         "kappalib",
		URL:            cfg.URL,
		AvatarStore:    avatar.NewNoOp(),
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			return claims.User != nil
		}),
	}

	svc := auth.NewService(opts)

	if clientID, clientSecret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"); clientID != "" && clientSecret != "" {
		svc.AddProvider("google", clientID, clientSecret)
	}

	if clientID, clientSecret := os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"); clientID != "" && clientSecret != "" {
		svc.AddProvider("github", clientID, clientSecret)
	}

	if clientID, clientSecret := os.Getenv("YANDEX_CLIENT_ID"), os.Getenv("YANDEX_CLIENT_SECRET"); clientID != "" && clientSecret != "" {
		svc.AddProvider("yandex", clientID, clientSecret)
	}

	authRoutes, _ := svc.Handlers()

	return &Service{
		auth:       svc,
		AuthRoutes: authRoutes,
	}
}

func (s *Service) Middleware() func(http.Handler) http.Handler {
	m := s.auth.Middleware()
	return m.Trace
}

func (s *Service) TokenService() *token.Service {
	return s.auth.TokenService()
}
