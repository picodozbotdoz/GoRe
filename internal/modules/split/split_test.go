package split

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/gore/internal/config"
)

func TestSplitFiftyFifty(t *testing.T) {
	counts := map[string]int{}
	handler := New([]config.SplitConfig{
		{
			Source: "$remote_addr",
			Target: "X-Variant",
			Rules: []config.SplitRule{
				{Percent: 50, Value: "A"},
				{Percent: 50, Value: "B"},
			},
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Variant", r.Header.Get("X-Variant"))
		w.WriteHeader(200)
	}))

	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = fmt.Sprintf("10.0.%d.%d:1234", i/256, i%256)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		counts[rec.Header().Get("X-Variant")]++
	}

	if counts["A"] == 0 || counts["B"] == 0 {
		t.Errorf("distribution A=%d B=%d, both should be > 0", counts["A"], counts["B"])
	}
	t.Logf("distribution: A=%d B=%d", counts["A"], counts["B"])
}

func TestSplitDefault(t *testing.T) {
	handler := New([]config.SplitConfig{
		{
			Source: "$remote_addr",
			Target: "X-Variant",
			Rules:  []config.SplitRule{},
			Default: "default",
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Variant", r.Header.Get("X-Variant"))
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Variant") != "default" {
		t.Errorf("X-Variant = %q, want default", rec.Header().Get("X-Variant"))
	}
}

func TestSplitEmpty(t *testing.T) {
	called := false
	handler := New(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called")
	}
}
