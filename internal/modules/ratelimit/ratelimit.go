package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(rate float64, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

type Limiter struct {
	buckets map[string]*tokenBucket
	rate    float64
	burst   int
	mu      sync.RWMutex
}

func New(rate string, burst int) *Limiter {
	var r float64
	if len(rate) >= 2 {
		switch rate[len(rate)-1] {
		case 's':
			r = parseFloat(rate[:len(rate)-1])
		case 'm':
			r = parseFloat(rate[:len(rate)-1]) / 60
		default:
			r = parseFloat(rate)
		}
	}
	if r <= 0 {
		r = 10
	}
	if burst <= 0 {
		burst = int(r)
	}
	return &Limiter{buckets: make(map[string]*tokenBucket), rate: r, burst: burst}
}

func (l *Limiter) getBucket(key string) *tokenBucket {
	l.mu.RLock()
	bucket, ok := l.buckets[key]
	l.mu.RUnlock()
	if ok {
		return bucket
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if bucket, ok = l.buckets[key]; ok {
		return bucket
	}
	bucket = newTokenBucket(l.rate, l.burst)
	l.buckets[key] = bucket
	return bucket
}

func (l *Limiter) Allow(key string) bool {
	return l.getBucket(key).allow()
}

func (l *Limiter) ServeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseFloat(s string) float64 {
	var result float64
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			result = result*10 + float64(ch-'0')
		}
	}
	return result
}
