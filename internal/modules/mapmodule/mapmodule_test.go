package mapmodule

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/gore/internal/config"
)

func TestMapHeaderMatching(t *testing.T) {
	handler := New([]config.MapConfig{
		{
			Source: "$http_user_agent",
			Target: "X-Device",
			Rules: []config.MapRule{
				{Pattern: "(?i)mobile", Value: "mobile"},
			},
			Default: "desktop",
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Result", r.Header.Get("X-Device"))
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Mobile Safari")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Result") != "mobile" {
		t.Errorf("X-Result = %q, want mobile", rec.Header().Get("X-Result"))
	}
}

func TestMapHeaderDefault(t *testing.T) {
	handler := New([]config.MapConfig{
		{
			Source: "$http_user_agent",
			Target: "X-Device",
			Rules: []config.MapRule{
				{Pattern: "mobile", Value: "mobile"},
			},
			Default: "desktop",
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Result", r.Header.Get("X-Device"))
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Result") != "desktop" {
		t.Errorf("X-Result = %q, want desktop", rec.Header().Get("X-Result"))
	}
}

func TestMapEmpty(t *testing.T) {
	called := false
	handler := New(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called")
	}
}

func TestResolveSource(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "TestAgent")
	req.RemoteAddr = "10.0.0.1:80"
	req.Host = "example.com"

	tests := []struct {
		source string
		want   string
	}{
		{"$http_user_agent", "TestAgent"},
		{"$remote_addr", "10.0.0.1:80"},
		{"$host", "example.com"},
		{"$method", "GET"},
	}

	for _, tt := range tests {
		got := resolveSource(req, tt.source)
		if got != tt.want {
			t.Errorf("resolveSource(%q) = %q, want %q", tt.source, got, tt.want)
		}
	}
}
