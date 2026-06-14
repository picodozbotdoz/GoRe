package static

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticServeFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "static-test")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)

	handler := New(tmpDir, false)
	req := httptest.NewRequest("GET", "/test.txt", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("body = %q, want hello", w.Body.String())
	}
}

func TestStaticDirectoryTraversal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "static-test")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(filepath.Join(tmpDir, "secret.txt"), []byte("secret"), 0644)

	handler := New(tmpDir, false)

	// Try various traversal patterns
	patterns := []string{
		"/../../../etc/passwd",
		"/..%2F..%2F..%2Fetc/passwd",
		"/%2e%2e/%2e%2e/%2e%2e/etc/passwd",
	}

	for _, pattern := range patterns {
		req := httptest.NewRequest("GET", pattern, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 404 && w.Code != 403 {
			t.Errorf("GET %s: status = %d, want 403 or 404", pattern, w.Code)
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Errorf("GET %s: leaked file content", pattern)
		}
	}
}

func TestStaticNotFound(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "static-test")
	defer os.RemoveAll(tmpDir)

	handler := New(tmpDir, false)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
