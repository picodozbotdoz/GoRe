package proxy

import (
	"sync/atomic"
)

type Server struct {
	Addr    string
	Weight  int
	Healthy int32
}

type Balancer interface {
	Next() *Server
}

type RoundRobinBalancer struct {
	servers []*Server
	counter uint64
}

func NewRoundRobin(servers []*Server) *RoundRobinBalancer {
	for _, s := range servers {
		if s.Weight == 0 {
			s.Weight = 1
		}
		atomic.StoreInt32(&s.Healthy, 1)
	}
	return &RoundRobinBalancer{servers: servers}
}

func (b *RoundRobinBalancer) Next() *Server {
	n := len(b.servers)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		idx := atomic.AddUint64(&b.counter, 1) % uint64(n)
		if atomic.LoadInt32(&b.servers[idx].Healthy) == 1 {
			return b.servers[idx]
		}
	}
	return nil
}
