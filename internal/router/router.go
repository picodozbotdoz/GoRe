package router

import "net/http"

type Router struct {
	trie *Htrie
}

func NewRouter() *Router {
	return &Router{trie: New()}
}

func (r *Router) AddRoute(path string, handler http.Handler) {
	r.trie.Insert(path, handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler, ok := r.trie.Search(req.URL.Path)
	if !ok {
		http.NotFound(w, req)
		return
	}
	handler.(http.Handler).ServeHTTP(w, req)
}
