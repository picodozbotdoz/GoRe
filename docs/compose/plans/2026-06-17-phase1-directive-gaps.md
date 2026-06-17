# Phase 1: Directive Gap Clearing — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clear ~30 production-relevant directive gaps from the audit, focusing on quick-win config fields + simple middleware logic.

**Architecture:** Each batch adds config fields to `internal/config/config.go`, implements module logic in `internal/modules/<name>/`, and wires into `internal/modules/chain.go` or `internal/server/server.go`. All modules follow the existing `func(http.Handler) http.Handler` middleware pattern.

**Tech Stack:** Go 1.26, `net/http`, `crypto/tls`, standard library only.

---

## Batch 1: Core Server Config

### Task 1.1: error_page — Custom error pages per status code

**Files:**
- Modify: `internal/config/config.go` (add `ErrorPage` to `ModulesConfig`)
- Create: `internal/modules/errorpage/errorpage.go`
- Modify: `internal/modules/chain.go` (wire error_page middleware)
- Test: `internal/modules/errorpage/errorpage_test.go`

- [ ] **Step 1: Add config fields**

```go
// In config.go, add to ModulesConfig:
ErrorPage *ErrorPageConfig `yaml:"error_page,omitempty"`

// New types:
type ErrorPageConfig struct {
	Pages map[int]string `yaml:"pages,omitempty"` // status code → body or file path
}
```

- [ ] **Step 2: Write failing test**

