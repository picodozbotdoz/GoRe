package gzipstatic

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGzipStaticServesPrecompressedFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gzipstatic-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	original := []byte("hello world this is a test file with enough content")
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), original, 0644)

	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	gz.Write(original)
	gz.Close()
	os.WriteFile(filepath.Join(tmpDir, "test.txt.gz"), buf.Bytes(), 0644)

	handler := New(tmpDir)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fallback"))
	})

	req := httptest.NewRequest("GET", "/test.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Content-Encoding = %q, want gzip", w.Header().Get("Content-Encoding"))
	}

	gzReader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gzReader.Close()
	body, _ := io.ReadAll(gzReader)
	if string(body) != string(original) {
		t.Errorf("body = %q, want %q", string(body), string(original))
	}
}

func TestGzipStaticFallsBackWithoutAcceptEncoding(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "gzipstatic-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "test.txt.gz"), []byte("gzipped"), 0644)

	handler := New(tmpDir)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fallback"))
	})

	req := httptest.NewRequest("GET", "/test.txt", nil)
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Body.String() != "fallback" {
		t.Errorf("body = %q, want fallback when no Accept-Encoding", w.Body.String())
	}
}

func TestGzipStaticReturns404WhenNoGzFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "gzipstatic-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("plain"), 0644)

	handler := New(tmpDir)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fallback"))
	})

	req := httptest.NewRequest("GET", "/test.txt", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Body.String() != "fallback" {
		t.Errorf("body = %q, want fallback when no .gz file", w.Body.String())
	}
}

func TestGzipStaticDetectsContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/index.html", "text/html"},
		{"/style.css", "text/css"},
		{"/app.js", "application/javascript"},
		{"/data.json", "application/json"},
		{"/feed.xml", "application/xml"},
		{"/readme.txt", "text/plain"},
		{"/image.svg", "image/svg+xml"},
		{"/binary.dat", "application/octet-stream"},
	}

	for _, tt := range tests {
		got := detectContentType(tt.path)
		if got != tt.want {
			t.Errorf("detectContentType(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
