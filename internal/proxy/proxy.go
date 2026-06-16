package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Upstream struct {
	Name       string
	Balancer   Balancer
	Proxy      *httputil.ReverseProxy
	SetHeaders map[string]string
}

type TimeoutConfig struct {
	Connect   int // seconds
	Read      int // seconds
	Send      int // seconds
	Idle      int // seconds
	Keepalive int // max idle connections per upstream host
}

func NewUpstream(name string, servers []*Server, strategy string, timeouts *TimeoutConfig, setHeaders map[string]string, buffered bool) *Upstream {
	var balancer Balancer
	switch strategy {
	case "least-conn":
		balancer = NewRoundRobin(servers)
	default:
		balancer = NewRoundRobin(servers)
	}

	if timeouts == nil {
		timeouts = &TimeoutConfig{Connect: 60, Read: 60, Send: 60, Idle: 90}
	}

	u := &Upstream{Name: name, Balancer: balancer, SetHeaders: setHeaders}
	u.Proxy = &httputil.ReverseProxy{
		Director:     u.director,
		Transport:    u.transport(timeouts),
		ErrorHandler: u.ErrorHandler,
	}
	if !buffered {
		u.Proxy.FlushInterval = -1
	}
	return u
}

func (u *Upstream) director(req *http.Request) {
	server := u.Balancer.Next()
	if server == nil {
		return
	}

	target, err := url.Parse("http://" + server.Addr)
	if err != nil {
		return
	}

	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host

	for k, v := range u.SetHeaders {
		req.Header.Set(k, v)
	}
}

func (u *Upstream) transport(tc *TimeoutConfig) *http.Transport {
	keepalivePerHost := tc.Keepalive
	if keepalivePerHost <= 0 {
		keepalivePerHost = 10
	}
	return &http.Transport{
		MaxIdleConns:        keepalivePerHost * 10,
		MaxIdleConnsPerHost: keepalivePerHost,
		IdleConnTimeout:     time.Duration(tc.Idle) * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(tc.Connect) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: time.Duration(tc.Read) * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		ForceAttemptHTTP2: true,
	}
}

func (u *Upstream) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Printf("upstream error: %v\n", err)
	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

func (u *Upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isWebSocketUpgrade(r) {
		u.proxyWebSocket(w, r)
		return
	}
	u.Proxy.ServeHTTP(w, r)
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func (u *Upstream) proxyWebSocket(w http.ResponseWriter, r *http.Request) {
	server := u.Balancer.Next()
	if server == nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		return
	}

	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		fmt.Printf("websocket hijack error: %v\n", err)
		return
	}
	defer clientConn.Close()

	backendAddr := server.Addr
	backendConn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		fmt.Printf("websocket backend dial error: %v\n", err)
		return
	}
	defer backendConn.Close()

	// Forward the original HTTP request to backend
	r.URL.Scheme = "http"
	r.URL.Host = backendAddr
	r.Header.Del("Connection")
	r.Header.Del("Upgrade")
	if err := r.Write(backendConn); err != nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(backendConn, io.MultiReader(clientBuf, clientConn))
	}()

	go func() {
		defer wg.Done()
		io.Copy(clientConn, backendConn)
	}()

	wg.Wait()
}