```go
package errorpage

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorPage(t *testing.T) {
	cfg := &ErrorPageConfig{Pages: map[int]string{
		404: "custom 404 page",
		500: "custom 500 page",
	}}
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("original"))
	}))

	req := httptest.NewRequest("GET", "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec.Body.String() != "custom 404 page" {
		t.Fatalf("expected custom body, got %q", rec.Body.String())
	}
}

func TestErrorPageNoMatch(t *testing.T) {
	cfg := &ErrorPageConfig{Pages: map[int]string{
		404: "custom 404",
	}}
	handler := New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected original body, got %q", rec.Body.String())
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /media/doz/sdcard1/mimocode-workspace/gore && go test ./internal/modules/errorpage/ -v`
Expected: FAIL (package doesn't exist)

- [ ] **Step 4: Write implementation**

```go
package errorpage

import (
	"bytes"
	"fmt"
	"net/http"
)

type ErrorPageConfig struct {
	Pages map[int]string `yaml:"pages,omitempty"`
}

func New(cfg *ErrorPageConfig) func(http.Handler) http.Handler {
	if cfg == nil || len(cfg.Pages) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(sw, r)
			if body, ok := cfg.Pages[sw.status]; ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Del("Content-Length")
				w.WriteHeader(sw.status)
				fmt.Fprint(w, body)
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.written = true
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.written {
		sw.status = 200
		sw.written = true
	}
	return sw.ResponseWriter.Write(b)
}

func (sw *statusWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd /media/doz/sdcard1/mimocode-workspace/gore && go test ./internal/modules/errorpage/ -v`
Expected: PASS

- [ ] **Step 6: Wire into chain.go**

Add to `chain.go` imports and `BuildChain`:
```go
import "github.com/user/gore/internal/modules/errorpage"

// In BuildChain, before return:
if cfg.ErrorPage != nil {
    handler = errorpage.New(cfg.ErrorPage)(handler)
}
```

- [ ] **Step 7: Run full test suite**

Run: `cd /media/doz/sdcard1/mimocode-workspace/gore && go test ./...`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/modules/errorpage/ internal/modules/chain.go
git commit -m "feat: add error_page directive support"
```

---

### Task 1.2: server_tokens — Hide server identity

**Files:**
- Modify: `internal/config/config.go` (add `ServerTokens` to `ModulesConfig`)
- Modify: `internal/server/server.go` (strip Server header)
- Test: `internal/server/server_test.go`

- [ ] **Step 1: Add config field**

```go
// In ModulesConfig:
ServerTokens *bool `yaml:"server_tokens,omitempty"`
```

- [ ] **Step 2: Write failing test**

```go
func TestServerTokensHidden(t *testing.T) {
	cfg := &config.Config{...}
	cfg.Modules.ServerTokens = boolPtr(false)
	srv := server.New(cfg)
	// ... start test server, GET /, check no "Server" header
}

func boolPtr(b bool) *bool { return &b }
```

- [ ] **Step 3: Run test to verify it fails**

- [ ] **Step 4: Write implementation in server.go**

In `buildHTTP2Config` or as middleware in `BuildChain`:
```go
if cfg.ServerTokens != nil && !*cfg.ServerTokens {
    handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Del("Server")
        handler.ServeHTTP(w, r)
    })
}
```

- [ ] **Step 5: Run test to verify it passes**

- [ ] **Step 6: Commit**

---

### Task 1.3: default_type — Default Content-Type

**Files:**
- Modify: `internal/config/config.go` (add `DefaultType` to `ModulesConfig`)
- Modify: `internal/modules/chain.go` (wire middleware)
- Create: `internal/modules/defaulttype/defaulttype.go`
- Test: `internal/modules/defaulttype/defaulttype_test.go`

- [ ] **Step 1: Add config field**

```go
DefaultType string `yaml:"default_type,omitempty"`
```

- [ ] **Step 2–5: TDD cycle** (same pattern as error_page)

- [ ] **Step 6: Commit**

---

### Task 1.4: keepalive_timeout (configurable) + keepalive_requests

**Files:**
- Modify: `internal/config/config.go` (add to `Listen` or `ModulesConfig`)
- Modify: `internal/server/server.go` (wire to `http.Server.IdleTimeout`)

- [ ] **Step 1: Add config fields**

```go
// In Listen struct:
KeepAliveTimeout int `yaml:"keepalive_timeout,omitempty"`
KeepAliveRequests int `yaml:"keepalive_requests,omitempty"`
```

- [ ] **Step 2: Write test** — verify custom timeout is applied

- [ ] **Step 3–4: Implement** — replace hardcoded `120s` in `server.go:232`

```go
idleTimeout := 120
if listen.KeepAliveTimeout > 0 {
    idleTimeout = listen.KeepAliveTimeout
}
srv := &http.Server{
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  time.Duration(idleTimeout) * time.Second,
}
```

- [ ] **Step 5: Run tests, commit**

---

### Task 1.5: client_body_timeout, client_header_timeout, send_timeout

**Files:**
- Modify: `internal/config/config.go` (add to `ModulesConfig` or `Listen`)
- Modify: `internal/server/server.go` (wire to `http.Server` timeouts)

- [ ] **Step 1: Add config fields**

```go
// In ModulesConfig or Listen:
ClientBodyTimeout    int `yaml:"client_body_timeout,omitempty"`
ClientHeaderTimeout  int `yaml:"client_header_timeout,omitempty"`
SendTimeout          int `yaml:"send_timeout,omitempty"`
```

- [ ] **Step 2–4: TDD** — replace hardcoded 30s timeouts in server.go

```go
readTimeout := 30
if cfg.Modules.ClientHeaderTimeout > 0 {
    readTimeout = cfg.Modules.ClientHeaderTimeout
}
writeTimeout := 30
if cfg.Modules.SendTimeout > 0 {
    writeTimeout = cfg.Modules.SendTimeout
}
srv := &http.Server{
    ReadTimeout:  time.Duration(readTimeout) * time.Second,
    WriteTimeout: time.Duration(writeTimeout) * time.Second,
    IdleTimeout:  time.Duration(idleTimeout) * time.Second,
}
```

- [ ] **Step 5: Run tests, commit**

---

### Task 1.6: merge_slashes — Merge consecutive URI slashes

**Files:**
- Modify: `internal/config/config.go` (add `MergeSlashes` to `ModulesConfig`)
- Modify: `internal/modules/chain.go` (wire middleware)
- Create: `internal/modules/mergeslashes/mergeslashes.go`
- Test: `internal/modules/mergeslashes/mergeslashes_test.go`

- [ ] **Step 1: Add config field**

```go
MergeSlashes *bool `yaml:"merge_slashes,omitempty"`
```

- [ ] **Step 2–5: TDD** — middleware that collapses `//+` to `/` in `r.URL.Path`

