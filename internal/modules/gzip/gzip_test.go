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

func TestGzipMinLength(t *testing.T) {
	handler := New(6, nil)
	handler.MinLength = 100

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("short")) // 5 bytes < 100
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("should not compress short response")
	}
}

func TestGzipMinLengthExceeds(t *testing.T) {
	handler := New(6, nil)
	handler.MinLength = 10

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("this is a longer response"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("should compress response exceeding min_length")
	}
}

func TestGzipVary(t *testing.T) {
	handler := New(6, nil)
	handler.Vary = true

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world this is long enough"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("expected Vary header, got %q", w.Header().Get("Vary"))
	}
}

func TestGzipVaryDisabled(t *testing.T) {
	handler := New(6, nil)
	handler.Vary = false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world this is long enough"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Vary") == "Accept-Encoding" {
		t.Fatal("should not set Vary header when disabled")
	}
}

func TestGzipProxied(t *testing.T) {
	handler := New(6, nil)
	handler.Proxied = false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world this is long enough"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("should not compress proxied request when proxied=false")
	}
}

func TestGzipProxiedEnabled(t *testing.T) {
	handler := New(6, nil)
	handler.Proxied = true

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world this is long enough"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("should compress proxied request when proxied=true")
	}
}

func TestGzipDisable(t *testing.T) {
	handler := New(6, nil)
	handler.Disable = "MSIE"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world this is long enough"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.1)")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("should not compress for disabled User-Agent")
	}
}

func TestGzipDisableNoMatch(t *testing.T) {
	handler := New(6, nil)
	handler.Disable = "MSIE"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world this is long enough"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Chrome 90.0)")
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("should compress for non-matching User-Agent")
	}
}
