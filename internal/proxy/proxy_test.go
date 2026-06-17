package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
		s := balancer.Next(nil)
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
		s := balancer.Next(nil)
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
	upstream := NewUpstream("test", servers, "round-robin", nil, nil, true, 0, 0, "", "", 0, 0, nil, nil, nil, false, nil, "", "", "", nil, nil, nil, nil, false, 0)

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

func TestProxyRedirect(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://old.example.com/new")
		w.WriteHeader(http.StatusFound)
	}))
	defer backend.Close()

	servers := []*Server{{Addr: backend.Listener.Addr().String(), Weight: 1}}
	boolTrue := true
	upstream := NewUpstream("test", servers, "round-robin", nil, nil, true, 0, 0, "old.example.com new.example.com", "", 0, 0, &boolTrue, &boolTrue, nil, false, nil, "", "", "", nil, nil, nil, nil, false, 0)

	req := httptest.NewRequest("GET", "/redirect", nil)
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	if w.Code != 302 {
		t.Errorf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "http://new.example.com/new" {
		t.Errorf("Location = %q, want http://new.example.com/new", loc)
	}
}

func TestProxyPassRequestHeadersDisabled(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "" {
			w.Write([]byte("headers forwarded"))
		} else {
			w.Write([]byte("headers stripped"))
		}
	}))
	defer backend.Close()

	servers := []*Server{{Addr: backend.Listener.Addr().String(), Weight: 1}}
	boolFalse := false
	upstream := NewUpstream("test", servers, "round-robin", nil, map[string]string{"X-Custom": "value"}, true, 0, 0, "", "", 0, 0, &boolFalse, nil, nil, false, nil, "", "", "", nil, nil, nil, nil, false, 0)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	if w.Body.String() != "headers stripped" {
		t.Errorf("body = %q, want headers stripped", w.Body.String())
	}
}

func TestProxyPassRequestBodyDisabled(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			buf := make([]byte, 1024)
			n, _ := r.Body.Read(buf)
			if n > 0 {
				w.Write([]byte("body present"))
			} else {
				w.Write([]byte("body absent"))
			}
		} else {
			w.Write([]byte("body absent"))
		}
	}))
	defer backend.Close()

	servers := []*Server{{Addr: backend.Listener.Addr().String(), Weight: 1}}
	boolFalse := false
	upstream := NewUpstream("test", servers, "round-robin", nil, nil, true, 0, 0, "", "", 0, 0, nil, &boolFalse, nil, false, nil, "", "", "", nil, nil, nil, nil, false, 0)

	req := httptest.NewRequest("POST", "/", strings.NewReader("test body"))
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	if w.Body.String() != "body absent" {
		t.Errorf("body = %q, want body absent", w.Body.String())
	}
}

func TestNextUpstreamFlags(t *testing.T) {
	u := &Upstream{NextUpstream: "error timeout"}
	flags := u.parseNextUpstreamFlags()
	if !flags["error"] {
		t.Error("expected error flag")
	}
	if !flags["timeout"] {
		t.Error("expected timeout flag")
	}
	if flags["invalid_header"] {
		t.Error("unexpected invalid_header flag")
	}
}

func TestLeastConnBalancer(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1, ActiveConns: 5},
		{Addr: "127.0.0.1:8081", Weight: 1, ActiveConns: 1},
	}
	balancer := NewLeastConn(servers)
	s := balancer.Next(nil)
	if s.Addr != "127.0.0.1:8081" {
		t.Errorf("expected least conn server 8081, got %v", s.Addr)
	}
	if s.ActiveConns != 2 {
		t.Errorf("expected ActiveConns=2, got %d", s.ActiveConns)
	}
}

func TestLeastConnSkipsUnhealthy(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1, ActiveConns: 10},
		{Addr: "127.0.0.1:8081", Weight: 1, ActiveConns: 1},
	}
	balancer := NewLeastConn(servers)
	atomic.StoreInt32(&servers[1].Healthy, 0)
	s := balancer.Next(nil)
	if s.Addr != "127.0.0.1:8080" {
		t.Errorf("expected 8080 when 8081 unhealthy, got %v", s.Addr)
	}
}

