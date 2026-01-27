package web

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
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
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
	}
	go rl.cleanupLoop()
	return rl
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

		limiter := rate.NewLimiter(rate.Limit(10), 20)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) cleanupLoop() {
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

func RateLimitMiddleware(rl *RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			limiter := rl.getVisitor(ip)
			if !limiter.Allow() {
				logger.Warn("Rate limit exceeded for IP: %s", ip)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error": "Too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func WwwRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if after, ok := strings.CutPrefix(r.Host, "www."); ok {
			host := after
			target := "https://" + host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	airHash := os.Getenv("AIR_PROXY_HASH")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := generateNonce()

		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		var cspBuilder strings.Builder
		cspBuilder.WriteString("default-src 'self'; ")
		cspBuilder.WriteString("connect-src 'self' https://stats.ch1kulya.ru https://cdn.jsdelivr.net/; ")
		cspBuilder.WriteString("img-src 'self' https: data:; ")
		cspBuilder.WriteString(fmt.Sprintf("script-src 'self' 'nonce-%s' https://stats.ch1kulya.ru https://challenges.cloudflare.com https://cdn.jsdelivr.net", nonce))
		if airHash != "" {
			cspBuilder.WriteString(fmt.Sprintf(" '%s'", airHash))
		}
		cspBuilder.WriteString("; ")
		cspBuilder.WriteString("frame-src 'self' https://challenges.cloudflare.com; ")
		cspBuilder.WriteString("style-src 'self' 'unsafe-inline' https://rsms.me https://cdn.jsdelivr.net; ")
		cspBuilder.WriteString("font-src 'self' data: https://rsms.me https://fonts.scalar.com https://cdn.jsdelivr.net; ")

		w.Header().Set("Content-Security-Policy", cspBuilder.String())
		ctx := templ.WithNonce(r.Context(), nonce)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func StaticCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2628000, immutable")
		next.ServeHTTP(w, r)
	})
}

func UserAgentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.UserAgent()
		if ua == "" {
			ua = "[Empty]"
		}
		logger.Debug("User-Agent: %s", ua)
		next.ServeHTTP(w, r)
	})
}
