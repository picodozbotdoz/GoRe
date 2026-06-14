package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

const (
	nginxH1   = "127.0.0.1:8180"
	nginxH2C  = "127.0.0.1:8181"
	nginxH2TL = "127.0.0.1:8182"
	nginxH3   = "127.0.0.1:8183"
)

var nginxCertPath = "/tmp/nginx-bench/cert.pem"

func nginxH1Client() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
}

func nginxH2CClient() *http.Client {
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

func nginxH2TLSClient() *http.Client {
	ca, _ := os.ReadFile(nginxCertPath)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	return &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, NextProtos: []string{"h2"}},
		},
		Timeout: 10 * time.Second,
	}
}

func nginxH3Client() *http.Client {
	ca, _ := os.ReadFile(nginxCertPath)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	return &http.Client{
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, NextProtos: []string{"h3"}},
		},
		Timeout: 10 * time.Second,
	}
}

func nginxSkipH3(b *testing.B) bool {
	client := nginxH3Client()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/bench", nginxH3), nil)
	_, err := client.Do(req)
	if err != nil {
		b.Logf("HTTP/3 not available, skipping: %v", err)
		return true
	}
	return false
}

// --- Nginx HTTP/1.1 ---

func BenchmarkNginxH1Small(b *testing.B) {
	client := nginxH1Client()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", nginxH1))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkNginxH1Parallel(b *testing.B) {
	client := nginxH1Client()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("http://%s/bench", nginxH1))
			if err != nil {
				b.Error(err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// --- Nginx HTTP/2 (h2c) ---

func BenchmarkNginxH2CSmall(b *testing.B) {
	client := nginxH2CClient()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("http://%s/bench", nginxH2C))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkNginxH2CParallel(b *testing.B) {
	client := nginxH2CClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("http://%s/bench", nginxH2C))
			if err != nil {
				b.Error(err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// --- Nginx HTTP/2 TLS ---

func BenchmarkNginxH2TLSSmall(b *testing.B) {
	client := nginxH2TLSClient()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", nginxH2TL))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkNginxH2TLSParallel(b *testing.B) {
	client := nginxH2TLSClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("https://%s/bench", nginxH2TL))
			if err != nil {
				b.Error(err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// --- Nginx HTTP/3 ---

func BenchmarkNginxH3Small(b *testing.B) {
	if nginxSkipH3(b) {
		return
	}
	client := nginxH3Client()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/bench", nginxH3))
		if err != nil {
			b.Skipf("H3 error: %v", err)
			return
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkNginxH3Parallel(b *testing.B) {
	if nginxSkipH3(b) {
		return
	}
	client := nginxH3Client()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("https://%s/bench", nginxH3))
			if err != nil {
				b.Error(err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}
