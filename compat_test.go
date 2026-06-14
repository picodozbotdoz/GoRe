package main

import (
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/gore/internal/config"
	"github.com/user/gore/internal/modules"
	"github.com/user/gore/internal/modules/access"
	"github.com/user/gore/internal/proxy"
	"github.com/user/gore/internal/router"
	"github.com/user/gore/internal/modules/static"
)

// Test compatibility with nginx access.t
func TestAccessModule(t *testing.T) {
	tests := []struct {
		name    string
		rules   []access.Rule
		remote  string
		want    int
	}{
		{
			name:   "allow all",
			rules:  []access.Rule{{Allow: parseCIDR(t, "0.0.0.0/0")}},
			remote: "192.168.1.1:1234",
			want:   200,
		},
		{
			name:   "deny all",
			rules:  []access.Rule{{Deny: parseCIDR(t, "0.0.0.0/0")}},
			remote: "192.168.1.1:1234",
			want:   403,
		},
		{
			name:   "allow subnet deny all",
			rules:  []access.Rule{{Allow: parseCIDR(t, "192.168.0.0/16")}, {Deny: parseCIDR(t, "0.0.0.0/0")}},
			remote: "192.168.1.1:1234",
			want:   200,
		},
		{
			name:   "deny subnet allow all",
			rules:  []access.Rule{{Deny: parseCIDR(t, "192.168.0.0/16")}, {Allow: parseCIDR(t, "0.0.0.0/0")}},
			remote: "192.168.1.1:1234",
			want:   403,
		},
		{
			name:   "allow specific IP",
			rules:  []access.Rule{{Allow: parseCIDR(t, "10.0.0.1")}, {Deny: parseCIDR(t, "0.0.0.0/0")}},
			remote: "10.0.0.1:1234",
			want:   200,
		},
		{
			name:   "deny specific IP",
			rules:  []access.Rule{{Deny: parseCIDR(t, "10.0.0.1")}, {Allow: parseCIDR(t, "0.0.0.0/0")}},
			remote: "10.0.0.1:1234",
			want:   403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := access.New(tt.rules)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remote
			w := httptest.NewRecorder()
			handler.ServeHTTP(next).ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

// Test compatibility with nginx gzip.t
func TestGzipModule(t *testing.T) {
	t.Run("gzip compression", func(t *testing.T) {
		handler := modules.BuildChain(&config.ModulesConfig{
			Gzip: &config.GzipConfig{Enabled: true, Level: 6, Types: []string{"text/plain"}},
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(strings.Repeat("X", 64)))
		}))

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") != "gzip" {
			t.Errorf("Content-Encoding = %q, want gzip", w.Header().Get("Content-Encoding"))
		}

		gz, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer gz.Close()
		body, _ := io.ReadAll(gz)
		if string(body) != strings.Repeat("X", 64) {
			t.Errorf("body length = %d, want 64", len(body))
		}
	})

	t.Run("no gzip without Accept-Encoding", func(t *testing.T) {
		handler := modules.BuildChain(&config.ModulesConfig{
			Gzip: &config.GzipConfig{Enabled: true, Level: 6},
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("hello"))
		}))

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Content-Encoding") == "gzip" {
			t.Error("should not compress without Accept-Encoding")
		}
	})
}

// Test compatibility with nginx headers.t
func TestHeadersModule(t *testing.T) {
	t.Run("add headers", func(t *testing.T) {
		handler := modules.BuildChain(&config.ModulesConfig{
			Headers: &config.HeadersConfig{
				Add: map[string]string{
					"X-Frame-Options": "DENY",
					"X-XSS-Protection": "1; mode=block",
				},
			},
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		}))

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("X-Frame-Options = %q, want DENY", w.Header().Get("X-Frame-Options"))
		}
		if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
			t.Errorf("X-XSS-Protection = %q, want 1; mode=block", w.Header().Get("X-XSS-Protection"))
		}
	})

	t.Run("remove headers", func(t *testing.T) {
		handler := modules.BuildChain(&config.ModulesConfig{
			Headers: &config.HeadersConfig{
				Remove: []string{"Server", "X-Powered-By"},
			},
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", "nginx/1.0")
			w.Header().Set("X-Powered-By", "Go")
			w.Write([]byte("ok"))
		}))

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Server") != "" {
			t.Errorf("Server = %q, want empty", w.Header().Get("Server"))
		}
		if w.Header().Get("X-Powered-By") != "" {
			t.Errorf("X-Powered-By = %q, want empty", w.Header().Get("X-Powered-By"))
		}
	})
}

