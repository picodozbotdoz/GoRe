package server

import (
	"context"
	"crypto/tls"
	"fmt"
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
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/user/gore/internal/config"
	gorelog 	"github.com/user/gore/internal/log"
	"github.com/user/gore/internal/modules"
	"github.com/user/gore/internal/modules/authrequest"
	"github.com/user/gore/internal/modules/static"
	"github.com/user/gore/internal/modules/status"
	"github.com/user/gore/internal/proxy"
	"github.com/user/gore/internal/router"
)

type Server struct {
	cfg        *config.Config
	router     *router.Router
	servers    []*http.Server
	http3Srvs  []*http3.Server
	upstreams  map[string]*proxy.Upstream
	addrs      []string
	addrMu     sync.Mutex
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
			servers[i] = &proxy.Server{Addr: srv.Addr, Weight: srv.Weight}
		}
		s.upstreams[name] = proxy.NewUpstream(name, servers, upstreamCfg.Strategy, &proxy.TimeoutConfig{
			Connect:   upstreamCfg.GetConnectTimeout(),
			Read:      upstreamCfg.GetReadTimeout(),
			Send:      upstreamCfg.GetSendTimeout(),
			Idle:      upstreamCfg.GetIdleTimeout(),
			Keepalive: upstreamCfg.Keepalive,
		}, upstreamCfg.SetHeaders, upstreamCfg.GetBuffering(), upstreamCfg.GetRetries())
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

func (s *Server) buildLocationHandler(loc config.Location) http.Handler {
	var handler http.Handler
	if loc.Proxy != nil {
		if upstream, ok := s.upstreams[loc.Proxy.Upstream]; ok {
			handler = upstream
		}
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
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newPath := pat.ReplaceAllString(r.URL.Path, repl)
			if code >= 300 && code < 400 {
				http.Redirect(w, r, newPath, code)
				return
			}
			r.URL.Path = newPath
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
	if loc.AuthRequest != "" {
		handler = authrequest.New(loc.AuthRequest)(handler)
	}
	return modules.BuildChain(&s.cfg.Modules, handler)
}

func (s *Server) buildHTTP2Config(listen *config.Listen) *http2.Server {
	return &http2.Server{
		MaxConcurrentStreams: uint32(listen.HTTP2.GetMaxConcurrentStreams()),
		MaxReadFrameSize:    uint32(listen.HTTP2.GetMaxFrameSize()),
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

		srv := &http.Server{
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		if listen.TLS != nil {
			cert, err := tls.LoadX509KeyPair(listen.TLS.Cert, listen.TLS.Key)
			if err != nil {
				ln.Close()
				return err
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
