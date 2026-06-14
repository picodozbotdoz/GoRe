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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
)

func generateTestCert(t *testing.T, dir string) (certPath, keyPath string) {
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
	cf, _ := os.Create(certPath)
	pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	cf.Close()
	kDER, _ := x509.MarshalECPrivateKey(key)
	kf, _ := os.Create(keyPath)
	pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kDER})
	kf.Close()
	return certPath, keyPath
}

func h3TestSetup(t *testing.T, handler http.Handler) (addr string, certPath string, stop func()) {
	t.Helper()
	dir := t.TempDir()
	certPath, keyPath := generateTestCert(t, dir)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr = ln.Addr().String()
	ln.Close()

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	h3Srv := &http3.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	udpLn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}

	go h3Srv.Serve(udpLn)
	time.Sleep(100 * time.Millisecond)

	return addr, certPath, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h3Srv.Shutdown(ctx)
	}
}

func h3TestClient(certPath string) *http.Client {
	caCert, _ := os.ReadFile(certPath)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return &http.Client{
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caCertPool,
				NextProtos: []string{"h3"},
			},
		},
		Timeout: 5 * time.Second,
	}
}

func TestH3Get(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello h3"))
	})

	addr, certPath, stop := h3TestSetup(t, mux)
	defer stop()

	client := h3TestClient(certPath)
	resp, err := client.Get(fmt.Sprintf("https://%s/", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Proto != "HTTP/3.0" {
		t.Errorf("proto = %q, want HTTP/3.0", resp.Proto)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello h3" {
		t.Errorf("body = %q, want %q", string(body), "hello h3")
	}
}

func TestH3Multiplexed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	addr, certPath, stop := h3TestSetup(t, mux)
	defer stop()

	client := h3TestClient(certPath)

	for i := 0; i < 10; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/", addr))
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("request %d: status = %d, want 200", i, resp.StatusCode)
		}
		if resp.Proto != "HTTP/3.0" {
			t.Errorf("request %d: proto = %q, want HTTP/3.0", i, resp.Proto)
		}
	}
}

func TestH3PostBody(t *testing.T) {
	var received string
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		received = string(body)
		mu.Unlock()
		w.WriteHeader(200)
		w.Write(body)
	})

	addr, certPath, stop := h3TestSetup(t, mux)
	defer stop()

	client := h3TestClient(certPath)
	resp, err := client.Post(
		fmt.Sprintf("https://%s/echo", addr),
		"text/plain",
		strings.NewReader("h3-body-data"),
	)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "h3-body-data" {
		t.Errorf("body = %q, want %q", string(body), "h3-body-data")
	}
	mu.Lock()
	got := received
	mu.Unlock()
	if got != "h3-body-data" {
		t.Errorf("server received = %q, want %q", got, "h3-body-data")
	}
}

func TestH3Headers(t *testing.T) {
	var receivedFoo string
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedFoo = r.Header.Get("X-Foo")
		mu.Unlock()
		w.Header().Set("X-Response", "bar")
		w.Header().Set("X-Echo", r.Header.Get("X-Foo"))
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	addr, certPath, stop := h3TestSetup(t, mux)
	defer stop()

	client := h3TestClient(certPath)
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://%s/", addr), nil)
	req.Header.Set("X-Foo", "baz")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Response") != "bar" {
		t.Errorf("X-Response = %q, want %q", resp.Header.Get("X-Response"), "bar")
	}
	if resp.Header.Get("X-Echo") != "baz" {
		t.Errorf("X-Echo = %q, want %q", resp.Header.Get("X-Echo"), "baz")
	}
	mu.Lock()
	got := receivedFoo
	mu.Unlock()
	if got != "baz" {
		t.Errorf("server received X-Foo = %q, want %q", got, "baz")
	}
}

func TestH3Redirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/new", 301)
	})
	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("redirected"))
	})

	addr, _, stop := h3TestSetup(t, mux)
	defer stop()

	client := &http.Client{
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				NextProtos:         []string{"h3"},
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("https://%s/old", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 301 {
		t.Errorf("status = %d, want 301", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/new" {
		t.Errorf("Location = %q, want %q", loc, "/new")
	}
}

func TestH3LargeResponse(t *testing.T) {
	largeBody := strings.Repeat("X", 100000)
	mux := http.NewServeMux()
	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(largeBody)))
		w.Write([]byte(largeBody))
	})

	addr, certPath, stop := h3TestSetup(t, mux)
	defer stop()

	client := h3TestClient(certPath)
	resp, err := client.Get(fmt.Sprintf("https://%s/big", addr))
	if err != nil {
		t.Fatalf("GET failed: %v", err)
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

func TestH3ConnectionReuse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	addr, certPath, stop := h3TestSetup(t, mux)
	defer stop()

	client := h3TestClient(certPath)

	for i := 0; i < 20; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/", addr))
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("request %d: status = %d, want 200", i, resp.StatusCode)
		}
	}
}
