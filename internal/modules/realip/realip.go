package realip

import (
	"net"
	"net/http"
	"strings"
)

var defaultHeaders = []string{"X-Forwarded-For", "X-Real-IP", "X-Client-IP"}

func New(from string) func(http.Handler) http.Handler {
	headers := defaultHeaders
	if from != "" {
		headers = strings.Split(from, ",")
		for i := range headers {
			headers[i] = strings.TrimSpace(headers[i])
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, h := range headers {
				if val := r.Header.Get(h); val != "" {
					ip := extractFirstIP(val)
					if ip != "" {
						host, _, err := net.SplitHostPort(r.RemoteAddr)
						if err == nil {
							r.Header.Set("X-Real-IP", ip)
							r.RemoteAddr = ip + ":0"
							_ = host
						}
						break
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractFirstIP(header string) string {
	if idx := strings.Index(header, ","); idx != -1 {
		header = header[:idx]
	}
	header = strings.TrimSpace(header)
	host, _, err := net.SplitHostPort(header)
	if err != nil {
		host = header
	}
	if ip := net.ParseIP(host); ip != nil {
		return host
	}
	return ""
}
