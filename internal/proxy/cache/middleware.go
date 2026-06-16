package cache

import (
	"bytes"
	"net/http"
	"strings"
)

type responseCapture struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	rc.body.Write(b)
	return len(b), nil
}

func (rc *responseCapture) Unwrap() http.ResponseWriter {
	return rc.ResponseWriter
}

type Config struct {
	Enabled bool
	TTL     int
	MaxSize int
}

func Middleware(cfg *Config, cache *Cache) func(http.Handler) http.Handler {
	if !cfg.Enabled || cache == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("Cache-Control") == "no-cache" {
				next.ServeHTTP(w, r)
				return
			}

			key := cache.Key(r)
			if entry, ok := cache.Get(key); ok {
				for k, vv := range entry.Header {
					for _, v := range vv {
						w.Header().Add(k, v)
					}
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(entry.Status)
				if r.Method != http.MethodHead {
					w.Write(entry.Body)
				}
				return
			}

			rc := &responseCapture{ResponseWriter: w, status: 200}
			next.ServeHTTP(rc, r)

			ct := rc.Header().Get("Content-Type")
			if rc.status == 200 && !isUncacheable(ct) {
				entry := &Entry{
					Status: rc.status,
					Header: rc.Header().Clone(),
					Body:   rc.body.Bytes(),
				}
				delete(entry.Header, "X-Cache")
				cache.Set(key, entry)
			}

			w.Header().Set("X-Cache", "MISS")
		})
	}
}

func isUncacheable(ct string) bool {
	uncacheable := []string{"application/octet-stream", "multipart/"}
	for _, prefix := range uncacheable {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}
