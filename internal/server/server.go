package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/crypto/ocsp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/user/gore/internal/config"
	gorelog "github.com/user/gore/internal/log"
	"github.com/user/gore/internal/modules"
	"github.com/user/gore/internal/modules/authrequest"
	"github.com/user/gore/internal/modules/mirror"
	"github.com/user/gore/internal/modules/static"
	"github.com/user/gore/internal/modules/status"
	"github.com/user/gore/internal/modules/subfilter"
	"github.com/user/gore/internal/proxy"
	"github.com/user/gore/internal/router"
)

type Server struct {
	cfg       *config.Config
	router    *router.Router
	servers   []*http.Server
	http3Srvs []*http3.Server
	upstreams map[string]*proxy.Upstream
	addrs     []string
	addrMu    sync.Mutex
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:       cfg,
		router:    router.NewRouter(),
		upstreams: make(map[string]*proxy.Upstream),
	}
	s.initUpstreams()
	s.initRoutes()
	return s
}

func (s *Server) initUpstreams() {
	for name, upstreamCfg := range s.cfg.Upstreams {
		servers := make([]*proxy.Server, len(upstreamCfg.Servers))
		for i, srv := range upstreamCfg.Servers {
			servers[i] = &proxy.Server{
				Addr:      srv.Addr,
				Weight:    srv.Weight,
				Backup:    srv.Backup,
				Down:      srv.Down,
				SlowStart: srv.SlowStart,
			}
			if servers[i].Weight == 0 {
				servers[i].Weight = 1
			}
			servers[i].FullWeight = servers[i].Weight
		}
		var proxySSL *proxy.ProxySSLConfig
		if upstreamCfg.ProxySSL != nil {
			proxySSL = &proxy.ProxySSLConfig{
				Verify:             upstreamCfg.ProxySSL.Verify,
				Certificate:        upstreamCfg.ProxySSL.Certificate,
				CertificateKey:     upstreamCfg.ProxySSL.CertificateKey,
				TrustedCertificate: upstreamCfg.ProxySSL.TrustedCertificate,
				Protocols:          upstreamCfg.ProxySSL.Protocols,
				Ciphers:            upstreamCfg.ProxySSL.Ciphers,
				ServerName:         upstreamCfg.ProxySSL.ServerName,
				SessionReuse:       upstreamCfg.ProxySSL.SessionReuse,
				Name:               upstreamCfg.ProxySSL.Name,
			}
		}

		var cacheCfg *proxy.CacheConfig
		if upstreamCfg.Cache != nil {
			cacheCfg = &proxy.CacheConfig{
				UseStale: upstreamCfg.Cache.Enabled,
			}
		}

		s.upstreams[name] = proxy.NewUpstream(name, servers, upstreamCfg.Strategy, &proxy.TimeoutConfig{
			Connect:           upstreamCfg.GetConnectTimeout(),
			Read:              upstreamCfg.GetReadTimeout(),
			Send:              upstreamCfg.GetSendTimeout(),
			Idle:              upstreamCfg.GetIdleTimeout(),
			Keepalive:         upstreamCfg.Keepalive,
			KeepaliveTimeout:  upstreamCfg.KeepaliveTimeout,
			KeepaliveRequests: upstreamCfg.KeepaliveRequests,
			Resolver:          s.cfg.Modules.Resolver,
		}, upstreamCfg.SetHeaders, upstreamCfg.GetBuffering(), upstreamCfg.GetRetries(), 0, "", upstreamCfg.NextUpstream, upstreamCfg.NextUpstreamTries, upstreamCfg.NextUpstreamTimeout, nil, nil, nil, false, nil, "", "", "", nil, nil, proxySSL, cacheCfg, upstreamCfg.ProxyProtocol, config.ParseSize(upstreamCfg.MaxTempFileSize), upstreamCfg.Resolve, upstreamCfg.Zone)
		if upstreamCfg.HealthCheck != nil && upstreamCfg.HealthCheck.Enabled {
			proxy.StartHealthCheck(servers, upstreamCfg.HealthCheck.GetInterval(), upstreamCfg.HealthCheck.Path)
		}
	}
}

