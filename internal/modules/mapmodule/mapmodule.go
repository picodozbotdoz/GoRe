package mapmodule

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/user/gore/internal/config"
)

type compiledRule struct {
	re    *regexp.Regexp
	value string
}

type mappedHeader struct {
	source     string
	target     string
	rules      []compiledRule
	defaultVal string
}

func New(configs []config.MapConfig) func(http.Handler) http.Handler {
	var maps []mappedHeader
	for _, cfg := range configs {
		mh := mappedHeader{
			source:     cfg.Source,
			target:     cfg.Target,
			defaultVal: cfg.Default,
		}
		for _, r := range cfg.Rules {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				continue
			}
			mh.rules = append(mh.rules, compiledRule{re: re, value: r.Value})
		}
		if len(mh.rules) > 0 {
			maps = append(maps, mh)
		}
	}

	if len(maps) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, m := range maps {
				val := resolveSource(r, m.source)
				result := m.defaultVal
				for _, rule := range m.rules {
					if rule.re.MatchString(val) {
						result = rule.value
						break
					}
				}
				r.Header.Set(m.target, result)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func resolveSource(r *http.Request, source string) string {
	source = strings.TrimPrefix(source, "$")
	switch {
	case strings.HasPrefix(source, "http_"):
		headerName := strings.TrimPrefix(source, "http_")
		headerName = strings.ReplaceAll(headerName, "_", "-")
		return r.Header.Get(headerName)
	case source == "remote_addr":
		return r.RemoteAddr
	case source == "host":
		return r.Host
	case source == "request_uri":
		return r.URL.RequestURI()
	case source == "method":
		return r.Method
	default:
		return r.Header.Get(source)
	}
}
