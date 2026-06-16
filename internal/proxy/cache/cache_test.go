package cache

import (
	"net/http"
	"testing"
	"time"
)

func TestCacheGetSet(t *testing.T) {
	c := New(10, 60*time.Second)

	entry := &Entry{Status: 200, Header: http.Header{"X-Test": {"value"}}, Body: []byte("body")}
	c.Set("GET /test", entry)

	got, ok := c.Get("GET /test")
	if !ok {
		t.Fatal("cache miss")
	}
	if got.Status != 200 {
		t.Errorf("status = %d, want 200", got.Status)
	}
	if string(got.Body) != "body" {
		t.Errorf("body = %q, want %q", string(got.Body), "body")
	}
}

func TestCacheMiss(t *testing.T) {
	c := New(10, 60*time.Second)

	_, ok := c.Get("GET /missing")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestCacheTTL(t *testing.T) {
	c := New(10, 1*time.Millisecond)

	entry := &Entry{Status: 200, Body: []byte("body")}
	c.Set("GET /test", entry)

	time.Sleep(10 * time.Millisecond)

	_, ok := c.Get("GET /test")
	if ok {
		t.Error("expected cache miss after TTL")
	}
}

func TestCacheEviction(t *testing.T) {
	c := New(2, 60*time.Second)

	c.Set("GET /a", &Entry{Status: 200, Body: []byte("a")})
	c.Set("GET /b", &Entry{Status: 200, Body: []byte("b")})
	c.Set("GET /c", &Entry{Status: 200, Body: []byte("c")})

	if len(c.entries) > 2 {
		t.Errorf("entries = %d, want <= 2", len(c.entries))
	}
}

func TestCachePurge(t *testing.T) {
	c := New(10, 60*time.Second)
	c.Set("GET /a", &Entry{Status: 200, Body: []byte("a")})
	c.Set("GET /b", &Entry{Status: 200, Body: []byte("b")})

	c.Purge()

	if len(c.entries) != 0 {
		t.Errorf("entries = %d, want 0 after purge", len(c.entries))
	}
}

func TestCacheStats(t *testing.T) {
	c := New(10, 60*time.Second)
	c.Set("GET /a", &Entry{Status: 200, Body: []byte("a")})
	c.Get("GET /a")
	c.Get("GET /missing")

	hits, misses := c.Stats()
	if hits != 1 {
		t.Errorf("hits = %d, want 1", hits)
	}
	if misses != 1 {
		t.Errorf("misses = %d, want 1", misses)
	}
}
