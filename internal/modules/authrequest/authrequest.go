package authrequest

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func New(authURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, authURL, nil)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			subReq.Header.Set("X-Original-URI", r.URL.RequestURI())
			subReq.Header.Set("X-Original-Method", r.Method)
			for _, h := range []string{"Authorization", "Cookie", "X-Forwarded-For", "X-Forwarded-Proto"} {
				if v := r.Header.Get(h); v != "" {
					subReq.Header.Set(h, v)
				}
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(subReq)
			if err != nil {
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(resp.StatusCode)
				io.Copy(w, resp.Body)
				return
			}

			for _, hv := range resp.Header.Values("X-Auth-Request-Set") {
				parts := strings.SplitN(hv, "=", 2)
				if len(parts) == 2 {
					w.Header().Set(parts[0], strings.Trim(parts[1], "\""))
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
