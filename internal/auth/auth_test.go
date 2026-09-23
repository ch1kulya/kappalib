package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pkgz/auth/v2/token"
)

func TestXSRFProtection(t *testing.T) {
	svc := NewService(Config{Secret: "test-secret", URL: "http://localhost"}, NewUserStore())

	claims := token.Claims{User: &token.User{ID: "user", Name: "user"}}
	claims.ID = "xsrf-id"
	claims.Audience = []string{"kappalib"}

	rec := httptest.NewRecorder()
	if _, err := svc.TokenService().Set(rec, claims); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	cookies := rec.Result().Cookies()
	var xsrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "XSRF-TOKEN" {
			xsrfCookie = c
		}
	}
	if xsrfCookie == nil {
		t.Fatal("XSRF-TOKEN cookie not set")
	}
	if xsrfCookie.Value != claims.ID {
		t.Fatalf("XSRF-TOKEN = %q, want %q", xsrfCookie.Value, claims.ID)
	}
	if xsrfCookie.HttpOnly {
		t.Fatal("XSRF-TOKEN cookie must be readable by scripts")
	}

	tests := []struct {
		name    string
		method  string
		header  string
		wantErr bool
	}{
		{name: "get without header", method: http.MethodGet},
		{name: "head without header", method: http.MethodHead},
		{name: "options without header", method: http.MethodOptions},
		{name: "post without header", method: http.MethodPost, wantErr: true},
		{name: "put without header", method: http.MethodPut, wantErr: true},
		{name: "patch without header", method: http.MethodPatch, wantErr: true},
		{name: "delete without header", method: http.MethodDelete, wantErr: true},
		{name: "post with wrong header", method: http.MethodPost, header: "forged", wantErr: true},
		{name: "post with valid header", method: http.MethodPost, header: xsrfCookie.Value},
		{name: "delete with valid header", method: http.MethodDelete, header: xsrfCookie.Value},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", nil)
			for _, c := range cookies {
				r.AddCookie(c)
			}
			if tt.header != "" {
				r.Header.Set("X-XSRF-TOKEN", tt.header)
			}

			_, _, err := svc.TokenService().Get(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