func (s *Server) initRoutes() {
	for _, httpCfg := range s.cfg.HTTP.Servers {
		for _, loc := range httpCfg.Locations {
			s.router.AddRoute(loc.Path, s.buildLocationHandler(loc))
		}
	}

	if s.cfg.Modules.Status != nil && s.cfg.Modules.Status.Enabled {
		s.router.AddRoute(s.cfg.Modules.Status.GetPath(), status.NewHandler(s.cfg.Modules.Status.GetPath()))
		gorelog.SetRequestTracker(status.Get().ReqStart, status.Get().ReqDone)
	}
}

func resolveVariable(r *http.Request, variable string) string {
	variable = strings.TrimPrefix(variable, "$")
	switch {
	case strings.HasPrefix(variable, "http_"):
		headerName := strings.TrimPrefix(variable, "http_")
		headerName = strings.ReplaceAll(headerName, "_", "-")
		return r.Header.Get(headerName)
	case variable == "remote_addr":
		return r.RemoteAddr
	case variable == "host":
		return r.Host
	case variable == "request_uri":
		return r.URL.RequestURI()
	case variable == "method":
		return r.Method
	default:
		return r.Header.Get(variable)
	}
}

func (s *Server) buildLocationHandler(loc config.Location) http.Handler {
	var handler http.Handler
	if loc.Proxy != nil {
		if dynUpstream := loc.Proxy.GetDynamicUpstream(); dynUpstream != "" {
			upstreams := s.upstreams
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				value := resolveVariable(r, dynUpstream)
				if upstream, ok := upstreams[value]; ok {
					upstream.ServeHTTP(w, r)
					return
				}
				if upstream, ok := upstreams[loc.Proxy.Upstream]; ok {
					upstream.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
			})
		} else if upstream, ok := s.upstreams[loc.Proxy.Upstream]; ok {
			handler = upstream
		}
	} else if loc.Alias != "" {
		aliasDir := loc.Alias
		prefix := loc.Path
		autoindex := false
		if loc.Autoindex != nil {
			autoindex = *loc.Autoindex
		}
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri := r.URL.Path[len(prefix):]
			if uri == "" {
				uri = "/"
			}
			r2 := r.Clone(r.Context())
			r2.URL.Path = uri
			static.NewWithCache(aliasDir, autoindex, loc.CacheControl).ServeHTTP(w, r2)
		})
	} else if loc.Root != "" && len(loc.TryFiles) > 0 {
		autoindex := false
		if loc.Autoindex != nil {
			autoindex = *loc.Autoindex
		}
		prefix := loc.Path
		root := loc.Root
		files := loc.TryFiles
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri := r.URL.Path[len(prefix):]
			if uri == "" {
				uri = "/"
			}
			for i, f := range files {
				if strings.HasPrefix(f, "=") {
					code, _ := strconv.Atoi(f[1:])
					http.Error(w, http.StatusText(code), code)
					return
				}
				candidate := strings.Replace(f, "$uri", uri, 1)
				fullPath := filepath.Join(root, candidate)
				if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
					r2 := r.Clone(r.Context())
					r2.URL.Path = candidate
					static.NewWithCache(root, autoindex, loc.CacheControl).ServeHTTP(w, r2)
					return
				}
				if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
					indexPath := filepath.Join(fullPath, "index.html")
					if _, err := os.Stat(indexPath); err == nil {
						r2 := r.Clone(r.Context())
						r2.URL.Path = candidate + "/index.html"
						static.NewWithCache(root, autoindex, loc.CacheControl).ServeHTTP(w, r2)
						return
					}
				}
				if i == len(files)-1 {
					if strings.HasPrefix(f, "/") {
						http.Redirect(w, r, f, http.StatusFound)
					} else {
						http.NotFound(w, r)
					}
					return
				}
			}
			http.NotFound(w, r)
		})
	} else if loc.Root != "" {
		autoindex := false
		if loc.Autoindex != nil {
			autoindex = *loc.Autoindex
		}
		prefix := loc.Path
		rootHandler := static.NewWithCache(loc.Root, autoindex, loc.CacheControl)
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = r.URL.Path[len(prefix):]
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			rootHandler.ServeHTTP(w, r)
		})
	} else if loc.Rewrite != nil {
		pat := regexp.MustCompile(loc.Rewrite.Pattern)
		repl := loc.Rewrite.Replacement
		code := loc.Rewrite.Code
		logRewrite := loc.Rewrite.Log
		breakAfter := loc.Rewrite.Break
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newPath := pat.ReplaceAllString(r.URL.Path, repl)
			if logRewrite {
				gorelog.Infof("rewrite %s -> %s", r.URL.Path, newPath)
			}
			if code >= 300 && code < 400 {
				http.Redirect(w, r, newPath, code)
				return
			}
			r.URL.Path = newPath
			if breakAfter {
				return
			}
			http.NotFound(w, r)
		})
	} else if loc.Return != "" {
		returnSpec := loc.Return
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.SplitN(returnSpec, " ", 2)
			if code, err := strconv.Atoi(parts[0]); err == nil {
				body := ""
				if len(parts) > 1 {
					body = strings.Trim(parts[1], "'\"")
				}
				if code >= 300 && code < 400 {
					if body != "" {
						w.Header().Set("Location", body)
					}
					w.WriteHeader(code)
					return
				}
				w.WriteHeader(code)
				if body != "" {
					w.Write([]byte(body))
				}
				return
			}
			http.Redirect(w, r, returnSpec, http.StatusMovedPermanently)
		})
	}
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	if len(loc.LimitExcept) > 0 {
		allowed := make(map[string]bool, len(loc.LimitExcept))
		for _, m := range loc.LimitExcept {
			allowed[m] = true
		}
		nextHandler := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[r.Method] {
				w.Header().Set("Allow", strings.Join(loc.LimitExcept, ", "))
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}
			nextHandler.ServeHTTP(w, r)
		})
	}
	if loc.Internal {
		nextHandler := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Internal") == "" {
				http.NotFound(w, r)
				return
			}
			nextHandler.ServeHTTP(w, r)
		})
	}
	if loc.Satisfy == "any" || loc.Satisfy == "all" {
		satisfyMode := loc.Satisfy
		nextHandler := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if satisfyMode == "any" || satisfyMode == "all" {
				nextHandler.ServeHTTP(w, r)
				return
			}
			nextHandler.ServeHTTP(w, r)
		})
	}
	if loc.AuthRequest != "" {
		handler = authrequest.New(loc.AuthRequest)(handler)
	}
	if len(loc.AuthRequestSet) > 0 {
		nextHandler := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for authHeader, reqHeader := range loc.AuthRequestSet {
				if val := r.Header.Get(authHeader); val != "" {
					r.Header.Set(reqHeader, val)
				}
			}
			nextHandler.ServeHTTP(w, r)
		})
	}
	if len(loc.SubFilter) > 0 {
		once := loc.SubFilterOnce != nil && *loc.SubFilterOnce
		handler = subfilter.New(loc.SubFilter, once, loc.SubFilterTypes)(handler)
	}
	if loc.Mirror != "" {
		handler = mirror.New(loc.Mirror)(handler)
	}
	return modules.BuildChain(&s.cfg.Modules, handler)
}

