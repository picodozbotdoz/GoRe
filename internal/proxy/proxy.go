package proxy

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Upstream struct {
	Name     string
	Balancer Balancer
	Proxy    *httputil.ReverseProxy
}

func NewUpstream(name string, servers []*Server, strategy string) *Upstream {
	var balancer Balancer
	switch strategy {
	case "least-conn":
		balancer = NewRoundRobin(servers)
	default:
		balancer = NewRoundRobin(servers)
	}

	u := &Upstream{Name: name, Balancer: balancer}
	u.Proxy = &httputil.ReverseProxy{
		Director:     u.director,
		Transport:    u.transport(),
		ErrorHandler: u.ErrorHandler,
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
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
}

func (u *Upstream) transport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	}
}

func (u *Upstream) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("upstream error: %v", err)
	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

func (u *Upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u.Proxy.ServeHTTP(w, r)
}
