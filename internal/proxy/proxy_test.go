package proxy

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRoundRobinBalancer(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewRoundRobin(servers)
	seen := make(map[string]int)
	for i := 0; i < 100; i++ {
		s := balancer.Next()
		seen[s.Addr]++
	}
	if seen["127.0.0.1:8080"] != 50 || seen["127.0.0.1:8081"] != 50 {
		t.Errorf("distribution = %v, want equal", seen)
	}
}

func TestRoundRobinSkipsUnhealthy(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewRoundRobin(servers)
	atomic.StoreInt32(&servers[1].Healthy, 0)
	for i := 0; i < 10; i++ {
		s := balancer.Next()
		if s.Addr != "127.0.0.1:8080" {
			t.Errorf("Next() = %v, want 127.0.0.1:8080", s.Addr)
		}
	}
}

func TestUpstreamServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend"))
	}))
	defer backend.Close()

	servers := []*Server{{Addr: backend.Listener.Addr().String(), Weight: 1}}
	upstream := NewUpstream("test", servers, "round-robin")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "backend" {
		t.Errorf("body = %q, want backend", w.Body.String())
	}
}
