package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHtrieExactMatch(t *testing.T) {
	trie := New()
	trie.Insert("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("home")) }))
	trie.Insert("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("api")) }))

	_, ok := trie.Search("/")
	if !ok {
		t.Error("Search(/) should return true")
	}
	_, ok = trie.Search("/api")
	if !ok {
		t.Error("Search(/api) should return true")
	}
}

func TestHtriePrefixMatch(t *testing.T) {
	trie := New()
	trie.Insert("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, ok := trie.Search("/static/file.txt")
	if !ok {
		t.Error("Search(/static/file.txt) should match prefix /static/")
	}
}

func TestHtrieNoMatch(t *testing.T) {
	trie := New()
	trie.Insert("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	_, ok := trie.Search("/other")
	if ok {
		t.Error("Search(/other) should return false")
	}
}

func TestRouterServeHTTP(t *testing.T) {
	router := NewRouter()
	router.AddRoute("/home", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("home")) }))
	router.AddRoute("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("api")) }))

	tests := []struct {
		path   string
		want   string
		status int
	}{
		{"/home", "home", 200},
		{"/api", "api", 200},
		{"/notfound", "", 404},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, tt.status)
		}
		if tt.want != "" && w.Body.String() != tt.want {
			t.Errorf("GET %s: body = %q, want %q", tt.path, w.Body.String(), tt.want)
		}
	}
}
