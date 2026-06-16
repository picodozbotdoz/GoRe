package brotli

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
)

func New(level int, types []string) func(http.Handler) http.Handler {
	if level < 0 || level > 11 {
		level = 6
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
				next.ServeHTTP(w, r)
				return
			}

			cw := &captureWriter{ResponseWriter: w, buf: &bytes.Buffer{}}
			next.ServeHTTP(cw, r)

			ct := cw.Header().Get("Content-Type")
			if len(types) > 0 {
				matched := false
				for _, t := range types {
					if strings.HasPrefix(ct, t) {
						matched = true
						break
					}
				}
				if !matched {
					copyHeaders(w, cw)
					w.WriteHeader(cw.status)
					w.Write(cw.buf.Bytes())
					return
				}
			}

			var compressed bytes.Buffer
			writer := brotli.NewWriterLevel(&compressed, level)
			writer.Write(cw.buf.Bytes())
			writer.Close()

			copyHeaders(w, cw)
			w.Header().Set("Content-Encoding", "br")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", compressed.Len()))
			w.Header().Del("Content-Length")
			w.WriteHeader(cw.status)
			w.Write(compressed.Bytes())
		})
	}
}

func copyHeaders(dst, src http.ResponseWriter) {
	for k, vv := range src.Header() {
		if k == "Content-Encoding" || k == "Content-Length" {
			continue
		}
		for _, v := range vv {
			dst.Header().Add(k, v)
		}
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
	cw.buf.Write(b)
	return len(b), nil
}

func (cw *captureWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}
