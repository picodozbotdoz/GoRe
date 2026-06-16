package split

import (
	"crypto/md5"
	"net/http"
	"sort"
	"strings"

	"github.com/user/gore/internal/config"
)

type splitMapping struct {
	source string
	target string
	bands  []band
}

type band struct {
	threshold uint64
	value     string
}

type splitClient struct {
	mappings []splitMapping
}

func New(configs []config.SplitConfig) func(http.Handler) http.Handler {
	var sc splitClient
	for _, cfg := range configs {
		sm := splitMapping{source: cfg.Source, target: cfg.Target}
		var pairs []struct {
			pct   float64
			value string
		}
		total := 0.0
		for _, r := range cfg.Rules {
			pct := r.Percent
			if total+pct > 100 {
				pct = 100 - total
			}
			if pct <= 0 {
				continue
			}
			pairs = append(pairs, struct {
				pct   float64
				value string
			}{pct: pct, value: r.Value})
			total += pct
		}
		if total < 100 {
			pairs = append(pairs, struct {
				pct   float64
				value string
			}{pct: 100 - total, value: cfg.Default})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].value < pairs[j].value })

		var cumulative uint64
		for _, p := range pairs {
			cumulative += uint64(p.pct * 100)
			sm.bands = append(sm.bands, band{threshold: cumulative, value: p.value})
		}
		if len(sm.bands) > 0 {
			sm.bands[len(sm.bands)-1].threshold = 10000
		}
		sc.mappings = append(sc.mappings, sm)
	}

	if len(sc.mappings) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, m := range sc.mappings {
				val := resolveSource(r, m.source)
				h := md5.Sum([]byte(val))
				hash := uint64(h[0])<<8 | uint64(h[1])
				hash = hash % 10000

				for _, b := range m.bands {
					if hash < b.threshold {
						r.Header.Set(m.target, b.value)
						break
					}
				}
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
