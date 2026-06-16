package authrequest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthRequestAllowed(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Auth-Request-Set", "X-User=admin")
		w.WriteHeader(200)
	}))
	defer authServer.Close()

	called := false
	handler := New(authServer.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
		w.Write([]byte("allowed"))
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer token123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("next handler not called on auth success")
	}
	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("X-User") != "admin" {
		t.Errorf("X-User = %q, want %q", rec.Header().Get("X-User"), "admin")
	}
}

func TestAuthRequestDenied(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte("forbidden"))
	}))
	defer authServer.Close()

	called := false
	handler := New(authServer.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called on auth denial")
	}
	if rec.Code != 403 {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "forbidden") {
		t.Errorf("body = %q, want 'forbidden'", rec.Body.String())
	}
}

func TestAuthRequestForwardsHeaders(t *testing.T) {
	var gotURI, gotMethod, gotAuth, gotCookie string
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURI = r.Header.Get("X-Original-URI")
		gotMethod = r.Header.Get("X-Original-Method")
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		w.WriteHeader(200)
	}))
	defer authServer.Close()

	handler := New(authServer.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/admin?foo=bar", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Cookie", "session=abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotURI != "/admin?foo=bar" {
		t.Errorf("X-Original-URI = %q, want /admin?foo=bar", gotURI)
	}
	if gotMethod != "GET" {
		t.Errorf("X-Original-Method = %q, want GET", gotMethod)
	}
	if gotAuth != "Bearer token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer token")
	}
	if gotCookie != "session=abc" {
		t.Errorf("Cookie = %q, want %q", gotCookie, "session=abc")
	}
}

func TestAuthServerUnreachable(t *testing.T) {
	handler := New("http://127.0.0.1:1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 502 {
		t.Errorf("status = %d, want 502 when auth server unreachable", rec.Code)
	}
}
