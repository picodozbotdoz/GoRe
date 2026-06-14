package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	yaml := `
worker_processes: auto
listen:
  - addr: ":80"
  - addr: ":443"
    tls:
      cert: /etc/ssl/cert.pem
      key: /etc/ssl/key.pem
http:
  server:
    - name: example.com
      locations:
        - path: /static/
          root: /var/www
        - path: /api/
          proxy:
            upstream: backend
upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:8080
        weight: 1
modules:
  gzip:
    enabled: true
    level: 6
  access:
    rules:
      - allow: 192.168.0.0/16
      - deny: all
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(yaml)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WorkerProcesses != "auto" {
		t.Errorf("WorkerProcesses = %q, want %q", cfg.WorkerProcesses, "auto")
	}
	if len(cfg.Listen) != 2 {
		t.Errorf("len(Listen) = %d, want 2", len(cfg.Listen))
	}
	if cfg.Listen[1].TLS == nil {
		t.Error("Listen[1].TLS is nil, want non-nil")
	}
	if len(cfg.HTTP.Servers) != 1 {
		t.Errorf("len(Servers) = %d, want 1", len(cfg.HTTP.Servers))
	}
	if cfg.Modules.Gzip == nil || !cfg.Modules.Gzip.Enabled {
		t.Error("Gzip not enabled")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	yaml := `
listen:
  - addr: ":8080"
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(yaml)
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WorkerProcesses != "auto" {
		t.Errorf("WorkerProcesses = %q, want auto", cfg.WorkerProcesses)
	}
}
