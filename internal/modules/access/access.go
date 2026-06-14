package access

import (
	"net"
	"net/http"
	"strings"
)

type Rule struct {
	Allow *net.IPNet
	Deny  *net.IPNet
}

type Handler struct {
	Rules []Rule
}

func New(rules []Rule) *Handler {
	return &Handler{Rules: rules}
}

func (h *Handler) ServeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := parseIP(r.RemoteAddr)

		// Process rules in order, stop at first match
		for _, rule := range h.Rules {
			if rule.Allow != nil && rule.Allow.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
			if rule.Deny != nil && rule.Deny.Contains(ip) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// No rules matched, allow by default
		next.ServeHTTP(w, r)
	})
}

func parseIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.ParseIP(remoteAddr)
	}
	return net.ParseIP(host)
}

func ParseCIDR(cidr string) (*net.IPNet, error) {
	if cidr == "all" {
		_, network, err := net.ParseCIDR("0.0.0.0/0")
		return network, err
	}
	if !strings.Contains(cidr, "/") {
		cidr = cidr + "/32"
	}
	_, network, err := net.ParseCIDR(cidr)
	return network, err
}
