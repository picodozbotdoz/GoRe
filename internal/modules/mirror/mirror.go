package mirror

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func New(mirrorURL string) func(http.Handler) http.Handler {
	if mirrorURL == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	mirrorURL = strings.TrimRight(mirrorURL, "/")
	client := &http.Client{Timeout: 5 * time.Second}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			mirrorReq, err := http.NewRequestWithContext(r.Context(), r.Method, mirrorURL+r.URL.Path, nil)
			if err != nil {
				return
			}
			for k, vv := range r.Header {
				for _, v := range vv {
					mirrorReq.Header.Add(k, v)
				}
			}
			mirrorReq.Header.Set("X-Mirror-Request", "true")

			resp, err := client.Do(mirrorReq)
			if err != nil {
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		})
	}
}
