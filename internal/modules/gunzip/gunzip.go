package gunzip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func New() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			cw := &captureWriter{ResponseWriter: w}
			next.ServeHTTP(cw, r)

			if cw.Header().Get("Content-Encoding") != "gzip" {
				return
			}

			reader, err := gzip.NewReader(bytes.NewReader(cw.body))
			if err != nil {
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			defer reader.Close()

			decompressed, err := io.ReadAll(reader)
			if err != nil {
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}

			for k, vv := range cw.Header() {
				if k == "Content-Encoding" || k == "Content-Length" {
					continue
				}
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.Header().Del("Content-Encoding")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(decompressed)))
			w.WriteHeader(cw.status)
			w.Write(decompressed)
		})
	}
}

type captureWriter struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (cw *captureWriter) WriteHeader(code int) {
	cw.status = code
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	cw.body = append(cw.body, b...)
	return len(b), nil
}

func (cw *captureWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}
