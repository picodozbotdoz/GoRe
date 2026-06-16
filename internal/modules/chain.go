package modules

import (
	"net/http"

	"github.com/user/gore/internal/config"
	"github.com/user/gore/internal/log"
	"github.com/user/gore/internal/modules/access"
	"github.com/user/gore/internal/modules/bodylimit"
	"github.com/user/gore/internal/modules/gzip"
	"github.com/user/gore/internal/modules/headers"
	"github.com/user/gore/internal/modules/ratelimit"
)

func BuildChain(cfg *config.ModulesConfig, next http.Handler) http.Handler {
	handler := next

	if cfg.Gzip != nil && cfg.Gzip.Enabled {
		handler = gzip.New(cfg.Gzip.Level, cfg.Gzip.Types).ServeHTTP(handler)
	}

	if cfg.Headers != nil {
		handler = headers.New(cfg.Headers.Add, cfg.Headers.Remove).ServeHTTP(handler)
	}

	if cfg.RateLimit != nil {
		handler = ratelimit.New(cfg.RateLimit.Rate, cfg.RateLimit.Burst).ServeHTTP(handler)
	}

	if cfg.Access != nil {
		rules := make([]access.Rule, len(cfg.Access.Rules))
		for i, r := range cfg.Access.Rules {
			if r.Allow != "" {
				network, _ := access.ParseCIDR(r.Allow)
				rules[i].Allow = network
			}
			if r.Deny != "" {
				network, _ := access.ParseCIDR(r.Deny)
				rules[i].Deny = network
			}
		}
		handler = access.New(rules).ServeHTTP(handler)
	}

	if cfg.ClientMaxBodySize != "" {
		maxBytes := config.ParseSize(cfg.ClientMaxBodySize)
		handler = bodylimit.New(maxBytes)(handler)
	}

	handler = log.AccessMiddleware(
		cfg.AccessLog != nil && cfg.AccessLog.Enabled,
		cfg.AccessLog.GetOutput(),
		cfg.AccessLog.GetFormat(),
	)(handler)

	return handler
}
