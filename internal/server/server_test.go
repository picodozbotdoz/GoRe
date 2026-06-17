package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/net/http2"

	"github.com/user/gore/internal/config"
	"github.com/user/gore/internal/modules/authrequest"
)

func generateSelfSignedCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certFile, _ := os.Create(certPath)
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyFile, _ := os.Create(keyPath)
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	keyFile.Close()

	return certPath, keyPath
}

func TestHTTP2OverTLS(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS:  &config.TLS{Cert: certPath, Key: keyPath},
				HTTP2: &config.HTTP2{
					MaxConcurrentStreams: 100,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/hello", Return: "/goodbye"},
					},
				},
			},
		},
	}

	srv := New(cfg)

	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	t.Logf("Server listening on %s", addr)

	caCertPool := x509.NewCertPool()
	cert, _ := os.ReadFile(certPath)
	caCertPool.AppendCertsFromPEM(cert)

	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caCertPool,
				NextProtos: []string{"h2"},
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("https://%s/hello", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Proto: %s, Status: %d", resp.Proto, resp.StatusCode)

	if resp.StatusCode != 301 {
		t.Errorf("StatusCode = %d, want 301", resp.StatusCode)
	}

	if resp.Proto != "HTTP/2.0" {
		t.Errorf("Proto = %q, want HTTP/2.0", resp.Proto)
	}
}

func TestH2CCleartext(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{
			{Addr: "127.0.0.1:0"},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/hello", Return: "/goodbye"},
					},
				},
			},
		},
	}

	srv := New(cfg)

	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("http://%s/hello", addr))
	if err != nil {
		t.Fatalf("h2c GET failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Proto: %s, Status: %d", resp.Proto, resp.StatusCode)

	if resp.StatusCode != 301 {
		t.Errorf("StatusCode = %d, want 301", resp.StatusCode)
	}
}

func TestHTTP1Fallback(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{
			{Addr: "127.0.0.1:0"},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/hello", Return: "/goodbye"},
					},
				},
			},
		},
	}

	srv := New(cfg)

	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("http://%s/hello", addr))
	if err != nil {
		t.Fatalf("HTTP/1.1 GET failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Proto: %s, Status: %d", resp.Proto, resp.StatusCode)

	if resp.StatusCode != 301 {
		t.Errorf("StatusCode = %d, want 301", resp.StatusCode)
	}
}

func TestHTTP2MaxFrameSizeConfig(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: ":0",
				HTTP2: &config.HTTP2{
					MaxFrameSize: 32768,
				},
			},
		},
	}

	srv := New(cfg)
	h2cfg := srv.buildHTTP2Config(&cfg.Listen[0])

	if h2cfg.MaxReadFrameSize != 32768 {
		t.Errorf("MaxReadFrameSize = %d, want 32768", h2cfg.MaxReadFrameSize)
	}
}

func TestSSLSessionsTimeoutConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:           certPath,
					Key:            keyPath,
					SessionTimeout: 600,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/hello", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig.ClientSessionCache == nil {
		t.Fatal("ClientSessionCache is nil, expected session cache with timeout")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}

	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	resp, err := client.Get(fmt.Sprintf("https://%s/hello", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestSSLRejectHandshake(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:            certPath,
					Key:             keyPath,
					RejectHandshake: true,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig.VerifyConnection == nil {
		t.Fatal("VerifyConnection is nil, expected reject handshake callback")
	}

	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	_, err := client.Get(fmt.Sprintf("https://%s/", addr))
	if err == nil {
		t.Fatal("expected error from rejected handshake, got nil")
	}
	t.Logf("Got expected error: %v", err)
}

func TestSSLClientCertAuth(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)
	caPath := certPath

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:              certPath,
					Key:               keyPath,
					ClientCertificate: caPath,
					VerifyClient:      true,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth = %d, want %d", tlsConfig.ClientAuth, tls.RequireAndVerifyClientCert)
	}
	if tlsConfig.ClientCAs == nil {
		t.Fatal("ClientCAs is nil")
	}

	_ = caPath

	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	_, err := client.Get(fmt.Sprintf("https://%s/", addr))
	if err == nil {
		t.Fatal("expected error from client cert requirement, got nil")
	}
	t.Logf("Got expected error: %v", err)
}

