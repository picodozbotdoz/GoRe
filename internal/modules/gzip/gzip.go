package gzip

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

type Handler struct {
	Level int
	Types map[string]bool
	pool  sync.Pool
}

func New(level int, types []string) *Handler {
	h := &Handler{Level: level, Types: make(map[string]bool)}
	for _, t := range types {
		h.Types[t] = true
	}
	h.pool = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, h.Level)
			return w
		},
	}
	return h
}

func (h *Handler) ServeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		var buf bytes.Buffer
		gz := h.pool.Get().(*gzip.Writer)
		gz.Reset(&buf)
		defer h.pool.Put(gz)

		rw := &captureResponseWriter{ResponseWriter: w, buf: &buf, gz: gz}
		next.ServeHTTP(rw, r)

		contentType := w.Header().Get("Content-Type")
		if !h.shouldCompress(contentType) {
			return
		}

		gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.Write(buf.Bytes())
	})
}

func (h *Handler) shouldCompress(contentType string) bool {
	if len(h.Types) == 0 {
		return true
	}
	for t := range h.Types {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}

type captureResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
	gz  *gzip.Writer
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	return w.gz.Write(b)
}

func (w *captureResponseWriter) Flush() {
	w.gz.Flush()
}
