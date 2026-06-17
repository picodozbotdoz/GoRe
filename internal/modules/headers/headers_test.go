package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/gore/internal/config"
)

func TestHeadersAdd(t *testing.T) {
	handler := New([]config.HeaderEntry{{Name: "X-Frame-Options", Value: "DENY"}}, nil)
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

func TestHeadersAddAlwaysFalseSkipsErrorCodes(t *testing.T) {
	handler := New([]config.HeaderEntry{{Name: "X-Frame-Options", Value: "DENY", Always: false}}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "" {
		t.Errorf("expected no X-Frame-Options on 404, got %q", w.Header().Get("X-Frame-Options"))
	}
}

func TestHeadersAddAlwaysTrueAppliesToErrorCodes(t *testing.T) {
	handler := New([]config.HeaderEntry{{Name: "X-Frame-Options", Value: "DENY", Always: true}}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options on 404 with always=true, got %q", w.Header().Get("X-Frame-Options"))
	}
}

func TestHeadersAddAlwaysTrueAppliesTo500(t *testing.T) {
	handler := New([]config.HeaderEntry{{Name: "X-Custom", Value: "test", Always: true}}, nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(next).ServeHTTP(w, req)

	if w.Header().Get("X-Custom") != "test" {
		t.Errorf("expected X-Custom on 500 with always=true, got %q", w.Header().Get("X-Custom"))
	}
}

func TestHeadersAddDefaultSkips4xx(t *testing.T) {
	handler := New([]config.HeaderEntry{{Name: "X-Frame-Options", Value: "DENY"}}, nil)

	codes := []int{http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError}
	for _, code := range codes {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			w.Write([]byte("error"))
		})

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(next).ServeHTTP(w, req)

		if w.Header().Get("X-Frame-Options") != "" {
			t.Errorf("expected no X-Frame-Options on %d, got %q", code, w.Header().Get("X-Frame-Options"))
		}
	}
}
