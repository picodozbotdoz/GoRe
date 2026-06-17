package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/gore/internal/proxy/cache"
)

// CacheConfig holds per-upstream cache directives from proxy_cache_* config fields.
type CacheConfig struct {
	Valid    map[string]time.Duration
	UseStale bool
	Lock     bool
	Key      string
	NoCache  string
	Bypass   string
}

type Upstream struct {
	Name                string
	Balancer            Balancer
	Proxy               *httputil.ReverseProxy
	SetHeaders          map[string]string
	MaxRetries          int
	Cache               *cache.Cache
	CacheConfig         *CacheConfig
	Redirect            string
	NextUpstream        string
	NextUpstreamTries   int
	NextUpstreamTimeout int
	PassRequestHeaders  *bool
	PassRequestBody     *bool
	ProxySSL            *ProxySSLConfig
	RequestBuffering    *bool
	InterceptErrors     bool
	ErrorPages          map[int]string
	CookieDomain        string
	CookiePath          string
	Method              string
	HideHeaders         []string
	SocketKeepalive     *bool
	BufferSize          int
	ProxyProtocol       bool
	MaxTempFileSize     int64
	cacheLocks          sync.Map
}

type TimeoutConfig struct {
	Connect          int // seconds
	Read             int // seconds
	Send             int // seconds
	Idle             int // seconds
	Keepalive        int // max idle connections per upstream host
	KeepaliveTimeout int // seconds to keep idle connections
	KeepaliveRequests int // max requests per keepalive connection
	Resolver         string // DNS resolver address (e.g. "8.8.8.8:53")
}

type ProxySSLConfig struct {
	Verify             bool
	Certificate        string
	CertificateKey     string
	TrustedCertificate string
	Protocols          string
	Ciphers            string
	ServerName         string
	SessionReuse       *bool
	Name               string
}