func parseECDHCurves(names string) []tls.CurveID {
	if names == "" {
		return nil
	}
	nameMap := map[string]tls.CurveID{
		"X25519": tls.X25519,
		"P-256":  tls.CurveP256,
		"P-384":  tls.CurveP384,
		"P-521":  tls.CurveP521,
	}
	var curves []tls.CurveID
	for _, name := range strings.Split(names, ":") {
		name = strings.TrimSpace(name)
		if id, ok := nameMap[name]; ok {
			curves = append(curves, id)
		}
	}
	return curves
}

type ocspStapler struct {
	mu       sync.RWMutex
	response []byte
	rawCert  []byte
	rawIssuer *x509.Certificate
	cert     tls.Certificate
	verify   bool
}

func newOCSPStapler(cert tls.Certificate, verify bool) (*ocspStapler, error) {
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse leaf cert: %w", err)
	}

	var issuer *x509.Certificate
	if len(cert.Certificate) > 1 {
		issuer, err = x509.ParseCertificate(cert.Certificate[1])
		if err != nil {
			return nil, fmt.Errorf("parse issuer cert: %w", err)
		}
	} else {
		roots := x509.NewCertPool()
		if !verify {
			roots.AddCert(leaf)
		}
		candidates, err := leaf.Verify(x509.VerifyOptions{
			Roots:     roots,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		if err == nil && len(candidates) > 0 && len(candidates[0]) > 1 {
			issuer = candidates[0][1]
		}
	}

	if issuer == nil {
		return nil, fmt.Errorf("cannot determine issuer certificate")
	}

	s := &ocspStapler{
		rawCert:  cert.Certificate[0],
		rawIssuer: issuer,
		cert:     cert,
		verify:   verify,
	}

	s.refresh()

	return s, nil
}

func (s *ocspStapler) refresh() {
	leaf, err := x509.ParseCertificate(s.rawCert)
	if err != nil {
		gorelog.Infof("OCSP: failed to parse leaf cert: %v", err)
		return
	}

	if len(leaf.OCSPServer) == 0 {
		gorelog.Infof("OCSP: no OCSP responder URLs in certificate")
		return
	}

	ocspReq, err := ocsp.CreateRequest(leaf, s.rawIssuer, &ocsp.RequestOptions{})
	if err != nil {
		gorelog.Infof("OCSP: failed to create request: %v", err)
		return
	}

	var rawResponse []byte
	for _, ocspURL := range leaf.OCSPServer {
		rawResponse, err = s.fetchOCSP(ocspURL, ocspReq)
		if err != nil {
			gorelog.Infof("OCSP: request to %s failed: %v", ocspURL, err)
			continue
		}
		break
	}
	if err != nil {
		gorelog.Infof("OCSP: all responders failed, stapling unavailable")
		return
	}

	resp, err := ocsp.ParseResponseForCert(rawResponse, leaf, s.rawIssuer)
	if err != nil {
		gorelog.Infof("OCSP: invalid response: %v", err)
		return
	}

	if resp.Status != ocsp.Good && resp.Status != ocsp.Revoked {
		gorelog.Infof("OCSP: unexpected status %d", resp.Status)
		return
	}

	s.mu.Lock()
	s.response = rawResponse
	s.mu.Unlock()

	gorelog.Infof("OCSP: stapled response cached (status=%d, expiry=%s)", resp.Status, resp.NextUpdate.Format(time.RFC3339))
}

func (s *ocspStapler) fetchOCSP(url string, req []byte) ([]byte, error) {
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(req))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/ocsp-request")
	httpReq.Header.Set("Accept", "application/ocsp-response")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (s *ocspStapler) getStapledResponse(cert *x509.Certificate) *tls.Certificate {
	s.mu.RLock()
	response := s.response
	s.mu.RUnlock()

	if len(response) == 0 {
		return nil
	}

	c := tls.Certificate{
		Certificate: s.cert.Certificate,
		PrivateKey:  s.cert.PrivateKey,
		Leaf:        s.cert.Leaf,
	}
	c.OCSPStaple = response
	return &c
}

func (s *Server) buildTLSConfig(listen *config.Listen) *tls.Config {
	cert, err := tls.LoadX509KeyPair(listen.TLS.Cert, listen.TLS.Key)
	if err != nil {
		return nil
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}
	if len(listen.TLS.Ciphers) > 0 {
		supported := tls.CipherSuites()
		nameToID := make(map[string]uint16, len(supported))
		for _, cs := range supported {
			nameToID[cs.Name] = cs.ID
		}
		var ciphers []uint16
		for _, name := range listen.TLS.Ciphers {
			if id, ok := nameToID[name]; ok {
				ciphers = append(ciphers, id)
			}
		}
		if len(ciphers) > 0 {
			tlsConfig.CipherSuites = ciphers
		}
	}
	if listen.TLS.MinVersion != "" {
		switch listen.TLS.MinVersion {
		case "1.0":
			tlsConfig.MinVersion = tls.VersionTLS10
		case "1.1":
			tlsConfig.MinVersion = tls.VersionTLS11
		case "1.2":
			tlsConfig.MinVersion = tls.VersionTLS12
		case "1.3":
			tlsConfig.MinVersion = tls.VersionTLS13
		}
	}
	if listen.TLS.SessionTimeout > 0 {
		tlsConfig.ClientSessionCache = &timedSessionCache{
			cache:   tls.NewLRUClientSessionCache(128),
			timeout: time.Duration(listen.TLS.GetSessionTimeout()) * time.Second,
		}
	}
	if listen.TLS.ECDHCurve != "" {
		if curves := parseECDHCurves(listen.TLS.ECDHCurve); len(curves) > 0 {
			tlsConfig.CurvePreferences = curves
		}
	}
	if listen.TLS.ClientCertificate != "" {
		caCert, err := os.ReadFile(listen.TLS.ClientCertificate)
		if err == nil {
			caPool := x509.NewCertPool()
			caPool.AppendCertsFromPEM(caCert)
			tlsConfig.ClientCAs = caPool
		}
		if listen.TLS.VerifyClient {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}
	if listen.TLS.RejectHandshake {
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			return fmt.Errorf("tls handshake rejected by configuration")
		}
	}
	if listen.TLS.Stapling {
		stapler, err := newOCSPStapler(cert, listen.TLS.StaplingVerify)
		if err != nil {
			gorelog.Infof("OCSP stapling init failed: %v (stapling disabled)", err)
		} else {
		tlsConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			for _, c := range tlsConfig.Certificates {
				leaf, parseErr := x509.ParseCertificate(c.Certificate[0])
				if parseErr != nil {
					continue
				}
				tlsLeaf := &tls.Certificate{
					Certificate: c.Certificate,
					Leaf:        leaf,
				}
				if err := clientHello.SupportsCertificate(tlsLeaf); err == nil {
					stapled := stapler.getStapledResponse(leaf)
					if stapled != nil {
						return stapled, nil
					}
					break
				}
			}
			return &tlsConfig.Certificates[0], nil
		}
		}
	}
	return tlsConfig
}

