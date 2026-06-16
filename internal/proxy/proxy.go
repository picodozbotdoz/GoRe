package proxy

import (
	"crypto/tls"
	"fmt"
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

type TimeoutConfig struct {
	Connect int // seconds
	Read    int // seconds
	Send    int // seconds
	Idle    int // seconds
}

func NewUpstream(name string, servers []*Server, strategy string, timeouts *TimeoutConfig) *Upstream {
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

	u := &Upstream{Name: name, Balancer: balancer}
	u.Proxy = &httputil.ReverseProxy{
		Director:     u.director,
		Transport:    u.transport(timeouts),
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
	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("X-Forwarded-Proto", "http")

	if req.TLS != nil {
		req.Header.Set("X-Forwarded-Proto", "https")
	}
}

func (u *Upstream) transport(tc *TimeoutConfig) *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
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
	u.Proxy.ServeHTTP(w, r)
}
