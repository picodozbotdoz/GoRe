package cache

import (
	"net/http"
	"sync"
	"time"
)

type Entry struct {
	Status    int
	Header    http.Header
	Body      []byte
	CreatedAt time.Time
	TTL       time.Duration
}

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
}

func New(maxSize int, ttl time.Duration) *Cache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	c := &Cache{
		entries: make(map[string]*Entry),
		maxSize: maxSize,
		ttl:     ttl,
	}
	go c.evict()
	return c
}

func (c *Cache) Key(r *http.Request) string {
	return r.Method + " " + r.URL.RequestURI()
}

func (c *Cache) Get(key string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		c.misses++
		return nil, false
	}
	ttl := c.ttl
	if entry.TTL > 0 {
		ttl = entry.TTL
	}
	if time.Since(entry.CreatedAt) > ttl {
		c.misses++
		return nil, false
	}
	c.hits++
	return entry, true
}

func (c *Cache) Set(key string, entry *Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}
	entry.CreatedAt = time.Now()
	c.entries[key] = entry
}

func (c *Cache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*Entry)
}

func (c *Cache) GetStale(key string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	return entry, true
}

func (c *Cache) Stats() (hits, misses int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses
}

func (c *Cache) evict() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.entries {
			if now.Sub(v.CreatedAt) > c.ttl {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for k, v := range c.entries {
		if oldestKey == "" || v.CreatedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.CreatedAt
		}
	}
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}
