package server

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/user/gore/internal/config"
	"github.com/user/gore/internal/modules"
	"github.com/user/gore/internal/proxy"
	"github.com/user/gore/internal/router"
	"github.com/user/gore/internal/modules/static"
)

type Server struct {
	cfg       *config.Config
	router    *router.Router
	servers   []*http.Server
	upstreams map[string]*proxy.Upstream
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
		s.upstreams[name] = proxy.NewUpstream(name, servers, upstreamCfg.Strategy)
	}
}

func (s *Server) initRoutes() {
	for _, httpCfg := range s.cfg.HTTP.Servers {
		for _, loc := range httpCfg.Locations {
			s.router.AddRoute(loc.Path, s.buildLocationHandler(loc))
		}
	}
}

func (s *Server) buildLocationHandler(loc config.Location) http.Handler {
	var handler http.Handler
	if loc.Proxy != nil {
		if upstream, ok := s.upstreams[loc.Proxy.Upstream]; ok {
			handler = upstream
		}
	} else if loc.Root != "" {
		autoindex := false
		if loc.Autoindex != nil {
			autoindex = *loc.Autoindex
		}
		prefix := loc.Path
		rootHandler := static.New(loc.Root, autoindex)
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Strip location prefix from path
			r.URL.Path = r.URL.Path[len(prefix):]
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			rootHandler.ServeHTTP(w, r)
		})
	} else if loc.Return != "" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, loc.Return, http.StatusMovedPermanently)
		})
	}
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	return modules.BuildChain(&s.cfg.Modules, handler)
}

func (s *Server) Start() error {
	var wg sync.WaitGroup
	for _, listen := range s.cfg.Listen {
		srv := &http.Server{
			Addr:         listen.Addr,
			Handler:      s.router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		if listen.TLS != nil {
			cert, err := tls.LoadX509KeyPair(listen.TLS.Cert, listen.TLS.Key)
			if err != nil {
				return err
			}
			srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		}
		s.servers = append(s.servers, srv)
		wg.Add(1)
		go func(srv *http.Server, addr string) {
			defer wg.Done()
			log.Printf("listening on %s", addr)
			if srv.TLSConfig != nil {
				srv.ListenAndServeTLS("", "")
			} else {
				srv.ListenAndServe()
			}
		}(srv, listen.Addr)
	}
	wg.Wait()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	for _, srv := range s.servers {
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