```go
func New(enabled bool) func(http.Handler) http.Handler {
    if !enabled {
        return func(next http.Handler) http.Handler { return next }
    }
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            for strings.Contains(r.URL.Path, "//") {
                r.URL.Path = strings.ReplaceAll(r.URL.Path, "//", "/")
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

- [ ] **Step 6: Commit**

---

## Batch 2: Gzip Enhancements

### Task 2.1: gzip_min_length, gzip_vary, gzip_proxied, gzip_disable

**Files:**
- Modify: `internal/config/config.go` (add fields to `GzipConfig`)
- Modify: `internal/modules/gzip/gzip.go` (add logic)
- Test: `internal/modules/gzip/gzip_test.go`

- [ ] **Step 1: Add config fields**

```go
type GzipConfig struct {
    Enabled  bool     `yaml:"enabled"`
    Level    int      `yaml:"level,omitempty"`
    Types    []string `yaml:"types,omitempty"`
    MinLength int     `yaml:"min_length,omitempty"`
    Vary     bool     `yaml:"vary,omitempty"`
    Proxied  bool     `yaml:"proxied,omitempty"`
    Disable  string   `yaml:"disable,omitempty"` // User-Agent pattern
}
```

- [ ] **Step 2: Write failing tests**

```go
func TestGzipMinLength(t *testing.T) {
    h := gzip.New(6, nil)
    h.MinLength = 100

    handler := h.ServeHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("short")) // 5 bytes < 100
    }))

    req := httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Accept-Encoding", "gzip")
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Header().Get("Content-Encoding") == "gzip" {
        t.Fatal("should not compress short response")
    }
}

func TestGzipVary(t *testing.T) {
    h := gzip.New(6, nil)
    h.Vary = true

    handler := h.ServeHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte("hello world this is long enough"))
    }))

    req := httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Accept-Encoding", "gzip")
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Header().Get("Vary") != "Accept-Encoding" {
        t.Fatalf("expected Vary header, got %q", rec.Header().Get("Vary"))
    }
}
```

- [ ] **Step 3: Run tests to verify they fail**

- [ ] **Step 4: Implement** — add MinLength check in `ServeHTTP`, add Vary header after compression

```go
// In ServeHTTP, after accept-encoding check:
if h.MinLength > 0 {
    // Need to check response size — use content-length if set, else buffer
    cl := rw.Header().Get("Content-Length")
    if cl != "" {
        size, _ := strconv.ParseInt(cl, 10, 64)
        if size < int64(h.MinLength) {
            next.ServeHTTP(w, r)
            return
        }
    }
}

// After gz.Close():
if h.Vary {
    w.Header().Set("Vary", "Accept-Encoding")
}
```

- [ ] **Step 5: Run tests, commit**

---

## Batch 3: Rate/Conn Limit Enhancements

### Task 3.1: limit_req_status (configurable)

**Files:**
- Modify: `internal/config/config.go` (add `Status` to `RateLimitConfig`)
- Modify: `internal/modules/ratelimit/ratelimit.go` (use config)
- Test: `internal/modules/ratelimit/ratelimit_test.go`

- [ ] **Step 1: Add config field**

```go
type RateLimitConfig struct {
    Zone   string `yaml:"zone"`
    Rate   string `yaml:"rate"`
    Burst  int    `yaml:"burst,omitempty"`
    Status int    `yaml:"status,omitempty"` // default 429
}
```

- [ ] **Step 2: Update Limiter.New()**

```go
func New(rate string, burst int, status int) *Limiter {
    if status == 0 {
        status = http.StatusTooManyRequests
    }
    // ... store status in Limiter struct
}
```

- [ ] **Step 3: Update ServeHTTP** to use `l.status` instead of hardcoded 429

- [ ] **Step 4: Update chain.go** to pass status

- [ ] **Step 5: Run tests, commit**

---

### Task 3.2: limit_req_log_level, limit_conn_log_level

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/modules/ratelimit/ratelimit.go`
- Modify: `internal/modules/limitconn/limitconn.go`

- [ ] **Step 1: Add config fields**

```go
type RateLimitConfig struct {
    // ...existing...
    LogLevel string `yaml:"log_level,omitempty"`
}
type LimitConnConfig struct {
    // ...existing...
    LogLevel string `yaml:"log_level,omitempty"`
}
```

- [ ] **Step 2: Implement** — log rate limit events at configured level

- [ ] **Step 3: Run tests, commit**

---

## Batch 4: Sub Filter / Mirror Enhancements

### Task 4.1: sub_filter_once — Replace only first occurrence

**Files:**
- Modify: `internal/config/config.go` (add `SubFilterOnce` to `Location`)
- Modify: `internal/modules/subfilter/subfilter.go` (use bytes.Replace vs bytes.ReplaceAll)
- Test: `internal/modules/subfilter/subfilter_test.go`

- [ ] **Step 1: Add config field**

```go
// In Location struct:
SubFilterOnce *bool `yaml:"sub_filter_once,omitempty"`
```

- [ ] **Step 2: Update subfilter.New()** signature to accept `once bool`

```go
func New(replacements map[string]string, once bool) func(http.Handler) http.Handler {
    // ...
    if once {
        body = bytes.Replace(body, []byte(old), []byte(new), 1)
    } else {
        body = bytes.ReplaceAll(body, []byte(old), []byte(new))
    }
}
```