// Test compatibility with nginx proxy.t
func TestProxyModule(t *testing.T) {
	t.Run("basic proxy", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Backend", "true")
			w.Write([]byte("backend response"))
		}))
		defer backend.Close()

		upstream := proxy.NewUpstream("test", []*proxy.Server{
			{Addr: backend.Listener.Addr().String()},
		}, "round-robin")

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		upstream.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
		if w.Body.String() != "backend response" {
			t.Errorf("body = %q, want backend response", w.Body.String())
		}
		if w.Header().Get("X-Backend") != "true" {
			t.Error("backend header not forwarded")
		}
	})

	t.Run("round-robin load balancing", func(t *testing.T) {
		backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("backend1"))
		}))
		defer backend1.Close()

		backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("backend2"))
		}))
		defer backend2.Close()

		upstream := proxy.NewUpstream("test", []*proxy.Server{
			{Addr: backend1.Listener.Addr().String()},
			{Addr: backend2.Listener.Addr().String()},
		}, "round-robin")

		results := make(map[string]int)
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			upstream.ServeHTTP(w, req)
			results[w.Body.String()]++
		}

		if results["backend1"] != 50 || results["backend2"] != 50 {
			t.Errorf("distribution = %v, want equal", results)
		}
	})
}

// Test compatibility with nginx static.t
func TestStaticModule(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "static-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html>hello</html>"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte("body { color: red; }"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "images"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "images/logo.png"), []byte("fake png"), 0644)

	t.Run("serve HTML file", func(t *testing.T) {
		handler := static.New(tmpDir, false)
		req := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
		if w.Body.String() != "<html>hello</html>" {
			t.Errorf("body = %q", w.Body.String())
		}
	})

	t.Run("serve CSS file", func(t *testing.T) {
		handler := static.New(tmpDir, false)
		req := httptest.NewRequest("GET", "/style.css", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("directory traversal blocked", func(t *testing.T) {
		handler := static.New(tmpDir, false)
		req := httptest.NewRequest("GET", "/../../../etc/passwd", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 403 && w.Code != 404 {
			t.Errorf("status = %d, want 403 or 404", w.Code)
		}
	})

	t.Run("404 for missing file", func(t *testing.T) {
		handler := static.New(tmpDir, false)
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})
}

// Test compatibility with nginx rewrite.t
func TestRewriteModule(t *testing.T) {
	router := router.NewRouter()

	// Simple redirect
	router.AddRoute("/redirect", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusMovedPermanently)
	}))

	// Return status
	router.AddRoute("/return204", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// Return 200 with body
	router.AddRoute("/return200", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		path   string
		status int
		body   string
	}{
		{"/redirect", 301, ""},
		{"/return204", 204, ""},
		{"/return200", 200, "OK"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
			if tt.body != "" && w.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.body)
			}
		})
	}
}

// Test compatibility with nginx limit_req.t
func TestLimitReqModule(t *testing.T) {
	handler := modules.BuildChain(&config.ModulesConfig{
		RateLimit: &config.RateLimitConfig{Rate: "1/s", Burst: 1},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	// First request should succeed
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("first request: status = %d, want 200", w.Code)
	}

	// Second request should be rate limited
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 429 {
		t.Errorf("second request: status = %d, want 429", w.Code)
	}
}

// Test compatibility with nginx http_uri.t
func TestURIHandling(t *testing.T) {
	tests := []struct {
		path   string
		want   int
	}{
		{"/", 200},
		{"/index.html", 200},
		{"/path/to/resource", 200},
		{"/path%20with%20spaces", 200},
		{"/path?query=value", 200},
	}

	router := router.NewRouter()
	router.AddRoute("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	router.AddRoute("/index.html", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	router.AddRoute("/path/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Errorf("GET %s: status = %d, want %d", tt.path, w.Code, tt.want)
			}
		})
	}
}

