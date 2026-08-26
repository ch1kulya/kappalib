package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ch1kulya/logger"
	"golang.org/x/time/rate"
)

const (
	maxVisitors            = 9999
	visitorCleanupInterval = 2 * time.Minute
	visitorExpiry          = 5 * time.Minute
	visitorEmergencyExpiry = 1 * time.Minute
	defaultRateLimit       = 3
	defaultBurst           = 9
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	apiToken string
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		apiToken: os.Getenv("API_TOKEN"),
	}
	if rl.apiToken == "" {
		logger.Warn("API_TOKEN is not set")
	}
	go rl.cleanupVisitors()
	return rl
}

func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(visitorCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > visitorExpiry {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v, exists := rl.visitors[ip]; exists {
		v.lastSeen = time.Now()
		return v.limiter
	}

	if len(rl.visitors) >= maxVisitors {
		for k, v := range rl.visitors {
			if time.Since(v.lastSeen) > visitorEmergencyExpiry {
				delete(rl.visitors, k)
			}
		}
		if len(rl.visitors) >= maxVisitors {
			return rate.NewLimiter(rate.Limit(1), 1)
		}
	}

	limiter := rate.NewLimiter(rate.Limit(defaultRateLimit), defaultBurst)
	rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl.apiToken != "" {
				clientToken := r.Header.Get("X-Service-Token")
				if subtle.ConstantTimeCompare([]byte(clientToken), []byte(rl.apiToken)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}

			ip := getClientIP(r)
			if !rl.getVisitor(ip).Allow() {
				logger.Warn("Rate limit exceeded for IP: %s", ip)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "10")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"Too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CorsMiddleware(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowedOrigin == "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Service-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

var noCachePrefixes = []string{
	"/api/profile/",
	"/api/webhook/",
}

func hasAuth(r *http.Request) bool {
	if _, err := r.Cookie(SessionCookieName); err == nil {
		return true
	}
	if r.Header.Get("Authorization") != "" {
		return true
	}
	return false
}

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		noCache := false
		for _, prefix := range noCachePrefixes {
			if strings.HasPrefix(path, prefix) {
				noCache = true
				break
			}
		}

		if noCache {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else if r.Method == http.MethodGet {
			w.Header().Set("Vary", "Cookie")
			if hasAuth(r) {
				w.Header().Set("Cache-Control", "private, max-age=60")
			} else {
				w.Header().Set("Cache-Control", "public, max-age=6000")
			}
		}

		next.ServeHTTP(w, r)
	})
}

