package mirror

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMirrorSendsToBackend(t *testing.T) {
	var received []string
	var mu sync.Mutex

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received = append(received, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer backend.Close()

	handler := New(backend.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || received[0] != "/test" {
		t.Errorf("mirror received %v, want [/test]", received)
	}
}

func TestMirrorEmptyURL(t *testing.T) {
	called := false
	handler := New("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called when mirror URL is empty")
	}
}

func TestMirrorSetsHeader(t *testing.T) {
	var gotHeader bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Mirror-Request") == "true"
		w.WriteHeader(200)
	}))
	defer backend.Close()

	handler := New(backend.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	time.Sleep(100 * time.Millisecond)
	if !gotHeader {
		t.Error("X-Mirror-Request header not set")
	}
}

func TestMirrorForwardsBody(t *testing.T) {
	var receivedBody string
	var mu sync.Mutex

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		receivedBody = string(body)
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer backend.Close()

	handler := New(backend.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("POST", "/test", strings.NewReader("hello world"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if receivedBody != "hello world" {
		t.Errorf("mirror body = %q, want %q", receivedBody, "hello world")
	}
}

func TestMirrorForwardsBodyNil(t *testing.T) {
	var receivedBody string
	var mu sync.Mutex

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		receivedBody = string(body)
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer backend.Close()

	handler := New(backend.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if receivedBody != "" {
		t.Errorf("mirror body = %q, want empty", receivedBody)
	}
}