func NewUpstream(name string, servers []*Server, strategy string, timeouts *TimeoutConfig, setHeaders map[string]string, buffered bool, maxRetries int, bufferSize int, redirect string, nextUpstream string, nextUpstreamTries int, nextUpstreamTimeout int, passRequestHeaders *bool, passRequestBody *bool, requestBuffering *bool, interceptErrors bool, errorPages map[int]string, cookieDomain string, cookiePath string, method string, hideHeaders []string, socketKeepalive *bool, proxySSL *ProxySSLConfig, cacheConfig *CacheConfig, proxyProtocol bool, maxTempFileSize int64) *Upstream {
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
		RequestBuffering:   requestBuffering,
		InterceptErrors:    interceptErrors,
		ErrorPages:         errorPages,
		CookieDomain:       cookieDomain,
		CookiePath:         cookiePath,
		Method:             method,
		HideHeaders:        hideHeaders,
		SocketKeepalive:    socketKeepalive,
		BufferSize:         bufferSize,
		ProxySSL:           proxySSL,
		CacheConfig:        cacheConfig,
		ProxyProtocol:      proxyProtocol,
		MaxTempFileSize:    maxTempFileSize,
	}
	u.Proxy = &httputil.ReverseProxy{
		Director:     u.director,
		Transport:    u.transport(timeouts),
		ErrorHandler: u.ErrorHandler,
	}
	if !buffered {
		u.Proxy.FlushInterval = -1
	}
	if bufferSize > 0 {
		u.Proxy.BufferPool = newFixedBufferPool(bufferSize)
	}
	if interceptErrors || redirect != "" || len(hideHeaders) > 0 || cookieDomain != "" || cookiePath != "" {
		if u.Proxy.ModifyResponse == nil {
			u.Proxy.ModifyResponse = u.modifyResponse
		}
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

	if u.Method != "" {
		req.Method = u.Method
	}

	if u.PassRequestBody != nil && !*u.PassRequestBody {
		req.Body = nil
		req.ContentLength = 0
		req.Header.Del("Content-Length")
	}

	if u.RequestBuffering != nil && *u.RequestBuffering && req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			if u.MaxTempFileSize > 0 && int64(len(body)) > u.MaxTempFileSize {
				http.Error(nil, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
				return
			}
			req.Body = io.NopCloser(strings.NewReader(string(body)))
			req.ContentLength = int64(len(body))
		}
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

	if u.InterceptErrors {
		if u.ErrorPages != nil {
			if replacementPath, ok := u.ErrorPages[resp.StatusCode]; ok {
				resp.Header.Set("X-Error-Page", replacementPath)
				resp.Header.Set("X-Error-Status", fmt.Sprintf("%d", resp.StatusCode))
			}
		}
	}

	if len(u.HideHeaders) > 0 {
		for _, h := range u.HideHeaders {
			resp.Header.Del(h)
		}
	}

	if u.CookieDomain != "" || u.CookiePath != "" {
		rewriteCookies(resp, u.CookieDomain, u.CookiePath)
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

func rewriteCookies(resp *http.Response, domain, path string) {
	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, c := range cookies {
		if domain != "" {
			c.Domain = domain
		}
		if path != "" {
			c.Path = path
		}
		resp.Header.Add("Set-Cookie", c.String())
	}
}

type fixedBufferPool struct {
	size int
}

func newFixedBufferPool(size int) *fixedBufferPool {
	return &fixedBufferPool{size: size}
}

func (p *fixedBufferPool) Get() []byte {
	return make([]byte, p.size)
}

func (p *fixedBufferPool) Put(b []byte) {
	// no-op; let GC handle it
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

	keepAliveDuration := 30 * time.Second
	if u.SocketKeepalive != nil && !*u.SocketKeepalive {
		keepAliveDuration = 0
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: false,
	}

	if u.ProxySSL != nil {
		if u.ProxySSL.Verify {
			caCert, err := os.ReadFile(u.ProxySSL.TrustedCertificate)
			if err == nil {
				caPool := x509.NewCertPool()
				caPool.AppendCertsFromPEM(caCert)
				tlsCfg.RootCAs = caPool
			}
		}
		if u.ProxySSL.Certificate != "" {
			cert, err := tls.LoadX509KeyPair(u.ProxySSL.Certificate, u.ProxySSL.CertificateKey)
			if err == nil {
				tlsCfg.Certificates = []tls.Certificate{cert}
			}
		}
		if u.ProxySSL.ServerName != "" {
			tlsCfg.ServerName = u.ProxySSL.ServerName
		}
		if u.ProxySSL.Name != "" {
			tlsCfg.ServerName = u.ProxySSL.Name
		}
		if u.ProxySSL.SessionReuse != nil && !*u.ProxySSL.SessionReuse {
			tlsCfg.SessionTicketsDisabled = true
		}
	}

	dialer := &net.Dialer{
		Timeout:   time.Duration(tc.Connect) * time.Second,
		KeepAlive: keepAliveDuration,
	}

	var resolver *net.Resolver
	if tc.Resolver != "" {
		resolverAddr := tc.Resolver
		if !strings.Contains(resolverAddr, ":") {
			resolverAddr = resolverAddr + ":53"
		}
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", resolverAddr)
			},
		}
		dialer.Resolver = resolver
	}

	dialContext := dialer.DialContext

	if u.ProxyProtocol {
		origDialContext := dialContext
		dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := origDialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			host, portStr, _ := net.SplitHostPort(addr)
			port := 0
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
			ip := net.ParseIP(host)
			var header []byte
			if ip4 := ip.To4(); ip4 != nil {
				header = []byte(fmt.Sprintf("PROXY TCP4 %s %s %d %d\r\n", "0.0.0.0", host, 0, port))
			} else {
				header = []byte(fmt.Sprintf("PROXY TCP6 %s %s %d %d\r\n", "::", host, 0, port))
			}
			if _, err := conn.Write(header); err != nil {
				conn.Close()
				return nil, err
			}
			return conn, nil
		}
	}

	return &http.Transport{
		MaxIdleConns:        keepalivePerHost * 10,
		MaxIdleConnsPerHost: keepalivePerHost,
		IdleConnTimeout:     idleTimeout,
		DialContext:         dialContext,
		ResponseHeaderTimeout: time.Duration(tc.Read) * time.Second,
		TLSClientConfig:       tlsCfg,
		ForceAttemptHTTP2:     true,
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

	cc := u.CacheConfig
	if u.Cache != nil && r.Method == http.MethodGet {
		key := u.Cache.Key(r)
		if cc != nil && cc.Key != "" {
			key = cc.Key + " " + key
		}

		if cc != nil && cc.NoCache != "" {
			if r.Header.Get(cc.NoCache) != "" {
				u.proxyAndReturn(w, r)
				return
			}
		}
		if cc != nil && cc.Bypass != "" {
			if r.Header.Get(cc.Bypass) != "" {
				u.proxyAndReturn(w, r)
				return
			}
		}

		if cc != nil && cc.Lock {
			if _, loaded := u.cacheLocks.LoadOrStore(key, true); loaded {
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
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			defer u.cacheLocks.Delete(key)
		}

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

		ccw := &cacheCaptureWriter{ResponseWriter: w}
		rec := &retryResponseWriter{ResponseWriter: ccw, statusCode: 200}
		u.Proxy.ServeHTTP(rec, r)

		if rec.statusCode >= 200 && rec.statusCode < 400 {
			entry := &cache.Entry{
				Status: rec.statusCode,
				Header: ccw.header,
				Body:   ccw.body,
			}
			if cc != nil && cc.Valid != nil {
				statusKey := fmt.Sprintf("%d", rec.statusCode)
				if ttl, ok := cc.Valid[statusKey]; ok {
					entry.TTL = ttl
				}
			}
			u.Cache.Set(key, entry)
		} else if rec.statusCode >= 500 && cc != nil && cc.UseStale {
			if entry, ok := u.Cache.GetStale(key); ok {
				for k, vv := range entry.Header {
					for _, v := range vv {
						w.Header().Add(k, v)
					}
				}
				w.Header().Set("X-Cache", "STALE")
				w.WriteHeader(entry.Status)
				w.Write(entry.Body)
				return
			}
		}
		w.Header().Set("X-Cache", "MISS")
		return
	}

	u.proxyAndReturn(w, r)
}

func (u *Upstream) proxyAndReturn(w http.ResponseWriter, r *http.Request) {
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

		rec := &retryResponseWriter{ResponseWriter: w, statusCode: 200}
		u.Proxy.ServeHTTP(rec, r)

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
