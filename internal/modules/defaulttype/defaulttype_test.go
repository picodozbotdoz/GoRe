package defaulttype

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultTypeSetsContentType(t *testing.T) {
	handler := New("text/html")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html></html>"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html")
	}
}

func TestDefaultTypeDoesNotOverrideExisting(t *testing.T) {
	handler := New("text/html")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestDefaultTypeEmpty(t *testing.T) {
	handler := New("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json, got %q", rec.Header().Get("Content-Type"))
	}
}
