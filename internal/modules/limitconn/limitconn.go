package limitconn

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

type Limiter struct {
	mu       sync.Mutex
	counts   map[string]int
	limit    int
	logLevel string
}

func New(limit int, logLevel string) *Limiter {
	if limit <= 0 {
		return nil
	}
	return &Limiter{
		counts:   make(map[string]int),
		limit:    limit,
		logLevel: logLevel,
	}
}

func (l *Limiter) ServeHTTP(next http.Handler) http.Handler {
	if l == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r.RemoteAddr)

		l.mu.Lock()
		if l.counts[ip] >= l.limit {
			l.mu.Unlock()
			if l.logLevel != "" {
				log.Printf("[%s] connection limit exceeded for %s: %s %s", l.logLevel, ip, r.Method, r.URL.Path)
			}
			http.Error(w, "Too Many Connections", http.StatusServiceUnavailable)
			return
		}
		l.counts[ip]++
		l.mu.Unlock()

		defer func() {
			l.mu.Lock()
			l.counts[ip]--
			if l.counts[ip] <= 0 {
				delete(l.counts, ip)
			}
			l.mu.Unlock()
		}()

		next.ServeHTTP(w, r)
	})
}

func extractIP(remoteAddr string) string {
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
