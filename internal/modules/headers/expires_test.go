package headers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseExpiresAt(t *testing.T) {
	tests := []struct {
		expr  string
		valid bool
	}{
		{"@24:00", false},
		{"@25:00", false},
		{"@12:60", false},
		{"@12:30", true},
		{"@00:00", true},
		{"@23:59", true},
	}
	for _, tc := range tests {
		_, ok := ParseExpires(tc.expr)
		if ok != tc.valid {
			t.Errorf("ParseExpires(%q) = %v, want %v", tc.expr, ok, tc.valid)
		}
	}
}

func TestParseExpiresRelative(t *testing.T) {
	tests := []struct {
		expr  string
		valid bool
	}{
		{"access plus 1 hour", true},
		{"access plus 2 hours", true},
		{"access plus 30 minutes", true},
		{"access plus 1 day", true},
		{"access plus 1 week", true},
		{"access plus 1 month", true},
		{"access plus 1 year", true},
		{"access plus 1 year 2 months 3 days", true},
		{"access plus", false},
		{"invalid", false},
		{"", false},
	}
	for _, tc := range tests {
		_, ok := ParseExpires(tc.expr)
		if ok != tc.valid {
			t.Errorf("ParseExpires(%q) = %v, want %v", tc.expr, ok, tc.valid)
		}
	}
}

func TestExpiresHandlerSetsHeaders(t *testing.T) {
	handler := NewExpiresHandler("access plus 1 hour")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Expires") == "" {
		t.Fatal("expected Expires header to be set")
	}
	if !strings.HasPrefix(w.Header().Get("Cache-Control"), "max-age=") {
		t.Fatalf("expected Cache-Control header, got %q", w.Header().Get("Cache-Control"))
	}
}

func TestExpiresHandlerAtTime(t *testing.T) {
	handler := NewExpiresHandler("@12:30")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	expires := w.Header().Get("Expires")
	if expires == "" {
		t.Fatal("expected Expires header")
	}
	if !strings.HasSuffix(expires, "GMT") {
		t.Fatalf("expected Expires header in GMT format, got %q", expires)
	}
}

func TestExpiresHandlerEmptyExpr(t *testing.T) {
	handler := NewExpiresHandler("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Expires") != "" {
		t.Fatal("expected no Expires header for empty expression")
	}
}

func TestExpiresHandlerPreservesExistingCacheControl(t *testing.T) {
	handler := NewExpiresHandler("access plus 1 hour")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("expected existing Cache-Control preserved, got %q", w.Header().Get("Cache-Control"))
	}
}