func TestLeastConnSkipsDown(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1, ActiveConns: 10, Down: true},
		{Addr: "127.0.0.1:8081", Weight: 1, ActiveConns: 5},
	}
	balancer := NewLeastConn(servers)
	s := balancer.Next(nil)
	if s.Addr != "127.0.0.1:8081" {
		t.Errorf("expected 8081 when 8080 down, got %v", s.Addr)
	}
}

func TestIPHashBalancer(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewIPHash(servers)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	s1 := balancer.Next(req)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.10:12346"
	s2 := balancer.Next(req2)

	if s1.Addr != s2.Addr {
		t.Errorf("same IP should get same server: %v vs %v", s1.Addr, s2.Addr)
	}

	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "10.0.0.1:9999"
	s3 := balancer.Next(req3)

	if s3.Addr != "127.0.0.1:8080" && s3.Addr != "127.0.0.1:8081" {
		t.Errorf("unexpected server %v", s3.Addr)
	}
}

func TestIPHashUsesXForwardedFor(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewIPHash(servers)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.50")
	s1 := balancer.Next(req)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "127.0.0.1:5678"
	req2.Header.Set("X-Forwarded-For", "192.168.1.50")
	s2 := balancer.Next(req2)

	if s1.Addr != s2.Addr {
		t.Errorf("same X-Forwarded-For should get same server: %v vs %v", s1.Addr, s2.Addr)
	}
}

func TestConsistentHashBalancer(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewConsistentHash(servers)

	req := httptest.NewRequest("GET", "/foo", nil)
	s1 := balancer.Next(req)

	req2 := httptest.NewRequest("GET", "/foo", nil)
	s2 := balancer.Next(req2)

	if s1.Addr != s2.Addr {
		t.Errorf("same path should get same server: %v vs %v", s1.Addr, s2.Addr)
	}
}

func TestConsistentHashDifferentKeys(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewConsistentHash(servers)

	req1 := httptest.NewRequest("GET", "/alpha", nil)
	req2 := httptest.NewRequest("GET", "/beta", nil)
	s1 := balancer.Next(req1)
	s2 := balancer.Next(req2)

	if s1.Addr == "" || s2.Addr == "" {
		t.Fatal("expected valid servers")
	}
}

func TestBackupServers(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1, Backup: false},
		{Addr: "127.0.0.1:8081", Weight: 1, Backup: true},
	}
	balancer := NewRoundRobin(servers)

	s := balancer.Next(nil)
	if s.Addr != "127.0.0.1:8080" {
		t.Errorf("expected primary server, got %v", s.Addr)
	}

	atomic.StoreInt32(&servers[0].Healthy, 0)
	s = balancer.Next(nil)
	if s.Addr != "127.0.0.1:8081" {
		t.Errorf("expected backup server when primary unhealthy, got %v", s.Addr)
	}
}

func TestDownServersSkipped(t *testing.T) {
	servers := []*Server{
		{Addr: "127.0.0.1:8080", Weight: 1, Down: true},
		{Addr: "127.0.0.1:8081", Weight: 1},
	}
	balancer := NewRoundRobin(servers)

	s := balancer.Next(nil)
	if s.Addr != "127.0.0.1:8081" {
		t.Errorf("expected non-down server, got %v", s.Addr)
	}
}

func TestResolverConfig(t *testing.T) {
	servers := []*Server{{Addr: "127.0.0.1:8080", Weight: 1}}
	tc := &TimeoutConfig{
		Connect:  5,
		Read:     30,
		Send:     30,
		Idle:     60,
		Resolver: "8.8.8.8",
	}
	_ = NewUpstream("test", servers, "round-robin", tc, nil, true, 0, 0, "", "", 0, 0, nil, nil, nil, false, nil, "", "", "", nil, nil, nil, nil, false, 0)
}
