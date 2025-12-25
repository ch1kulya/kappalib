package web

import (
	"net/http"
	"strings"
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
	for {
		time.Sleep(2 * time.Minute)
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
			ip := r.RemoteAddr
			if strings.Contains(ip, ":") {
				ip = strings.Split(ip, ":")[0]
			}
			limiter := rl.getVisitor(ip)
			if !limiter.Allow() {
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

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		csp := "default-src 'self'; " +
			"connect-src 'self' https://stats.ch1kulya.ru https://cdn.jsdelivr.net/; " +
			"img-src 'self' https: data:; " +
			"script-src 'self' 'unsafe-inline' https://stats.ch1kulya.ru https://challenges.cloudflare.com https://cdn.jsdelivr.net; " +
			"frame-src 'self' https://challenges.cloudflare.com; " +
			"style-src 'self' 'unsafe-inline' https://rsms.me; " +
			"font-src 'self' data: https://rsms.me;"

		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}

func StaticCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2628000, immutable")
		next.ServeHTTP(w, r)
	})
}
