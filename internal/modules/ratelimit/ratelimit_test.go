package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	limiter := New("10/s", 10, 0, "")
	for i := 0; i < 10; i++ {
		if !limiter.Allow("test") {
			t.Errorf("request %d should be allowed", i)
		}
	}
	if limiter.Allow("test") {
		t.Error("11th request should be denied")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	limiter := New("10/s", 10, 0, "")
	for i := 0; i < 10; i++ {
		limiter.Allow("test")
	}
	time.Sleep(200 * time.Millisecond)
	if !limiter.Allow("test") {
		t.Error("should allow after refill")
	}
}

func TestRateLimiterServeHTTP(t *testing.T) {
	limiter := New("1/s", 1, 0, "")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	handler := limiter.ServeHTTP(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("first request: status = %d, want 200", w.Code)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("second request: status = %d, want 429", w.Code)
	}
}

func TestRateLimiterCustomStatus(t *testing.T) {
	limiter := New("1/s", 1, 503, "")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	handler := limiter.ServeHTTP(next)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:5678"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 503 {
		t.Errorf("second request: status = %d, want 503", w.Code)
	}
}