- [ ] **Step 3: Update server.go** to pass `once` flag

- [ ] **Step 4: Run tests, commit**

---

### Task 4.2: sub_filter_types — Restrict to content types

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/modules/subfilter/subfilter.go`

- [ ] **Step 1: Add config field**

```go
// In Location:
SubFilterTypes []string `yaml:"sub_filter_types,omitempty"`
```

- [ ] **Step 2: Implement** — check Content-Type before applying replacements

- [ ] **Step 3: Run tests, commit**

---

### Task 4.3: mirror_request_body — Forward body to mirror

**Files:**
- Modify: `internal/modules/mirror/mirror.go`

- [ ] **Step 1: Fix mirror.New()** — pass request body

```go
// In mirror handler, change nil body to:
var body io.Reader
if r.Body != nil {
    body = r.Body
}
mirrorReq, err := http.NewRequestWithContext(r.Context(), r.Method, mirrorURL+r.URL.Path, body)
```

- [ ] **Step 2: Run tests, commit**

---

## Batch 5: Headers Enhancements

### Task 5.1: expires — Set Expires/Cache-Control headers

**Files:**
- Modify: `internal/config/config.go` (add `Expires` to `HeadersConfig` or `Location`)
- Create: `internal/modules/headers/expires.go`
- Test: `internal/modules/headers/expires_test.go`

- [ ] **Step 1: Add config field**

```go
type HeadersConfig struct {
    Add    map[string]string `yaml:"add,omitempty"`
    Remove []string          `yaml:"remove,omitempty"`
    Expires string           `yaml:"expires,omitempty"` // "access plus 1 hour", "@time", duration
}
```

- [ ] **Step 2: Implement** — parse nginx-style expires directive, set `Expires` and `Cache-Control` headers

- [ ] **Step 3: Run tests, commit**

---

### Task 5.2: add_header with `always` flag

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/modules/headers/headers.go`

- [ ] **Step 1: Add always flag support**

```go
type HeaderEntry struct {
    Name    string `yaml:"name"`
    Value   string `yaml:"value"`
    Always  bool   `yaml:"always,omitempty"`
}
```

- [ ] **Step 2: Update responseWriter.WriteHeader** — only skip error codes if `always` is false

- [ ] **Step 3: Run tests, commit**

---

## Batch 6: Proxy Enhancements

### Task 6.1: proxy_redirect — Rewrite Location headers

**Files:**
- Modify: `internal/config/config.go` (add `Redirect` to `Proxy`)
- Modify: `internal/proxy/proxy.go` (add response modifier)
- Test: `internal/proxy/proxy_test.go`

- [ ] **Step 1: Add config field**

```go
type Proxy struct {
    // ...existing...
    Redirect string `yaml:"redirect,omitempty"` // "default" or custom pattern
}
```

- [ ] **Step 2: Implement** — modify `Upstream.Proxy.ModifyResponse` to rewrite Location headers

```go
if upstreamCfg.Redirect != "" {
    u.Proxy.ModifyResponse = func(resp *http.Response) error {
        if loc := resp.Header.Get("Location"); loc != "" {
            // rewrite based on redirect config
            resp.Header.Set("Location", rewriteRedirect(loc, upstreamCfg.Redirect))
        }
        return nil
    }
}
```

- [ ] **Step 3: Run tests, commit**

---

### Task 6.2: proxy_buffer_size — Wire existing config

**Files:**
- Modify: `internal/proxy/proxy.go` (use `BufferSize` from config)

- [ ] **Step 1: Wire proxy.BufferSize** to `httputil.ReverseProxy.BufferSize`

```go
if bufferSize != 0 {
    u.Proxy.BufferSize = bufferSize
}
```

- [ ] **Step 2: Run tests, commit**

---

### Task 6.3: proxy_next_upstream flags

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/proxy/proxy.go`

- [ ] **Step 1: Add config fields**

```go
type Upstream struct {
    // ...existing...
    NextUpstream        string `yaml:"next_upstream,omitempty"`        // "error timeout invalid_header"
    NextUpstreamTries   int    `yaml:"next_upstream_tries,omitempty"`
    NextUpstreamTimeout int    `yaml:"next_upstream_timeout,omitempty"`
}
```

- [ ] **Step 2: Parse flags and update retry logic**

```go
// In Upstream.ServeHTTP, check flags:
if (flags.Contains("error") && lastErr != nil) ||
   (flags.Contains("timeout") && isTimeout(lastErr)) ||
   (flags.Contains("invalid_header") && isInvalidHeader(lastErr)) {
    continue
}
```

- [ ] **Step 3: Run tests, commit**

---

### Task 6.4: proxy_pass_request_headers, proxy_pass_request_body

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/proxy/proxy.go`

