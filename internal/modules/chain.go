package modules

import (
	"net/http"

	"github.com/user/gore/internal/config"
	"github.com/user/gore/internal/log"
	"github.com/user/gore/internal/modules/access"
	"github.com/user/gore/internal/modules/basicauth"
	"github.com/user/gore/internal/modules/bodylimit"
	"github.com/user/gore/internal/modules/brotli"
	"github.com/user/gore/internal/modules/defaulttype"
	"github.com/user/gore/internal/modules/errorpage"
	"github.com/user/gore/internal/modules/gunzip"
	"github.com/user/gore/internal/modules/gzip"
	"github.com/user/gore/internal/modules/headers"
	"github.com/user/gore/internal/modules/limitconn"
	"github.com/user/gore/internal/modules/mapmodule"
	"github.com/user/gore/internal/modules/mergeslashes"
	"github.com/user/gore/internal/modules/ratelimit"
	"github.com/user/gore/internal/modules/realip"
	"github.com/user/gore/internal/modules/split"
)

func BuildChain(cfg *config.ModulesConfig, next http.Handler) http.Handler {
	handler := next

	if cfg.Gzip != nil && cfg.Gzip.Enabled {
		handler = gzip.New(cfg.Gzip.Level, cfg.Gzip.Types).ServeHTTP(handler)
	}

	if cfg.Brotli != nil && cfg.Brotli.Enabled {
		handler = brotli.New(cfg.Brotli.Level, cfg.Brotli.Types)(handler)
	}

	if cfg.Gunzip != nil && *cfg.Gunzip {
		handler = gunzip.New()(handler)
	}

	if cfg.Headers != nil {
		if cfg.Headers.Expires != "" {
			handler = headers.NewExpiresHandler(cfg.Headers.Expires)(handler)
		}
		handler = headers.New(cfg.Headers.Add, cfg.Headers.Remove).ServeHTTP(handler)
	}

	if cfg.RateLimit != nil {
		handler = ratelimit.New(cfg.RateLimit.Rate, cfg.RateLimit.Burst, cfg.RateLimit.Status, cfg.RateLimit.LogLevel).ServeHTTP(handler)
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

	if cfg.LimitConn != nil && cfg.LimitConn.Connections > 0 {
		handler = limitconn.New(cfg.LimitConn.Connections, cfg.LimitConn.LogLevel).ServeHTTP(handler)
	}

	if cfg.RealIP != nil {
		handler = realip.New(cfg.RealIP.From, cfg.RealIP.Recursive)(handler)
	}

	if len(cfg.Map) > 0 {
		handler = mapmodule.New(cfg.Map)(handler)
	}

	if len(cfg.SplitClients) > 0 {
		handler = split.New(cfg.SplitClients)(handler)
	}

	if cfg.BasicAuth != nil && len(cfg.BasicAuth.Users) > 0 {
		handler = basicauth.New(cfg.BasicAuth.Realm, cfg.BasicAuth.Users)(handler)
	}

	if cfg.ErrorPage != nil {
		handler = errorpage.New(cfg.ErrorPage.Pages)(handler)
	}

	if cfg.DefaultType != "" {
		handler = defaulttype.New(cfg.DefaultType)(handler)
	}

	if cfg.MergeSlashes != nil && *cfg.MergeSlashes {
		handler = mergeslashes.New(true)(handler)
	}

	if cfg.ServerTokens != nil && !*cfg.ServerTokens {
		serverTokensHandler := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverTokensHandler.ServeHTTP(w, r)
			w.Header().Del("Server")
		})
	}

	handler = log.AccessMiddleware(
		cfg.AccessLog != nil && cfg.AccessLog.Enabled,
		cfg.AccessLog.GetOutput(),
		cfg.AccessLog.GetFormat(),
	)(handler)

	return handler
}
