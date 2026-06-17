package proxy

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

func StartHealthCheck(servers []*Server, interval int, path string) {
	if len(servers) == 0 || interval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		client := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
			},
		}

		for range ticker.C {
			for _, s := range servers {
				checkServer(client, s, path)
			}
		}
	}()
}

func checkServer(client *http.Client, s *Server, path string) {
	if path != "" {
		resp, err := client.Get("http://" + s.Addr + path)
		if err != nil {
			atomic.StoreInt32(&s.Healthy, 0)
			return
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			if atomic.LoadInt32(&s.Healthy) == 0 && s.SlowStart > 0 {
				s.CreatedAt = time.Now().Unix()
			}
			atomic.StoreInt32(&s.Healthy, 1)
		} else {
			atomic.StoreInt32(&s.Healthy, 0)
		}
		return
	}

	conn, err := net.DialTimeout("tcp", s.Addr, 3*time.Second)
	if err != nil {
		atomic.StoreInt32(&s.Healthy, 0)
		return
	}
	conn.Close()
	if atomic.LoadInt32(&s.Healthy) == 0 && s.SlowStart > 0 {
		s.CreatedAt = time.Now().Unix()
	}
	atomic.StoreInt32(&s.Healthy, 1)
}
