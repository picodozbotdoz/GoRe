package mergeslashes

import (
	"net/http"
	"strings"
)

func New(enabled bool) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for strings.Contains(r.URL.Path, "//") {
				r.URL.Path = strings.ReplaceAll(r.URL.Path, "//", "/")
			}
			next.ServeHTTP(w, r)
		})
	}
}
