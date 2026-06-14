package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	yaml := `
worker_processes: 2
listen:
  - addr: ":8080"
  - addr: ":8443"
    tls:
      cert: server.crt
      key: server.key
    http2:
      enabled: true
      max_concurrent_streams: 100
      max_frame_size: 16384
http:
  server:
    - name: example.com
      locations:
        - path: /
          root: /var/www/html
        - path: /api/
          proxy:
            upstream: backend
upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:9000
        weight: 2
      - addr: 127.0.0.1:9001
        weight: 1
modules:
  gzip:
    enabled: true
    level: 6
  access:
    rules:
      - allow: 192.168.0.0/16
      - deny: all
  rate_limit:
    rate: 100r/s
    burst: 200
  headers:
    add:
      X-Frame-Options: DENY
    remove:
      - Server
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yaml); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WorkerProcesses != "2" {
		t.Errorf("WorkerProcesses = %q, want %q", cfg.WorkerProcesses, "2")
	}
	if len(cfg.Listen) != 2 {
		t.Fatalf("len(Listen) = %d, want 2", len(cfg.Listen))
	}
	if cfg.Listen[0].Addr != ":8080" {
		t.Errorf("Listen[0].Addr = %q, want %q", cfg.Listen[0].Addr, ":8080")
	}
	if cfg.Listen[1].TLS == nil {
		t.Fatal("Listen[1].TLS is nil")
	}
	if cfg.Listen[1].TLS.Cert != "server.crt" {
		t.Errorf("TLS.Cert = %q, want %q", cfg.Listen[1].TLS.Cert, "server.crt")
	}
	if cfg.Listen[1].HTTP2 == nil {
		t.Fatal("Listen[1].HTTP2 is nil")
	}
	if cfg.Listen[1].HTTP2.MaxConcurrentStreams != 100 {
		t.Errorf("HTTP2.MaxConcurrentStreams = %d, want 100", cfg.Listen[1].HTTP2.MaxConcurrentStreams)
	}
	if cfg.Listen[1].HTTP2.MaxFrameSize != 16384 {
		t.Errorf("HTTP2.MaxFrameSize = %d, want 16384", cfg.Listen[1].HTTP2.MaxFrameSize)
	}
}

func TestHTTP2Defaults(t *testing.T) {
	var h *HTTP2
	if h.GetMaxConcurrentStreams() != 250 {
		t.Errorf("nil GetMaxConcurrentStreams() = %d, want 250", h.GetMaxConcurrentStreams())
	}
	if h.GetMaxFrameSize() != 1048576 {
		t.Errorf("nil GetMaxFrameSize() = %d, want 1048576", h.GetMaxFrameSize())
	}
	h2 := &HTTP2{}
	if h2.GetMaxConcurrentStreams() != 250 {
		t.Errorf("empty GetMaxConcurrentStreams() = %d, want 250", h2.GetMaxConcurrentStreams())
	}

	h3 := &HTTP2{MaxConcurrentStreams: 50}
	if h3.GetMaxConcurrentStreams() != 50 {
		t.Errorf("configured GetMaxConcurrentStreams() = %d, want 50", h3.GetMaxConcurrentStreams())
	}
}
