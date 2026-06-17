package defaulttype

import (
	"net/http"
	"strings"
)

func New(defaultType string) func(http.Handler) http.Handler {
	if defaultType == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := &responseWriter{ResponseWriter: w, defaultType: defaultType}
			next.ServeHTTP(ww, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	defaultType string
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if ct := w.Header().Get("Content-Type"); ct == "" {
			if idx := strings.Index(w.defaultType, "/"); idx != -1 {
				w.Header().Set("Content-Type", w.defaultType)
			}
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
