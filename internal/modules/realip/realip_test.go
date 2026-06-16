package realip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRealIPFromXForwardedFor(t *testing.T) {
	handler := New("X-Forwarded-For, X-Real-IP")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Result", r.RemoteAddr)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Result") != "203.0.113.50:0" {
		t.Errorf("RemoteAddr = %q, want %q", rec.Header().Get("X-Result"), "203.0.113.50:0")
	}
}

func TestRealIPFromXRealIP(t *testing.T) {
	handler := New("X-Forwarded-For, X-Real-IP")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Result", r.RemoteAddr)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "198.51.100.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Result") != "198.51.100.1:0" {
		t.Errorf("RemoteAddr = %q, want %q", rec.Header().Get("X-Result"), "198.51.100.1:0")
	}
}

func TestRealIPIgnoresInvalidIP(t *testing.T) {
	handler := New("X-Forwarded-For")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Result", r.RemoteAddr)
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "not-an-ip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Result") != "10.0.0.1:1234" {
		t.Errorf("RemoteAddr = %q, should remain original", rec.Header().Get("X-Result"))
	}
}

func TestExtractFirstIP(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"203.0.113.50, 70.41.3.18", "203.0.113.50"},
		{"198.51.100.1", "198.51.100.1"},
		{"10.0.0.1:8080, 10.0.0.2", "10.0.0.1"},
		{"", ""},
		{"invalid", ""},
	}

	for _, tt := range tests {
		got := extractFirstIP(tt.header)
		if got != tt.want {
			t.Errorf("extractFirstIP(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}
