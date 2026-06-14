package headers

import "net/http"

type Handler struct {
	Add    map[string]string
	Remove []string
}

func New(add map[string]string, remove []string) *Handler {
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
	for name, value := range rw.handler.Add {
		rw.Header().Set(name, value)
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
