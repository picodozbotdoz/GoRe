package errorpage

import (
	"bytes"
	"fmt"
	"net/http"
)

func New(pages map[int]string) func(http.Handler) http.Handler {
	if len(pages) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			if body, ok := pages[sw.status]; ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Del("Content-Length")
				w.WriteHeader(sw.status)
				fmt.Fprint(w, body)
			} else if sw.buf.Len() > 0 {
				w.WriteHeader(sw.status)
				w.Write(sw.buf.Bytes())
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
	buf     bytes.Buffer
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.written = true
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.written {
		sw.status = 200
		sw.written = true
	}
	return sw.buf.Write(b)
}

func (sw *statusWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}
