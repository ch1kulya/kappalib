package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "xff multiple ips",
			xff:        "203.0.113.195, 70.41.3.18",
			remoteAddr: "127.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "xff single ip",
			xff:        "203.0.113.195",
			remoteAddr: "127.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "xri fallback",
			xri:        "198.51.100.1",
			remoteAddr: "127.0.0.1:1234",
			expected:   "198.51.100.1",
		},
		{
			name:       "remote addr with port",
			remoteAddr: "192.0.2.1:54321",
			expected:   "192.0.2.1",
		},
		{
			name:       "remote addr without port",
			remoteAddr: "192.0.2.1",
			expected:   "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				r.Header.Set("X-Real-IP", tt.xri)
			}
			r.RemoteAddr = tt.remoteAddr

			actual := getClientIP(r)
			if actual != tt.expected {
				t.Errorf("getClientIP() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestRateLimitMiddleware_AllowAndBlock(t *testing.T) {
	rl := NewRateLimiter()
	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.0.2.100"
	for i := 0; i < defaultBurst; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", ip)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", ip)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 after burst, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if string(body) != `{"error": "Too many requests"}` {
		t.Errorf("unexpected body: %q", string(body))
	}
}

func TestRateLimitMiddleware_IndependentIPs(t *testing.T) {
	rl := NewRateLimiter()
	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip1 := "192.0.2.101"
	ip2 := "192.0.2.102"

	for i := 0; i < defaultBurst; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", ip1)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	reqBlocked := httptest.NewRequest(http.MethodGet, "/", nil)
	reqBlocked.Header.Set("X-Real-IP", ip1)
	recBlocked := httptest.NewRecorder()
	handler.ServeHTTP(recBlocked, reqBlocked)
	if recBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("ip1 expected status 429, got %d", recBlocked.Code)
	}

	reqAllowed := httptest.NewRequest(http.MethodGet, "/", nil)
	reqAllowed.Header.Set("X-Real-IP", ip2)
	recAllowed := httptest.NewRecorder()
	handler.ServeHTTP(recAllowed, reqAllowed)
	if recAllowed.Code != http.StatusOK {
		t.Fatalf("ip2 expected status 200, got %d", recAllowed.Code)
	}
}

func TestRateLimiter_FallbackWhenMaxVisitors(t *testing.T) {
	rl := NewRateLimiter()

	rl.mu.Lock()
	rl.lastEmergencyCleanup = time.Now()
	for i := 0; i < maxVisitors; i++ {
		rl.visitors[fmt.Sprintf("10.0.%d.%d", i/256, i%256)] = &visitor{
			limiter:  rl.fallbackLimiter,
			lastSeen: time.Now(),
		}
	}
	rl.mu.Unlock()

	lim := rl.getVisitor("192.0.2.200")
	if lim != rl.fallbackLimiter {
		t.Fatal("expected fallbackLimiter when max visitors reached")
	}

	if !lim.Allow() {
		t.Fatal("expected fallbackLimiter to allow first token")
	}
	if lim.Allow() {
		t.Fatal("expected fallbackLimiter to block second immediate token")
	}
}

func TestRateLimiter_EmergencyCleanup(t *testing.T) {
	rl := NewRateLimiter()

	rl.mu.Lock()
	rl.lastEmergencyCleanup = time.Now().Add(-15 * time.Second)
	for i := 0; i < maxVisitors; i++ {
		rl.visitors[fmt.Sprintf("10.0.%d.%d", i/256, i%256)] = &visitor{
			limiter:  rl.fallbackLimiter,
			lastSeen: time.Now().Add(-2 * time.Minute),
		}
	}
	rl.mu.Unlock()

	lim := rl.getVisitor("192.0.2.201")
	if lim == rl.fallbackLimiter {
		t.Fatal("expected emergency cleanup to free slots and return fresh limiter")
	}

	rl.mu.Lock()
	count := len(rl.visitors)
	rl.mu.Unlock()

	if count != 1 {
		t.Fatalf("expected visitor map to contain only 1 new visitor, got %d", count)
	}
}
