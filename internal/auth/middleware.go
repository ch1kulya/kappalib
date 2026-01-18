package auth

import (
	"context"
	"net/http"

	"github.com/go-pkgz/auth/v2/token"
)

type contextKey string

const UserIDContextKey contextKey = "userID"
const SessionContextKey contextKey = "sessionID"

func BridgeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := token.GetUserInfo(r)
		if err == nil && user.Attributes != nil {
			sessionID, _ := user.Attributes[SessionIDKey].(string)
			kplUserID, _ := user.Attributes[KplUserIDKey].(string)

			if sessionID != "" && kplUserID != "" {
				if validUserID, valid := ValidateSession(r.Context(), sessionID); valid && validUserID == kplUserID {
					ctx := context.WithValue(r.Context(), UserIDContextKey, kplUserID)
					ctx = context.WithValue(ctx, SessionContextKey, sessionID)
					r = r.WithContext(ctx)
				}
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

func GetSessionIDFromContext(ctx context.Context) string {
	if sessionID, ok := ctx.Value(SessionContextKey).(string); ok {
		return sessionID
	}
	return ""
}
