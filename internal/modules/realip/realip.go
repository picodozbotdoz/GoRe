package realip

import (
	"net"
	"net/http"
	"strings"
)

var defaultHeaders = []string{"X-Forwarded-For", "X-Real-IP", "X-Client-IP"}

func New(from []string, recursive bool) func(http.Handler) http.Handler {
	headers := defaultHeaders
	if len(from) > 0 {
		headers = from
	}

	var trusted []*net.IPNet
	if recursive {
		for _, cidr := range from {
			_, network, err := net.ParseCIDR(cidr)
			if err == nil {
				trusted = append(trusted, network)
			}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, h := range headers {
				if val := r.Header.Get(h); val != "" {
					var ip string
					if recursive && len(trusted) > 0 {
						ip = extractIPRecursive(val, trusted)
					} else {
						ip = extractFirstIP(val)
					}
					if ip != "" {
						r.Header.Set("X-Real-IP", ip)
						r.RemoteAddr = ip + ":0"
					}
					break
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

func extractIPRecursive(header string, trusted []*net.IPNet) string {
	parts := strings.Split(header, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip := parseIP(strings.TrimSpace(parts[i]))
		if ip == nil {
			continue
		}
		isTrusted := false
		for _, cidr := range trusted {
			if cidr.Contains(ip) {
				isTrusted = true
				break
			}
		}
		if !isTrusted {
			return ip.String()
		}
	}
	return ""
}

func parseIP(s string) net.IP {
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		host = s
	}
	return net.ParseIP(host)
}
