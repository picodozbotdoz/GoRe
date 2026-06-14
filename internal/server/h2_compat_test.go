package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/user/gore/internal/config"
)

func h2TestServer(t *testing.T, cfg *config.Config) (addr string, stop func()) {
	t.Helper()
	srv := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr = ln.Addr().String()

	h2cfg := srv.buildHTTP2Config(&cfg.Listen[0])
	h := h2c.NewHandler(srv.router, h2cfg)
	httpSrv := &http.Server{
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go httpSrv.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	return addr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		httpSrv.Shutdown(ctx)
	}
}

func h2TestClient(addr string) *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, a string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, a)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}
}

func h2TestClientH1(addr string) *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}
}

// --- h2.t equivalent tests ---

func TestH2Get(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 body"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Proto != "HTTP/2.0" {
		t.Errorf("proto = %q, want HTTP/2.0", resp.Proto)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "body" {
		t.Errorf("body = %q, want %q", string(body), "body")
	}
}

func TestH2GetMultiplexed(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 body"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
			if err != nil {
				t.Errorf("GET failed: %v", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Errorf("status = %d, want 200", resp.StatusCode)
			}
			if resp.Proto != "HTTP/2.0" {
				t.Errorf("proto = %q, want HTTP/2.0", resp.Proto)
			}
		}()
	}
	wg.Wait()
}

func TestH2Head(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 body"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Head(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		t.Errorf("HEAD should have no body, got %d bytes", len(body))
	}
}

func TestH2Redirect301(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/redirect", Return: "301 /destination"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/redirect", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 301 {
		t.Errorf("status = %d, want 301", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/destination" {
		t.Errorf("Location = %q, want %q", loc, "/destination")
	}
}

func TestH2Redirect301NoLocation(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/redirect", Return: "301"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/redirect", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 301 {
		t.Errorf("status = %d, want 301", resp.StatusCode)
	}
}

func TestH2Status405(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/notallowed", Return: "405"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/notallowed", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 405 {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestH2Return200WithBody(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 hello-world"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello-world" {
		t.Errorf("body = %q, want %q", string(body), "hello-world")
	}
}

func TestH2MaxConcurrentStreams(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{
			Addr: "127.0.0.1:0",
			HTTP2: &config.HTTP2{
				MaxConcurrentStreams: 1,
			},
		}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 ok"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)

	var active int32
	var maxActive int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cur := atomic.AddInt32(&active, 1)
			for {
				old := atomic.LoadInt32(&maxActive)
				if cur <= old || atomic.CompareAndSwapInt32(&maxActive, old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&active, -1)

			resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
			if err != nil {
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()
}

func TestH2NotFound(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/exists", Return: "200 ok"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/nope", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestH2PostBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(200)
		w.Write(body)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	h2cfg := &http2.Server{MaxConcurrentStreams: 250}
	h := h2c.NewHandler(mux, h2cfg)
	httpSrv := &http.Server{Handler: h}
	go httpSrv.Serve(ln)
	time.Sleep(100 * time.Millisecond)
	defer httpSrv.Shutdown(context.Background())

	client := h2TestClient(addr)
	resp, err := client.Post(
		fmt.Sprintf("http://%s/echo", addr),
		"text/plain",
		strings.NewReader("test-data"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "test-data" {
		t.Errorf("body = %q, want %q", string(body), "test-data")
	}
}

func TestH2ResponseHeaders(t *testing.T) {
	var receivedXFoo string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		receivedXFoo = r.Header.Get("X-Foo")
		w.Header().Set("X-Response", "bar")
		w.Header().Set("X-Echo", r.Header.Get("X-Foo"))
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	h2cfg := &http2.Server{}
	h := h2c.NewHandler(mux, h2cfg)
	httpSrv := &http.Server{Handler: h}
	go httpSrv.Serve(ln)
	time.Sleep(100 * time.Millisecond)
	defer httpSrv.Shutdown(context.Background())

	client := h2TestClient(addr)
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/", addr), nil)
	req.Header.Set("X-Foo", "baz")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Response") != "bar" {
		t.Errorf("X-Response = %q, want %q", resp.Header.Get("X-Response"), "bar")
	}
	if resp.Header.Get("X-Echo") != "baz" {
		t.Errorf("X-Echo = %q, want %q", resp.Header.Get("X-Echo"), "baz")
	}
	if receivedXFoo != "baz" {
		t.Errorf("server received X-Foo = %q, want %q", receivedXFoo, "baz")
	}
}

func TestH2HTTP1Fallback(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 ok"},
			},
		}}},
	}
	srv := New(cfg)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	h2cfg := srv.buildHTTP2Config(&cfg.Listen[0])
	h := h2c.NewHandler(srv.router, h2cfg)
	httpSrv := &http.Server{Handler: h}
	go httpSrv.Serve(ln)
	time.Sleep(100 * time.Millisecond)
	defer httpSrv.Shutdown(context.Background())

	client := h2TestClientH1(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Proto != "HTTP/1.1" {
		t.Errorf("proto = %q, want HTTP/1.1", resp.Proto)
	}
}

func TestH2LargeResponse(t *testing.T) {
	largeBody := strings.Repeat("X", 100000)
	mux := http.NewServeMux()
	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(largeBody)))
		w.Write([]byte(largeBody))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()

	h2cfg := &http2.Server{}
	h := h2c.NewHandler(mux, h2cfg)
	httpSrv := &http.Server{Handler: h}
	go httpSrv.Serve(ln)
	time.Sleep(100 * time.Millisecond)
	defer httpSrv.Shutdown(context.Background())

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/big", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 100000 {
		t.Errorf("body length = %d, want 100000", len(body))
	}
}

func TestH2MultipleLocations(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/a", Return: "200 first"},
				{Path: "/b", Return: "200 second"},
				{Path: "/c", Return: "403 forbidden"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)

	tests := []struct {
		path   string
		status int
		body   string
	}{
		{"/a", 200, "first"},
		{"/b", 200, "second"},
		{"/c", 403, "forbidden"},
		{"/d", 404, ""},
	}

	for _, tt := range tests {
		resp, err := client.Get(fmt.Sprintf("http://%s%s", addr, tt.path))
		if err != nil {
			t.Errorf("GET %s failed: %v", tt.path, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("GET %s: status = %d, want %d", tt.path, resp.StatusCode, tt.status)
		}
		if tt.body != "" && string(body) != tt.body {
			t.Errorf("GET %s: body = %q, want %q", tt.path, string(body), tt.body)
		}
	}
}

func TestH2ReturnQuotedBody(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 'quoted body'"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)
	resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "quoted body" {
		t.Errorf("body = %q, want %q", string(body), "quoted body")
	}
}

func TestH2ConnectionReuse(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{Servers: []config.Server{{
			Locations: []config.Location{
				{Path: "/", Return: "200 ok"},
			},
		}}},
	}
	addr, stop := h2TestServer(t, cfg)
	defer stop()

	client := h2TestClient(addr)

	for i := 0; i < 20; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("request %d: status = %d, want 200", i, resp.StatusCode)
		}
		if resp.Proto != "HTTP/2.0" {
			t.Errorf("request %d: proto = %q, want HTTP/2.0", i, resp.Proto)
		}
	}
}
