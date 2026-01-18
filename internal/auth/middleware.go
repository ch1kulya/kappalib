package auth

import (
	"context"
	"net/http"

	"github.com/go-pkgz/auth/v2/token"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

func BridgeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := token.GetUserInfo(r)
		if err == nil && user.Attributes != nil {
			if kplUserID, ok := user.Attributes[KplUserIDKey].(string); ok && kplUserID != "" {
				ctx := context.WithValue(r.Context(), UserIDContextKey, kplUserID)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDContextKey).(string); ok {
		return userID
	}
	return ""
}
