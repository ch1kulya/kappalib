package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ch1kulya/logger"
	"golang.org/x/time/rate"
)

const maxVisitors = 9999

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
	go rl.cleanupVisitors()
	return rl
}

func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		if len(rl.visitors) >= maxVisitors {
			for k, val := range rl.visitors {
				if time.Since(val.lastSeen) > 1*time.Minute {
					delete(rl.visitors, k)
				}
			}
			if len(rl.visitors) >= maxVisitors {
				return rate.NewLimiter(rate.Limit(1), 1)
			}
		}

		limiter := rate.NewLimiter(rate.Limit(3), 9)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func RateLimitMiddleware(rl *RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientToken := r.Header.Get("X-Service-Token")
			isAuthorized := rl.apiToken != "" && subtle.ConstantTimeCompare([]byte(clientToken), []byte(rl.apiToken)) == 1
			if isAuthorized {
				next.ServeHTTP(w, r)
				return
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if !rl.getVisitor(ip).Allow() {
				logger.Warn("Rate limit exceeded for IP: %s", ip)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CorsMiddleware(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		logger.Warn("ALLOWED_ORIGIN is not set!")
		allowedOrigin = "*"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Service-Token, X-Profile-ID, X-Secret-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		next.ServeHTTP(w, r)
	})
}
