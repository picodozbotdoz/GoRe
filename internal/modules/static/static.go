package static

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	Root      string
	Autoindex bool
}

func New(root string, autoindex bool) *Handler {
	return &Handler{Root: root, Autoindex: autoindex}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "." {
		path = "/"
	}

	// If path is a directory, look for index.html
	if strings.HasSuffix(path, "/") {
		indexPath := filepath.Join(h.Root, path, "index.html")
		if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
			h.serveFile(w, r, indexPath)
			return
		}
	}

	fullPath := filepath.Join(h.Root, path)

	if !strings.HasPrefix(fullPath, filepath.Clean(h.Root)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if info.IsDir() {
		if !h.Autoindex {
			http.NotFound(w, r)
			return
		}
		h.serveDirectory(w, r, fullPath)
		return
	}

	h.serveFile(w, r, fullPath)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func (h *Handler) serveDirectory(w http.ResponseWriter, r *http.Request, dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<html><body><h1>Index of " + r.URL.Path + "</h1><ul>"))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		w.Write([]byte("<li><a href=\"" + name + "\">" + name + "</a></li>"))
	}
	w.Write([]byte("</ul></body></html>"))
}
