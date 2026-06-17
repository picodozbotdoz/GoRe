package proxy

import (
	"hash/fnv"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type Server struct {
	Addr        string
	Weight      int
	Healthy     int32
	ActiveConns int64
	Backup      bool
	Down        bool
}

type Balancer interface {
	Next(r *http.Request) *Server
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
		if !s.Down {
			atomic.StoreInt32(&s.Healthy, 1)
		}
	}
	return &RoundRobinBalancer{servers: servers}
}

func (b *RoundRobinBalancer) Next(r *http.Request) *Server {
	n := len(b.servers)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		idx := atomic.AddUint64(&b.counter, 1) % uint64(n)
		s := b.servers[idx]
		if s.Down || s.Backup {
			continue
		}
		if atomic.LoadInt32(&s.Healthy) == 1 {
			return s
		}
	}
	return fallbackToBackup(b.servers)
}

type LeastConnBalancer struct {
	servers []*Server
	mu      sync.Mutex
}

func NewLeastConn(servers []*Server) *LeastConnBalancer {
	for _, s := range servers {
		if s.Weight == 0 {
			s.Weight = 1
		}
		if !s.Down {
			atomic.StoreInt32(&s.Healthy, 1)
		}
	}
	return &LeastConnBalancer{servers: servers}
}

func (b *LeastConnBalancer) Next(r *http.Request) *Server {
	b.mu.Lock()
	defer b.mu.Unlock()
	var best *Server
	for _, s := range b.servers {
		if s.Down || s.Backup {
			continue
		}
		if atomic.LoadInt32(&s.Healthy) != 1 {
			continue
		}
		if best == nil || s.ActiveConns < best.ActiveConns {
			best = s
		}
	}
	if best == nil {
		return fallbackToBackup(b.servers)
	}
	if best != nil {
		atomic.AddInt64(&best.ActiveConns, 1)
	}
	return best
}

type IPHashBalancer struct {
	servers []*Server
}

func NewIPHash(servers []*Server) *IPHashBalancer {
	for _, s := range servers {
		if s.Weight == 0 {
			s.Weight = 1
		}
		if !s.Down {
			atomic.StoreInt32(&s.Healthy, 1)
		}
	}
	return &IPHashBalancer{servers: servers}
}

func (b *IPHashBalancer) Next(r *http.Request) *Server {
	ip := extractClientIP(r)
	idx := hashIP(ip) % uint32(len(b.servers))
	for i := uint32(0); i < uint32(len(b.servers)); i++ {
		s := b.servers[(idx+i)%uint32(len(b.servers))]
		if s.Down || s.Backup {
			continue
		}
		if atomic.LoadInt32(&s.Healthy) == 1 {
			return s
		}
	}
	return fallbackToBackup(b.servers)
}

type ConsistentHashBalancer struct {
	servers []*Server
	ring    []ringEntry
}

type ringEntry struct {
	hash     uint32
	server   *Server
	replicas int
}

func NewConsistentHash(servers []*Server) *ConsistentHashBalancer {
	for _, s := range servers {
		if s.Weight == 0 {
			s.Weight = 1
		}
		if !s.Down {
			atomic.StoreInt32(&s.Healthy, 1)
		}
	}
	b := &ConsistentHashBalancer{servers: servers}
	b.buildRing()
	return b
}

func (b *ConsistentHashBalancer) buildRing() {
	b.ring = nil
	for _, s := range b.servers {
		if s.Down {
			continue
		}
		replicas := s.Weight * 16
		if replicas < 1 {
			replicas = 1
		}
		for i := 0; i < replicas; i++ {
			h := fnvHash(s.Addr + "-" + string(rune(i)))
			b.ring = append(b.ring, ringEntry{hash: h, server: s, replicas: i})
		}
	}
}

func (b *ConsistentHashBalancer) Next(r *http.Request) *Server {
	if len(b.ring) == 0 {
		return fallbackToBackup(b.servers)
	}
	key := r.URL.Path
	if key == "" {
		key = "/"
	}
	h := fnvHash(key)
	idx := len(b.ring)
	for i, entry := range b.ring {
		if entry.hash >= h {
			idx = i
			break
		}
	}
	for i := 0; i < len(b.ring); i++ {
		entry := b.ring[(idx+i)%len(b.ring)]
		if entry.server.Down || entry.server.Backup {
			continue
		}
		if atomic.LoadInt32(&entry.server.Healthy) == 1 {
			return entry.server
		}
	}
	return fallbackToBackup(b.servers)
}

func fallbackToBackup(servers []*Server) *Server {
	for _, s := range servers {
		if s.Down || !s.Backup {
			continue
		}
		if atomic.LoadInt32(&s.Healthy) == 1 {
			return s
		}
	}
	return nil
}

func extractClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func hashIP(ip string) uint32 {
	return fnvHash(ip)
}

func fnvHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