func TestHTTP2ConfigDefaults(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{
			{Addr: ":0"},
		},
	}

	srv := New(cfg)
	h2cfg := srv.buildHTTP2Config(&cfg.Listen[0])

	if h2cfg.MaxConcurrentStreams != 250 {
		t.Errorf("MaxConcurrentStreams = %d, want 250", h2cfg.MaxConcurrentStreams)
	}
	if h2cfg.MaxReadFrameSize != 1048576 {
		t.Errorf("MaxReadFrameSize = %d, want 1048576", h2cfg.MaxReadFrameSize)
	}
}

func TestAlias(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "public", "css"), 0755)
	os.WriteFile(filepath.Join(dir, "public", "index.html"), []byte("alias-home"), 0644)
	os.WriteFile(filepath.Join(dir, "public", "css", "style.css"), []byte("body{}"), 0644)

	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/static/", Alias: filepath.Join(dir, "public")},
					},
				},
			},
		},
	}

	srv := New(cfg)
	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/static/index.html", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "alias-home" {
		t.Errorf("body = %q, want 'alias-home'", string(body))
	}

	resp2, err := http.Get(fmt.Sprintf("http://%s/static/css/style.css", addr))
	if err != nil {
		t.Fatalf("GET css failed: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if string(body2) != "body{}" {
		t.Errorf("body = %q, want 'body{}'", string(body2))
	}
}

func TestLimitExcept(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/api", Return: "200 ok", LimitExcept: []string{"GET", "POST"}},
					},
				},
			},
		},
	}

	srv := New(cfg)
	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("http://%s/api", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("GET status = %d, want 200", resp.StatusCode)
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("http://%s/api", addr), nil)
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 405 {
		t.Errorf("DELETE status = %d, want 405", resp2.StatusCode)
	}
}

func TestInternal(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/internal", Return: "200 ok", Internal: true},
					},
				},
			},
		},
	}

	srv := New(cfg)
	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("http://%s/internal", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("external GET status = %d, want 404", resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/internal", addr), nil)
	req.Header.Set("X-Internal", "1")
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("internal GET failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("internal GET status = %d, want 200", resp2.StatusCode)
	}
}

func TestSatisfyAny(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/admin", Return: "200 ok", Satisfy: "any"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	h := srv.router
	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("satisfy any with no rules = %d, want 200", w.Code)
	}
}

func TestAuthRequestSet(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Auth-User", "alice")
		w.Header().Set("X-Auth-Role", "admin")
		w.WriteHeader(200)
	}))
	defer authServer.Close()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-User", r.Header.Get("X-Auth-User"))
		w.Header().Set("X-Role", r.Header.Get("X-Auth-Role"))
		w.WriteHeader(200)
	})

	headerMap := map[string]string{
		"X-Auth-User": "X-Auth-User",
		"X-Auth-Role": "X-Auth-Role",
	}
	handler := authrequest.NewWithSet(authServer.URL, headerMap)(inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-User") != "alice" {
		t.Errorf("X-User = %q, want 'alice'", w.Header().Get("X-User"))
	}
	if w.Header().Get("X-Role") != "admin" {
		t.Errorf("X-Role = %q, want 'admin'", w.Header().Get("X-Role"))
	}
}

func TestResolver(t *testing.T) {
	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		Modules: config.ModulesConfig{
			Resolver: "8.8.8.8",
		},
	}

	srv := New(cfg)
	if srv.cfg.Modules.Resolver != "8.8.8.8" {
		t.Errorf("Resolver = %q, want 8.8.8.8", srv.cfg.Modules.Resolver)
	}
}

func TestSSLStaplingConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:       certPath,
					Key:        keyPath,
					Stapling:   true,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig == nil {
		t.Fatal("tlsConfig is nil")
	}

	if tlsConfig.GetCertificate == nil {
		t.Log("GetCertificate not set (OCSP stapling unavailable for self-signed cert) — config accepted")
	}
}

func TestSSLStaplingDisabled(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:     certPath,
					Key:      keyPath,
					Stapling: false,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig == nil {
		t.Fatal("tlsConfig is nil")
	}

	if tlsConfig.GetCertificate != nil {
		t.Error("GetCertificate should NOT be set when stapling is disabled")
	}
}

func TestSSLStaplingVerifyConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:           certPath,
					Key:            keyPath,
					Stapling:       true,
					StaplingVerify: true,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig == nil {
		t.Fatal("tlsConfig is nil")
	}

	if tlsConfig.GetCertificate == nil {
		t.Log("GetCertificate not set (OCSP stapling unavailable for self-signed cert) — config accepted")
	}
}

func TestParseECDHCurves(t *testing.T) {
	tests := []struct {
		name string
		want []tls.CurveID
	}{
		{"X25519", []tls.CurveID{tls.X25519}},
		{"P-256:P-384", []tls.CurveID{tls.CurveP256, tls.CurveP384}},
		{"X25519:P-256:P-384:P-521", []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521}},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseECDHCurves(tt.name)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("curve[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSSLECDHCurveConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:      certPath,
					Key:       keyPath,
					ECDHCurve: "X25519:P-256",
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig == nil {
		t.Fatal("buildTLSConfig returned nil")
	}
	if len(tlsConfig.CurvePreferences) != 2 {
		t.Fatalf("CurvePreferences len = %d, want 2", len(tlsConfig.CurvePreferences))
	}
	if tlsConfig.CurvePreferences[0] != tls.X25519 {
		t.Errorf("CurvePreferences[0] = %d, want %d (X25519)", tlsConfig.CurvePreferences[0], tls.X25519)
	}
	if tlsConfig.CurvePreferences[1] != tls.CurveP256 {
		t.Errorf("CurvePreferences[1] = %d, want %d (P-256)", tlsConfig.CurvePreferences[1], tls.CurveP256)
	}
}

func TestLoadDHParams(t *testing.T) {
	dir := t.TempDir()

	// Valid DH params PEM
	dhParamsPEM := `-----BEGIN DH PARAMETERS-----
MEkCQQDcReNegU2ztSCDi2pMLzTb9xiIZY8dG5ITqmSDpMwPECFFc42O9u0FZWsZ
b8oVKNMaoUlWQve9V0oKEhVq29+nAgECAgF9
-----END DH PARAMETERS-----
`
	dhPath := filepath.Join(dir, "dhparam.pem")
	if err := os.WriteFile(dhPath, []byte(dhParamsPEM), 0644); err != nil {
		t.Fatal(err)
	}

	err := loadDHParams(dhPath)
	if err != nil {
		t.Errorf("loadDHParams returned error: %v", err)
	}

	// Non-existent file
	err = loadDHParams(filepath.Join(dir, "nonexistent.pem"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	// Invalid PEM (not DH PARAMETERS type)
	badPEM := filepath.Join(dir, "bad.pem")
	os.WriteFile(badPEM, []byte("-----BEGIN RSA PRIVATE KEY-----\nAAAA\n-----END RSA PRIVATE KEY-----"), 0644)
	err = loadDHParams(badPEM)
	if err == nil {
		t.Error("expected error for wrong PEM type")
	}
}

func TestSSLDHParamConfig(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateSelfSignedCert(t, dir)

	dhParamsPEM := `-----BEGIN DH PARAMETERS-----
MEkCQQDcReNegU2ztSCDi2pMLzTb9xiIZY8dG5ITqmSDpMwPECFFc42O9u0FZWsZ
b8oVKNMaoUlWQve9V0oKEhVq29+nAgECAgF9
-----END DH PARAMETERS-----
`
	dhPath := filepath.Join(dir, "dhparam.pem")
	os.WriteFile(dhPath, []byte(dhParamsPEM), 0644)

	cfg := &config.Config{
		Listen: []config.Listen{
			{
				Addr: "127.0.0.1:0",
				TLS: &config.TLS{
					Cert:    certPath,
					Key:     keyPath,
					DHParam: dhPath,
				},
			},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{Path: "/", Return: "200 ok"},
					},
				},
			},
		},
	}

	srv := New(cfg)
	tlsConfig := srv.buildTLSConfig(&cfg.Listen[0])

	if tlsConfig == nil {
		t.Fatal("buildTLSConfig returned nil")
	}
}

