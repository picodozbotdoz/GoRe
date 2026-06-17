package gzipstatic

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	Root string
	next http.Handler
}

func New(root string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return &Handler{Root: root, next: next}
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		h.next.ServeHTTP(w, r)
		return
	}

	path := filepath.Clean(r.URL.Path)
	if path == "." {
		path = "/"
	}

	fullPath := filepath.Join(h.Root, path)
	gzPath := fullPath + ".gz"

	info, err := os.Stat(gzPath)
	if err != nil || info.IsDir() {
		h.next.ServeHTTP(w, r)
		return
	}

	file, err := os.Open(gzPath)
	if err != nil {
		h.next.ServeHTTP(w, r)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", detectContentType(fullPath))
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
