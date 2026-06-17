package subfilter

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

func New(replacements map[string]string, once bool, types []string) func(http.Handler) http.Handler {
	if len(replacements) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := &captureWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
			next.ServeHTTP(cw, r)

			if len(types) > 0 {
				ct := cw.Header().Get("Content-Type")
				if ct != "" {
					matched := false
					for _, t := range types {
						if strings.Contains(ct, t) {
							matched = true
							break
						}
					}
					if !matched {
						w.Header().Set("Content-Length", fmt.Sprintf("%d", cw.buf.Len()))
						w.WriteHeader(cw.status)
						w.Write(cw.buf.Bytes())
						return
					}
				}
			}

			body := cw.buf.Bytes()
			for old, new := range replacements {
				if once {
					body = bytes.Replace(body, []byte(old), []byte(new), 1)
				} else {
					body = bytes.ReplaceAll(body, []byte(old), []byte(new))
				}
			}

			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			w.WriteHeader(cw.status)
			w.Write(body)
		})
	}
}

type captureWriter struct {
	http.ResponseWriter
	status int
	buf    *bytes.Buffer
}

func (cw *captureWriter) WriteHeader(code int) {
	cw.status = code
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	if cw.status == 0 {
		cw.status = 200
	}
	cw.buf.Write(b)
	return len(b), nil
}

func (cw *captureWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}
