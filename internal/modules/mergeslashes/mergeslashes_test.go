package mergeslashes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMergeSlashes(t *testing.T) {
	var got string
	handler := New(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "//foo///bar//baz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "/foo/bar/baz" {
		t.Fatalf("expected /foo/bar/baz, got %q", got)
	}
}

func TestMergeSlashesDisabled(t *testing.T) {
	var got string
	handler := New(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "//foo", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "//foo" {
		t.Fatalf("expected //foo, got %q", got)
	}
}

func TestMergeSlashesSingleSlash(t *testing.T) {
	var got string
	handler := New(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/foo/bar", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "/foo/bar" {
		t.Fatalf("expected /foo/bar, got %q", got)
	}
}