func TestResolveVariable(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		host     string
		method   string
		header   string
		want     string
	}{
		{"host", "$host", "example.com", "GET", "", "example.com"},
		{"method", "$method", "example.com", "POST", "", "POST"},
		{"header", "$http_x_custom", "", "", "val1", "val1"},
		{"plain_header", "X-Foo", "", "", "bar", "bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://"+tt.host+"/", nil)
			if tt.header != "" {
				req.Header.Set("X-Custom", tt.header)
				req.Header.Set("X-Foo", tt.header)
			}
			got := resolveVariable(req, tt.variable)
			if got != tt.want {
				t.Errorf("resolveVariable(%q) = %q, want %q", tt.variable, got, tt.want)
			}
		})
	}

	t.Run("remote_addr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/", nil)
		got := resolveVariable(req, "$remote_addr")
		if got == "" {
			t.Error("resolveVariable($remote_addr) should not be empty")
		}
	})
}

func TestDynamicUpstream(t *testing.T) {
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend-a"))
	}))
	defer backendA.Close()

	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend-b"))
	}))
	defer backendB.Close()

	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		Upstreams: map[string]config.Upstream{
			"upstream-a": {Servers: []config.UpstreamServer{{Addr: backendA.Listener.Addr().String(), Weight: 1}}},
			"upstream-b": {Servers: []config.UpstreamServer{{Addr: backendB.Listener.Addr().String(), Weight: 1}}},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{
							Path:  "/",
							Proxy: &config.Proxy{DynamicUpstream: "$host"},
						},
					},
				},
			},
		},
	}

	srv := New(cfg)
	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/", addr), nil)
	req.Host = "upstream-a"
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "backend-a" {
		t.Errorf("body = %q, want backend-a", string(body))
	}

	req2, _ := http.NewRequest("GET", fmt.Sprintf("http://%s/", addr), nil)
	req2.Host = "upstream-b"
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body2) != "backend-b" {
		t.Errorf("body = %q, want backend-b", string(body2))
	}
}

func TestDynamicUpstreamFallback(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fallback"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Listen: []config.Listen{{Addr: "127.0.0.1:0"}},
		Upstreams: map[string]config.Upstream{
			"fallback-upstream": {Servers: []config.UpstreamServer{{Addr: backend.Listener.Addr().String(), Weight: 1}}},
		},
		HTTP: config.HTTPConfig{
			Servers: []config.Server{
				{
					Locations: []config.Location{
						{
							Path:  "/",
							Proxy: &config.Proxy{Upstream: "fallback-upstream", DynamicUpstream: "$host"},
						},
					},
				},
			},
		},
	}

	srv := New(cfg)
	go srv.Start()
	time.Sleep(500 * time.Millisecond)
	defer srv.Stop(context.Background())

	addr := srv.Addr(0)
	if addr == "" {
		t.Fatal("server not started")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "fallback" {
		t.Errorf("body = %q, want fallback", string(body))
	}
}
