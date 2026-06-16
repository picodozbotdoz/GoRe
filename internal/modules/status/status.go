package status

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var collector *Collector

func init() {
	collector = &Collector{
		startTime: time.Now(),
	}
}

type Collector struct {
	Active    atomic.Int64
	Requests  atomic.Int64
	Accepted  atomic.Int64
	Reading   atomic.Int64
	Writing   atomic.Int64
	startTime time.Time
}

func Get() *Collector {
	return collector
}

func (c *Collector) ConnAccepted() {
	c.Accepted.Add(1)
}

func (c *Collector) ConnClosed() {
	c.Active.Add(-1)
}

func (c *Collector) ReqStart() {
	c.Requests.Add(1)
	c.Active.Add(1)
	c.Writing.Add(1)
}

func (c *Collector) ReqDone() {
	c.Active.Add(-1)
	c.Writing.Add(-1)
}

func (c *Collector) Uptime() int64 {
	return int64(time.Since(c.startTime).Seconds())
}

type Handler struct {
	path string
}

func NewHandler(path string) http.Handler {
	return &Handler{path: path}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.path {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	uptime := collector.Uptime()
	active := collector.Active.Load()
	requests := collector.Requests.Load()
	accepted := collector.Accepted.Load()
	reading := collector.Reading.Load()
	writing := collector.Writing.Load()
	waiting := active - writing

	fmt.Fprintf(w, "Active connections: %d\n", active)
	fmt.Fprintf(w, "server accepts handled requests\n")
	fmt.Fprintf(w, " %d %d %d\n", accepted, accepted, requests)
	fmt.Fprintf(w, "Reading: %d Writing: %d Waiting: %d\n", reading, writing, waiting)
	fmt.Fprintf(w, "Uptime: %ds\n", uptime)
}
