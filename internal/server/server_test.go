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
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/net/http2"

	"github.com/user/gore/internal/config"
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
