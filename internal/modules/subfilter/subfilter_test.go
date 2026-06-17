package subfilter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubFilterReplaces(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("old content old"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content new" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content new")
	}
}

func TestSubFilterMultiplePatterns(t *testing.T) {
	handler := New(map[string]string{
		"foo": "bar",
		"baz": "qux",
	}, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("foo baz foo"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "bar qux bar" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "bar qux bar")
	}
}

func TestSubFilterEmpty(t *testing.T) {
	called := false
	handler := New(map[string]string{}, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("unchanged"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called")
	}
	if rec.Body.String() != "unchanged" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "unchanged")
	}
}

func TestSubFilterPreservesHeaders(t *testing.T) {
	handler := New(map[string]string{"a": "b"}, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(201)
		w.Write([]byte("a"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if rec.Header().Get("X-Custom") != "value" {
		t.Errorf("X-Custom = %q, want %q", rec.Header().Get("X-Custom"), "value")
	}
	if rec.Body.String() != "b" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "b")
	}
}

func TestSubFilterOnce(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, true, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("old content old"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content old" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content old")
	}
}

func TestSubFilterOnceDisabled(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("old content old"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content new" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content new")
	}
}

func TestSubFilterTypesMatch(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, []string{"text/html", "text/plain"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("old content"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content")
	}
}

func TestSubFilterTypesNoMatch(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, []string{"text/html"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("old content"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "old content" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "old content")
	}
}

func TestSubFilterTypesEmpty(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, []string{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("old content"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content")
	}
}

func TestSubFilterTypesNil(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("old content"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content")
	}
}

func TestSubFilterTypesSubStringMatch(t *testing.T) {
	handler := New(map[string]string{
		"old": "new",
	}, false, []string{"text/html"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("old content"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "new content" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "new content")
	}
}
