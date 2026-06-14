package gzip

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipCompression(t *testing.T) {
	handler := New(6, []string{"text/plain"})

	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, 6)
	gz.Write([]byte("hello world"))
	gz.Close()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(buf.Bytes())
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Content-Encoding = %q, want gzip", w.Header().Get("Content-Encoding"))
	}
}

func TestGzipSkipsUnsupportedType(t *testing.T) {
	handler := New(6, []string{"text/plain"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("binary"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress image/png")
	}
}

func TestGzipSkipsNoAcceptEncoding(t *testing.T) {
	handler := New(6, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if strings.Contains(w.Header().Get("Content-Encoding"), "gzip") {
		t.Error("should not compress without Accept-Encoding")
	}
}