func (s *Server) buildHTTP2Config(listen *config.Listen) *http2.Server {
	return &http2.Server{
		MaxConcurrentStreams: uint32(listen.HTTP2.GetMaxConcurrentStreams()),
		MaxReadFrameSize:     uint32(listen.HTTP2.GetMaxFrameSize()),
	}
}

func (s *Server) Start() error {
	var wg sync.WaitGroup
	for i := range s.cfg.Listen {
		listen := &s.cfg.Listen[i]

		ln, err := net.Listen("tcp", listen.Addr)
		if err != nil {
			return err
		}
		addr := ln.Addr().String()
		s.addrMu.Lock()
		s.addrs = append(s.addrs, addr)
		s.addrMu.Unlock()

		h2cfg := s.buildHTTP2Config(listen)

		idleTimeout := 120
		if listen.KeepAliveTimeout > 0 {
			idleTimeout = listen.KeepAliveTimeout
		}
		readTimeout := 30
		if s.cfg.Modules.ClientHeaderTimeout > 0 {
			readTimeout = s.cfg.Modules.ClientHeaderTimeout
		}
		writeTimeout := 30
		if s.cfg.Modules.SendTimeout > 0 {
			writeTimeout = s.cfg.Modules.SendTimeout
		}
		srv := &http.Server{
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
			IdleTimeout:  time.Duration(idleTimeout) * time.Second,
		}

		if listen.TLS != nil {
			tlsConfig := s.buildTLSConfig(listen)
			if tlsConfig == nil {
				ln.Close()
				return fmt.Errorf("failed to load TLS certificate")
			}
			srv.TLSConfig = tlsConfig

			if err := http2.ConfigureServer(srv, h2cfg); err != nil {
				ln.Close()
				return err
			}

			srv.Handler = s.addAltSvcHeader(s.router, listen)
			s.servers = append(s.servers, srv)
			wg.Add(1)
			go func(srv *http.Server, ln net.Listener, addr string) {
				defer wg.Done()
				gorelog.Infof("listening on %s (HTTP/2 + TLS)", addr)
				tlsLn := tls.NewListener(ln, srv.TLSConfig)
				if err := srv.Serve(tlsLn); err != nil && err != http.ErrServerClosed {
					gorelog.Infof("server error: %v", err)
				}
			}(srv, ln, addr)

			if listen.HTTP3 != nil && listen.HTTP3.Enabled != nil && *listen.HTTP3.Enabled {
				h3Srv := &http3.Server{
					Addr:      addr,
					TLSConfig: tlsConfig,
					Handler:   s.router,
				}
				s.http3Srvs = append(s.http3Srvs, h3Srv)

				udpAddr := ln.Addr().String()
				udpLn, err := net.ListenPacket("udp", udpAddr)
				if err != nil {
					gorelog.Infof("HTTP/3 UDP listen error on %s: %v (HTTP/3 disabled)", udpAddr, err)
				} else {
					wg.Add(1)
					go func(h3 *http3.Server, pc net.PacketConn) {
						defer wg.Done()
						gorelog.Infof("listening on %s (HTTP/3 QUIC)", pc.LocalAddr())
						if err := h3.Serve(pc); err != nil && err != http.ErrServerClosed {
							gorelog.Infof("HTTP/3 server error: %v", err)
						}
					}(h3Srv, udpLn)
				}
			}
		} else {
			h2Handler := h2c.NewHandler(s.router, h2cfg)
			srv.Handler = h2Handler

			s.servers = append(s.servers, srv)
			wg.Add(1)
			go func(srv *http.Server, ln net.Listener, addr string) {
				defer wg.Done()
				gorelog.Infof("listening on %s (HTTP/2 cleartext + HTTP/1.1)", addr)
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					gorelog.Infof("server error: %v", err)
				}
			}(srv, ln, addr)
		}
	}
	wg.Wait()
	return nil
}

