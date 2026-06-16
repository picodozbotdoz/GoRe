package brotli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestBrotliCompression(t *testing.T) {
	handler := New(6, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "br")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "br" {
		t.Errorf("Content-Encoding = %q, want br", rec.Header().Get("Content-Encoding"))
	}

	reader := brotli.NewReader(rec.Body)
	body, _ := io.ReadAll(reader)
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}
}

func TestBrotliNoCompression(t *testing.T) {
	handler := New(6, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "br" {
		t.Error("should not compress when client doesn't accept br")
	}
}

func TestBrotliTypeFilter(t *testing.T) {
	handler := New(6, []string{"text/"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"key":"value"}`))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "br")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "br" {
		t.Error("should not compress application/json when filter is text/")
	}
}

func TestBrotliTypeFilterMatch(t *testing.T) {
	handler := New(6, []string{"text/"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>hello</html>"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "br")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "br" {
		t.Errorf("Content-Encoding = %q, want br", rec.Header().Get("Content-Encoding"))
	}
}
