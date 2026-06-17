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

	"github.com/user/gore/internal/proxy/cache"
)

type Upstream struct {
	Name               string
	Balancer           Balancer
	Proxy              *httputil.ReverseProxy
	SetHeaders         map[string]string
	MaxRetries         int
	Cache              *cache.Cache
	Redirect           string
	NextUpstream       string
	NextUpstreamTries  int
	NextUpstreamTimeout int
	PassRequestHeaders *bool
	PassRequestBody    *bool
}

type TimeoutConfig struct {
	Connect          int // seconds
	Read             int // seconds
	Send             int // seconds
	Idle             int // seconds
	Keepalive        int // max idle connections per upstream host
	KeepaliveTimeout int // seconds to keep idle connections
	KeepaliveRequests int // max requests per keepalive connection
}

func NewUpstream(name string, servers []*Server, strategy string, timeouts *TimeoutConfig, setHeaders map[string]string, buffered bool, maxRetries int, bufferSize int, redirect string, nextUpstream string, nextUpstreamTries int, nextUpstreamTimeout int, passRequestHeaders *bool, passRequestBody *bool) *Upstream {
	var balancer Balancer
	switch strategy {
	case "least-conn":
		balancer = NewLeastConn(servers)
	case "ip_hash":
		balancer = NewIPHash(servers)
	case "hash":
		balancer = NewConsistentHash(servers)
	default:
		balancer = NewRoundRobin(servers)
	}

	if timeouts == nil {
		timeouts = &TimeoutConfig{Connect: 60, Read: 60, Send: 60, Idle: 90}
	}

	u := &Upstream{
		Name:               name,
		Balancer:           balancer,
		SetHeaders:         setHeaders,
		MaxRetries:         maxRetries,
		Redirect:           redirect,
		NextUpstream:       nextUpstream,
		NextUpstreamTries:  nextUpstreamTries,
		NextUpstreamTimeout: nextUpstreamTimeout,
		PassRequestHeaders: passRequestHeaders,
		PassRequestBody:    passRequestBody,
	}
	u.Proxy = &httputil.ReverseProxy{
		Director:     u.director,
		Transport:    u.transport(timeouts),
		ErrorHandler: u.ErrorHandler,
	}
	if !buffered {
		u.Proxy.FlushInterval = -1
	}
	if redirect != "" {
		u.Proxy.ModifyResponse = u.modifyResponse
	}
	return u
}

func (u *Upstream) director(req *http.Request) {
	server := u.Balancer.Next(req)
	if server == nil {
		return
	}

	target, err := url.Parse("http://" + server.Addr)
	if err != nil {
		return
	}

	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host

	if u.PassRequestBody != nil && !*u.PassRequestBody {
		req.Body = nil
		req.ContentLength = 0
		req.Header.Del("Content-Length")
	}

	if u.PassRequestHeaders == nil || *u.PassRequestHeaders {
		for k, v := range u.SetHeaders {
			req.Header.Set(k, v)
		}
	}
}

func (u *Upstream) modifyResponse(resp *http.Response) error {
	if u.Redirect != "" {
		if loc := resp.Header.Get("Location"); loc != "" {
			resp.Header.Set("Location", rewriteRedirect(loc, u.Redirect))
		}
	}
	return nil
}

func rewriteRedirect(loc, pattern string) string {
	if pattern == "default" || pattern == "" {
		return loc
	}
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) != 2 {
		return loc
	}
	re := strings.TrimSpace(parts[0])
	repl := strings.TrimSpace(parts[1])
	return strings.Replace(loc, re, repl, 1)
}

func (u *Upstream) transport(tc *TimeoutConfig) *http.Transport {
	keepalivePerHost := tc.Keepalive
	if keepalivePerHost <= 0 {
		keepalivePerHost = 10
	}
	idleTimeout := time.Duration(tc.Idle) * time.Second
	if tc.KeepaliveTimeout > 0 {
		idleTimeout = time.Duration(tc.KeepaliveTimeout) * time.Second
	}
	return &http.Transport{
		MaxIdleConns:        keepalivePerHost * 10,
		MaxIdleConnsPerHost: keepalivePerHost,
		IdleConnTimeout:     idleTimeout,
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

	if u.Cache != nil && r.Method == http.MethodGet {
		key := u.Cache.Key(r)
		if entry, ok := u.Cache.Get(key); ok {
			for k, vv := range entry.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(entry.Status)
			w.Write(entry.Body)
			return
		}
	}

	maxAttempts := u.MaxRetries + 1
	if u.NextUpstreamTries > 0 {
		maxAttempts = u.NextUpstreamTries
	}
	flags := u.parseNextUpstreamFlags()
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if r.Body != nil && r.GetBody != nil {
				newBody, err := r.GetBody()
				if err != nil {
					break
				}
				r.Body = newBody
			}
		}

		var rec *retryResponseWriter
		if u.Cache != nil && r.Method == http.MethodGet {
			ccw := &cacheCaptureWriter{ResponseWriter: w}
			rec = &retryResponseWriter{ResponseWriter: ccw, statusCode: 200}
			u.Proxy.ServeHTTP(rec, r)
			if rec.statusCode == 200 {
				entry := &cache.Entry{
					Status: rec.statusCode,
					Header: ccw.header,
					Body:   ccw.body,
				}
				u.Cache.Set(u.Cache.Key(r), entry)
				w.Header().Set("X-Cache", "MISS")
			}
		} else {
			rec = &retryResponseWriter{ResponseWriter: w, statusCode: 200}
			u.Proxy.ServeHTTP(rec, r)
		}

		if attempt < maxAttempts-1 && shouldRetry(rec.statusCode, lastErr, flags) {
			lastErr = fmt.Errorf("upstream returned %d", rec.statusCode)
			continue
		}
		return
	}

	if lastErr != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
}

func (u *Upstream) parseNextUpstreamFlags() map[string]bool {
	flags := make(map[string]bool)
	if u.NextUpstream == "" {
		flags["error"] = true
		flags["timeout"] = true
		flags["invalid_header"] = true
		return flags
	}
	for _, f := range strings.Fields(u.NextUpstream) {
		flags[f] = true
	}
	return flags
}

func shouldRetry(statusCode int, err error, flags map[string]bool) bool {
	if err != nil {
		if flags["error"] {
			return true
		}
		return false
	}
	if statusCode >= 500 && statusCode < 600 {
		if flags["error"] {
			return true
		}
	}
	return false
}

type retryResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

type cacheCaptureWriter struct {
	http.ResponseWriter
	header http.Header
	body   []byte
	status int
}

func (ccw *cacheCaptureWriter) Header() http.Header {
	if ccw.header == nil {
		ccw.header = make(http.Header)
	}
	return ccw.header
}

func (ccw *cacheCaptureWriter) WriteHeader(code int) {
	ccw.status = code
	realH := ccw.ResponseWriter.Header()
	for k, vv := range ccw.header {
		for _, v := range vv {
			realH.Add(k, v)
		}
	}
	ccw.ResponseWriter.WriteHeader(code)
}

func (ccw *cacheCaptureWriter) Write(b []byte) (int, error) {
	ccw.body = append(ccw.body, b...)
	n, err := ccw.ResponseWriter.Write(b)
	return n, err
}

func (ccw *cacheCaptureWriter) Unwrap() http.ResponseWriter {
	return ccw.ResponseWriter
}

func (rw *retryResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *retryResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *retryResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func (u *Upstream) proxyWebSocket(w http.ResponseWriter, r *http.Request) {
	server := u.Balancer.Next(r)
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
