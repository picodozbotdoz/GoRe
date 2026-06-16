package limitconn

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

type Limiter struct {
	mu       sync.Mutex
	counts   map[string]int
	limit    int
}

func New(limit int) *Limiter {
	if limit <= 0 {
		return nil
	}
	return &Limiter{
		counts: make(map[string]int),
		limit:  limit,
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
