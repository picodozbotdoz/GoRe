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
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func benchCert(dir string) (certPath, keyPath string) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")
	cf, _ := os.Create(certPath)
	pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	cf.Close()
	kDER, _ := x509.MarshalECPrivateKey(key)
	kf, _ := os.Create(keyPath)
	pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kDER})
	kf.Close()
	return
}

func benchH1(b *testing.B, handler http.Handler) (addr string, cleanup func()) {
	b.Helper()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr = ln.Addr().String()
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	return addr, func() { srv.Shutdown(context.Background()) }
}

func benchH2(b *testing.B, handler http.Handler) (addr string, cleanup func()) {
	b.Helper()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr = ln.Addr().String()
	h2cfg := &http2.Server{MaxConcurrentStreams: 250}
	h := h2c.NewHandler(handler, h2cfg)
	srv := &http.Server{Handler: h}
	go srv.Serve(ln)
	return addr, func() { srv.Shutdown(context.Background()) }
}

func benchH2TLS(b *testing.B, handler http.Handler) (addr string, certPath string, cleanup func()) {
	b.Helper()
	dir := b.TempDir()
	certPath, keyPath := benchCert(dir)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr = ln.Addr().String()
	cert, _ := tls.LoadX509KeyPair(certPath, keyPath)
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}
	srv := &http.Server{Handler: handler, TLSConfig: tlsCfg}
	h2cfg := &http2.Server{MaxConcurrentStreams: 250}
	http2.ConfigureServer(srv, h2cfg)
	go srv.Serve(tls.NewListener(ln, tlsCfg))
	return addr, certPath, func() { srv.Shutdown(context.Background()) }
}

func benchH3(b *testing.B, handler http.Handler) (addr string, certPath string, cleanup func()) {
	b.Helper()
	dir := b.TempDir()
	certPath, keyPath := benchCert(dir)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr = ln.Addr().String()
	ln.Close()
	cert, _ := tls.LoadX509KeyPair(certPath, keyPath)
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}
	h3Srv := &http3.Server{Addr: addr, TLSConfig: tlsCfg, Handler: handler}
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	udpLn, _ := net.ListenUDP("udp", udpAddr)
	go h3Srv.Serve(udpLn)
	return addr, certPath, func() { h3Srv.Shutdown(context.Background()) }
}

var (
	benchSmallBody = []byte("OK")
	benchMedBody   = []byte(strings.Repeat("A", 1024))
	benchLargeBody = []byte(strings.Repeat("B", 65536))
)

func benchMux(body []byte, headers int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/bench", func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < headers; i++ {
			w.Header().Set(fmt.Sprintf("X-Custom-%d", i), "value")
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		w.Write(body)
	})
	return mux
}

func benchClientH1(addr string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
}

func benchClientH2(addr string) *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
		Timeout: 10 * time.Second,
	}
}

func benchClientH2TLS(certPath string) *http.Client {
	ca, _ := os.ReadFile(certPath)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	return &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, NextProtos: []string{"h2"}},
		},
		Timeout: 10 * time.Second,
	}
}

func benchClientH3(certPath string) *http.Client {
	ca, _ := os.ReadFile(certPath)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	return &http.Client{
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, NextProtos: []string{"h3"}},
		},
		Timeout: 10 * time.Second,
	}
}

// --- HTTP/1.1 Benchmarks ---

func BenchmarkH1Small(b *testing.B) {
	addr, cleanup := benchH1(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH1(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH1Medium(b *testing.B) {
	addr, cleanup := benchH1(b, benchMux(benchMedBody, 0))
	defer cleanup()
	client := benchClientH1(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH1Large(b *testing.B) {
	addr, cleanup := benchH1(b, benchMux(benchLargeBody, 0))
	defer cleanup()
	client := benchClientH1(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// --- HTTP/2 Cleartext (h2c) Benchmarks ---

func BenchmarkH2CSmall(b *testing.B) {
	addr, cleanup := benchH2(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH2(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH2CMedium(b *testing.B) {
	addr, cleanup := benchH2(b, benchMux(benchMedBody, 0))
	defer cleanup()
	client := benchClientH2(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH2CLarge(b *testing.B) {
	addr, cleanup := benchH2(b, benchMux(benchLargeBody, 0))
	defer cleanup()
	client := benchClientH2(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH2CWithHeaders(b *testing.B) {
	addr, cleanup := benchH2(b, benchMux(benchSmallBody, 10))
	defer cleanup()
	client := benchClientH2(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// --- HTTP/2 TLS Benchmarks ---

func BenchmarkH2TLSSmall(b *testing.B) {
	addr, certPath, cleanup := benchH2TLS(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH2TLS(certPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH2TLSLarge(b *testing.B) {
	addr, certPath, cleanup := benchH2TLS(b, benchMux(benchLargeBody, 0))
	defer cleanup()
	client := benchClientH2TLS(certPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// --- HTTP/3 QUIC Benchmarks ---

func BenchmarkH3Small(b *testing.B) {
	addr, certPath, cleanup := benchH3(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH3(certPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH3Medium(b *testing.B) {
	addr, certPath, cleanup := benchH3(b, benchMux(benchMedBody, 0))
	defer cleanup()
	client := benchClientH3(certPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkH3Large(b *testing.B) {
	addr, certPath, cleanup := benchH3(b, benchMux(benchLargeBody, 0))
	defer cleanup()
	client := benchClientH3(certPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// --- Multiplexing Benchmarks ---

func benchParallel(b *testing.B, client *http.Client, addr string, scheme string) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("%s://%s/bench", scheme, addr))
			if err != nil {
				b.Error(err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

func BenchmarkH1Parallel8(b *testing.B) {
	addr, cleanup := benchH1(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	benchParallel(b, benchClientH1(addr), addr, "http")
}

func BenchmarkH2CParallel8(b *testing.B) {
	addr, cleanup := benchH2(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	benchParallel(b, benchClientH2(addr), addr, "http")
}

func BenchmarkH3Parallel8(b *testing.B) {
	addr, certPath, cleanup := benchH3(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	benchParallel(b, benchClientH3(certPath), addr, "https")
}

// --- Latency Benchmarks (single request timing) ---

func BenchmarkLatencyH1(b *testing.B) {
	addr, cleanup := benchH1(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH1(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkLatencyH2C(b *testing.B) {
	addr, cleanup := benchH2(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH2(addr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkLatencyH3(b *testing.B) {
	addr, certPath, cleanup := benchH3(b, benchMux(benchSmallBody, 0))
	defer cleanup()
	client := benchClientH3(certPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", addr))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
