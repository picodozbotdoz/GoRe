package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAliasField(t *testing.T) {
	cfg := `http:
  server:
    - locations:
        - path: /static
          alias: /var/www/public
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTP.Servers[0].Locations[0].Alias != "/var/www/public" {
		t.Errorf("Alias = %q, want /var/www/public", c.HTTP.Servers[0].Locations[0].Alias)
	}
}

func TestLimitExceptField(t *testing.T) {
	cfg := `http:
  server:
    - locations:
        - path: /api
          proxy:
            upstream: backend
          limit_except:
            - GET
            - POST
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	loc := c.HTTP.Servers[0].Locations[0]
	if len(loc.LimitExcept) != 2 || loc.LimitExcept[0] != "GET" || loc.LimitExcept[1] != "POST" {
		t.Errorf("LimitExcept = %v, want [GET POST]", loc.LimitExcept)
	}
}

func TestInternalField(t *testing.T) {
	cfg := `http:
  server:
    - locations:
        - path: /internal
          return: "200 ok"
          internal: true
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !c.HTTP.Servers[0].Locations[0].Internal {
		t.Error("Internal should be true")
	}
}

func TestSatisfyField(t *testing.T) {
	cfg := `http:
  server:
    - locations:
        - path: /admin
          proxy:
            upstream: backend
          satisfy: any
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTP.Servers[0].Locations[0].Satisfy != "any" {
		t.Errorf("Satisfy = %q, want 'any'", c.HTTP.Servers[0].Locations[0].Satisfy)
	}
}

func TestAuthRequestSetField(t *testing.T) {
	cfg := `http:
  server:
    - locations:
        - path: /admin
          auth_request: http://auth/internal
          auth_request_set:
            X-User: $upstream_http_x_auth_user
            X-Role: $upstream_http_x_auth_role
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	loc := c.HTTP.Servers[0].Locations[0]
	if len(loc.AuthRequestSet) != 2 {
		t.Fatalf("AuthRequestSet len = %d, want 2", len(loc.AuthRequestSet))
	}
	if loc.AuthRequestSet["X-User"] != "$upstream_http_x_auth_user" {
		t.Errorf("X-User = %q", loc.AuthRequestSet["X-User"])
	}
}

func TestECDHCurveField(t *testing.T) {
	cfg := `listen:
  - addr: ":443"
    tls:
      cert: cert.pem
      key: key.pem
      ecdh_curve: X25519:P-256
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen[0].TLS.ECDHCurve != "X25519:P-256" {
		t.Errorf("ECDHCurve = %q, want X25519:P-256", c.Listen[0].TLS.ECDHCurve)
	}
}

func TestDHParamField(t *testing.T) {
	cfg := `listen:
  - addr: ":443"
    tls:
      cert: cert.pem
      key: key.pem
      dhparam: /etc/ssl/dhparam.pem
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen[0].TLS.DHParam != "/etc/ssl/dhparam.pem" {
		t.Errorf("DHParam = %q, want /etc/ssl/dhparam.pem", c.Listen[0].TLS.DHParam)
	}
}

func TestCRLField(t *testing.T) {
	cfg := `listen:
  - addr: ":443"
    tls:
      cert: cert.pem
      key: key.pem
      crl: /etc/ssl/crl.pem
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen[0].TLS.CRL != "/etc/ssl/crl.pem" {
		t.Errorf("CRL = %q, want /etc/ssl/crl.pem", c.Listen[0].TLS.CRL)
	}
}

func TestPasswordFileField(t *testing.T) {
	cfg := `listen:
  - addr: ":443"
    tls:
      cert: cert.pem
      key: key.pem
      password_file: /etc/ssl/password.txt
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen[0].TLS.PasswordFile != "/etc/ssl/password.txt" {
		t.Errorf("PasswordFile = %q, want /etc/ssl/password.txt", c.Listen[0].TLS.PasswordFile)
	}
}

func TestEarlyDataField(t *testing.T) {
	cfg := `listen:
  - addr: ":443"
    tls:
      cert: cert.pem
      key: key.pem
      early_data: true
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Listen[0].TLS.EarlyData {
		t.Error("EarlyData should be true")
	}
}

func TestDynamicUpstreamField(t *testing.T) {
	cfg := `http:
  server:
    - locations:
        - path: /
          proxy:
            upstream: default_backend
            dynamic_upstream: $host
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	proxy := c.HTTP.Servers[0].Locations[0].Proxy
	if proxy.DynamicUpstream != "$host" {
		t.Errorf("DynamicUpstream = %q, want $host", proxy.DynamicUpstream)
	}
	if proxy.GetDynamicUpstream() != "host" {
		t.Errorf("GetDynamicUpstream() = %q, want host", proxy.GetDynamicUpstream())
	}
}

func TestDynamicUpstreamEmpty(t *testing.T) {
	proxy := &Proxy{Upstream: "backend"}
	if proxy.GetDynamicUpstream() != "" {
		t.Errorf("GetDynamicUpstream() = %q, want empty", proxy.GetDynamicUpstream())
	}
}

func TestResolverField(t *testing.T) {
	cfg := `modules:
  resolver: 8.8.8.8
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Modules.Resolver != "8.8.8.8" {
		t.Errorf("Resolver = %q, want 8.8.8.8", c.Modules.Resolver)
	}
}

func TestSSLStaplingFields(t *testing.T) {
	cfg := `listen:
  - addr: :443
    tls:
      cert: /path/to/cert.pem
      key: /path/to/key.pem
      stapling: true
      stapling_verify: true
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Listen) == 0 {
		t.Fatal("no listen entries")
	}
	tls := c.Listen[0].TLS
	if tls == nil {
		t.Fatal("TLS config is nil")
	}
	if !tls.Stapling {
		t.Error("Stapling should be true")
	}
	if !tls.StaplingVerify {
		t.Error("StaplingVerify should be true")
	}
}

func TestSSLStaplingDefaults(t *testing.T) {
	cfg := `listen:
  - addr: :443
    tls:
      cert: /path/to/cert.pem
      key: /path/to/key.pem
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	tls := c.Listen[0].TLS
	if tls.Stapling {
		t.Error("Stapling should default to false")
	}
	if tls.StaplingVerify {
		t.Error("StaplingVerify should default to false")
	}
}

func TestResolveField(t *testing.T) {
	cfg := `upstreams:
  backend:
    strategy: round-robin
    resolve: true
    servers:
      - addr: backend.local:8080
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Upstreams["backend"].Resolve {
		t.Error("Resolve should be true")
	}
}

func TestZoneField(t *testing.T) {
	cfg := `upstreams:
  backend:
    strategy: round-robin
    zone: backend_zone
    servers:
      - addr: 127.0.0.1:8080
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Upstreams["backend"].Zone != "backend_zone" {
		t.Errorf("Zone = %q, want backend_zone", c.Upstreams["backend"].Zone)
	}
}

func TestSlowStartField(t *testing.T) {
	cfg := `upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:8080
        weight: 5
        slow_start: 30
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cfg.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(filepath.Join(dir, "cfg.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Upstreams["backend"].Servers[0].SlowStart != 30 {
		t.Errorf("SlowStart = %d, want 30", c.Upstreams["backend"].Servers[0].SlowStart)
	}
}
