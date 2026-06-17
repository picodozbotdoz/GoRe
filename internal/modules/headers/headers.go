package headers

import (
	"net/http"

	"github.com/user/gore/internal/config"
)

type Handler struct {
	Add    []config.HeaderEntry
	Remove []string
}

func New(add []config.HeaderEntry, remove []string) *Handler {
	return &Handler{Add: add, Remove: remove}
}

func (h *Handler) ServeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, handler: h}
		next.ServeHTTP(rw, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	handler *Handler
	written bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.written {
		return
	}
	rw.written = true
	for _, name := range rw.handler.Remove {
		rw.Header().Del(name)
	}
	for _, entry := range rw.handler.Add {
		if !entry.Always && code >= 400 {
			continue
		}
		rw.Header().Set(entry.Name, entry.Value)
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