- [ ] **Step 1: Add config fields**

```go
type Proxy struct {
    // ...existing...
    PassRequestHeaders *bool `yaml:"pass_request_headers,omitempty"`
    PassRequestBody    *bool `yaml:"pass_request_body,omitempty"`
}
```

- [ ] **Step 2: Implement** — in director, optionally clear headers/body

- [ ] **Step 3: Run tests, commit**

---

## Batch 7: Upstream Enhancements

### Task 7.1: least_conn (fix stub) + ip_hash + hash

**Files:**
- Modify: `internal/proxy/balancer.go`
- Modify: `internal/proxy/proxy.go`

- [ ] **Step 1: Implement LeastConnBalancer**

```go
type LeastConnBalancer struct {
    servers []*Server
    mu      sync.Mutex
}

func (b *LeastConnBalancer) Next() *Server {
    b.mu.Lock()
    defer b.mu.Unlock()
    var best *Server
    for _, s := range b.servers {
        if atomic.LoadInt32(&s.Healthy) != 1 {
            continue
        }
        if best == nil || s.ActiveConns < best.ActiveConns {
            best = s
        }
    }
    if best != nil {
        atomic.AddInt64(&best.ActiveConns, 1)
    }
    return best
}
```

- [ ] **Step 2: Implement IPHashBalancer**

```go
type IPHashBalancer struct {
    servers []*Server
}

func (b *IPHashBalancer) Next() *Server {
    // hash based on request context — needs request passed in
}
```

- [ ] **Step 3: Update NewUpstream switch**

```go
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
```

- [ ] **Step 4: Run tests, commit**

---

### Task 7.2: backup, down server support

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/proxy/balancer.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add fields to UpstreamServer**

```go
type UpstreamServer struct {
    Addr   string `yaml:"addr"`
    Weight int    `yaml:"weight,omitempty"`
    Backup bool   `yaml:"backup,omitempty"`
    Down   bool   `yaml:"down,omitempty"`
}
```

- [ ] **Step 2: Update balancer** — skip `down` servers, use `backup` only when all primary are unhealthy

- [ ] **Step 3: Run tests, commit**

---

### Task 7.3: upstream keepalive_timeout, keepalive_requests

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/proxy/proxy.go`

- [ ] **Step 1: Add config fields**

```go
type Upstream struct {
    // ...existing...
    KeepaliveTimeout int `yaml:"keepalive_timeout,omitempty"`
    KeepaliveRequests int `yaml:"keepalive_requests,omitempty"`
}
```

- [ ] **Step 2: Wire to transport settings**

```go
// In transport():
if tc.KeepaliveTimeout > 0 {
    transport.IdleConnTimeout = time.Duration(tc.KeepaliveTimeout) * time.Second
}
```

- [ ] **Step 3: Run tests, commit**

---

## Batch 8: Real IP Enhancements

### Task 8.1: set_real_ip_from (multi-CIDR) + real_ip_recursive

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/modules/realip/realip.go`
- Test: `internal/modules/realip/realip_test.go`

- [ ] **Step 1: Add config fields**

```go
type RealIPConfig struct {
    From      []string `yaml:"from,omitempty"`      // multiple CIDRs
    Recursive bool     `yaml:"recursive,omitempty"`
}
```

- [ ] **Step 2: Implement recursive IP parsing**

```go
func extractIPRecursive(header string, trusted []*net.IPNet) string {
    parts := strings.Split(header, ",")
    for i := len(parts) - 1; i >= 0; i-- {
        ip := parseIP(strings.TrimSpace(parts[i]))
        if ip == nil {
            continue
        }
        isTrusted := false
        for _, cidr := range trusted {
            if cidr.Contains(ip) {
                isTrusted = true
                break
            }
        }
        if !isTrusted {
            return ip.String()
        }
    }
    return ""
}
```

- [ ] **Step 3: Run tests, commit**

---

## Verification

After all tasks complete:

- [ ] Run full test suite: `go test -race ./...`
- [ ] Run linter: `golangci-lint run`
- [ ] Build binary: `go build ./cmd/gore`
- [ ] Update `docs/directive-audit.md` — change statuses for all implemented directives
- [ ] Commit audit doc update
