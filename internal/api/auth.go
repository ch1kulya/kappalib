package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/ch1kulya/kappalib/internal/data"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDContextKey).(string); ok {
		return userID
	}
	return ""
}

func extractToken(r *http.Request) string {
	if cookie, err := r.Cookie("kpl_session"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	authHeader := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return after
	}

	return ""
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)

		if token != "" {
			userID, err := data.VerifyToken(r.Context(), token)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromContext(r.Context())
		if userID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Authentication required"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
