package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeadersAdd(t *testing.T) {
	handler := New(map[string]string{"X-Frame-Options": "DENY"}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", w.Header().Get("X-Frame-Options"))
	}
}

func TestHeadersRemove(t *testing.T) {
	handler := New(nil, []string{"Server"})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "nginx/1.0")
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("Server") != "" {
		t.Errorf("Server = %q, want empty", w.Header().Get("Server"))
	}
}