func (s *Server) Addr(i int) string {
	s.addrMu.Lock()
	defer s.addrMu.Unlock()
	if i < len(s.addrs) {
		return s.addrs[i]
	}
	return ""
}

func (s *Server) Stop(ctx context.Context) error {
	for _, h3 := range s.http3Srvs {
		h3.Shutdown(ctx)
	}
	for _, srv := range s.servers {
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) addAltSvcHeader(handler http.Handler, listen *config.Listen) http.Handler {
	if listen.HTTP3 == nil || listen.HTTP3.Enabled == nil || !*listen.HTTP3.Enabled {
		return handler
	}
	_, port, _ := net.SplitHostPort(listen.Addr)
	if port == "" {
		port = "443"
	}
	altSvc := fmt.Sprintf(`h3=":%s"; ma=86400`, port)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Alt-Svc", altSvc)
		handler.ServeHTTP(w, r)
	})
}

type timedSessionCache struct {
	cache   tls.ClientSessionCache
	timeout time.Duration
}

func (c *timedSessionCache) Get(sessionKey string) (*tls.ClientSessionState, bool) {
	return c.cache.Get(sessionKey)
}

func (c *timedSessionCache) Put(sessionKey string, cs *tls.ClientSessionState) {
	c.cache.Put(sessionKey, cs)
}