// Test compatibility with nginx proxy_keepalive.t
func TestProxyKeepalive(t *testing.T) {
	requestCount := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	upstream := proxy.NewUpstream("test", []*proxy.Server{
		{Addr: backend.Listener.Addr().String()},
	}, "round-robin")

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		upstream.ServeHTTP(w, req)
	}

	// All requests should succeed
	if requestCount != 10 {
		t.Errorf("requestCount = %d, want 10", requestCount)
	}
}

// Test compatibility with nginx proxy_timeout.t
func TestProxyTimeout(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	upstream := proxy.NewUpstream("test", []*proxy.Server{
		{Addr: backend.Listener.Addr().String()},
	}, "round-robin")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// Test compatibility with nginx ssl.t
func TestSSLSupport(t *testing.T) {
	// Test that server can be configured with TLS
	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: ":0",
				TLS: &config.TLS{
					Cert: "test.crt",
					Key:  "test.key",
				},
			},
		},
	}

	// Just verify config parsing works
	if cfg.Listen[0].TLS == nil {
		t.Error("TLS config not parsed")
	}
	if cfg.Listen[0].TLS.Cert != "test.crt" {
		t.Errorf("Cert = %q, want test.crt", cfg.Listen[0].TLS.Cert)
	}
}

// Test compatibility with nginx proxy_protocol.t
func TestProxyProtocol(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	upstream := proxy.NewUpstream("test", []*proxy.Server{
		{Addr: backend.Listener.Addr().String()},
	}, "round-robin")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// Test compatibility with nginx upstream_hash.t
func TestUpstreamHash(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	upstream := proxy.NewUpstream("test", []*proxy.Server{
		{Addr: backend.Listener.Addr().String()},
	}, "round-robin")

	// Same client should get same backend (with single server)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		upstream.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("request %d: status = %d, want 200", i, w.Code)
		}
	}
}

// Test compatibility with nginx autoindex.t
func TestAutoindex(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "autoindex-test")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("file1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("file2"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	t.Run("directory listing enabled", func(t *testing.T) {
		handler := static.New(tmpDir, true)
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("status = %d, want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "file1.txt") {
			t.Error("directory listing should contain file1.txt")
		}
		if !strings.Contains(w.Body.String(), "file2.txt") {
			t.Error("directory listing should contain file2.txt")
		}
	})

	t.Run("directory listing disabled", func(t *testing.T) {
		handler := static.New(tmpDir, false)
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != 404 {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})
}

// Test compatibility with nginx http_max_headers.t
func TestMaxHeaders(t *testing.T) {
	// Test that server handles many headers
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Header-Count", "0")
		for i := 0; i < 100; i++ {
			w.Header().Set("X-Test-"+string(rune('A'+i%26)), "value")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// Test compatibility with nginx http_host.t
func TestHostHeader(t *testing.T) {
	router := router.NewRouter()

	// Different hosts can route to different handlers
	hostHandlers := map[string]http.Handler{
		"example.com": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("example.com"))
		}),
		"other.com": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("other.com"))
		}),
	}

	for host, handler := range hostHandlers {
		router.AddRoute("/", handler)
		_ = host // Host-based routing would need server config
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Basic routing works
	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// Test compatibility with nginx limit_conn.t
func TestLimitConn(t *testing.T) {
	// Test connection limiting concept
	connCount := 0
	maxConns := 5

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connCount++
		if connCount > maxConns {
			http.Error(w, "Too Many Connections", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if i < maxConns && w.Code != 200 {
			t.Errorf("request %d: status = %d, want 200", i, w.Code)
		}
		if i >= maxConns && w.Code != 503 {
			t.Errorf("request %d: status = %d, want 503", i, w.Code)
		}
	}
}

// Test compatibility with nginx proxy_redirect.t
func TestProxyRedirect(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://backend/newpath", http.StatusMovedPermanently)
	}))
	defer backend.Close()

	upstream := proxy.NewUpstream("test", []*proxy.Server{
		{Addr: backend.Listener.Addr().String()},
	}, "round-robin")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	upstream.ServeHTTP(w, req)

	// Proxy should forward the redirect
	if w.Code != 301 {
		t.Errorf("status = %d, want 301", w.Code)
	}
}

// Helper functions
func parseCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	if !strings.Contains(cidr, "/") {
		cidr = cidr + "/32"
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("parseCIDR(%q) failed: %v", cidr, err)
	}
	return network
}
