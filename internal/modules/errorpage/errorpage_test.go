package errorpage

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorPage(t *testing.T) {
	pages := map[int]string{
		404: "custom 404 page",
		500: "custom 500 page",
	}
	handler := New(pages)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("original"))
	}))

	req := httptest.NewRequest("GET", "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec.Body.String() != "custom 404 page" {
		t.Fatalf("expected custom body, got %q", rec.Body.String())
	}
}

func TestErrorPageNoMatch(t *testing.T) {
	pages := map[int]string{
		404: "custom 404",
	}
	handler := New(pages)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected original body, got %q", rec.Body.String())
	}
}

func TestErrorPageNil(t *testing.T) {
	handler := New(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("original"))
	}))

	req := httptest.NewRequest("GET", "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec.Body.String() != "original" {
		t.Fatalf("expected original body, got %q", rec.Body.String())
	}
}
