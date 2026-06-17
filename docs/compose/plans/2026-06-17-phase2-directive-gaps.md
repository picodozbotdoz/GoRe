# Phase 2: Directive Gap Clearing — Implementation Plan

> **For agentic workers:** Use compose:subagent to implement task-by-task.

**Goal:** Clear ~38 medium/small-effort directive gaps from the audit.

**Architecture:** Same as Phase 1 — config fields in `config.go`, module logic in `internal/modules/` or `internal/proxy/`, wire into chain.go/server.go. Follow existing patterns.

**Tech Stack:** Go 1.26, `net/http`, `crypto/tls`, standard library only.

---

## Batch 9: Proxy SSL (8 directives)

All proxy_ssl directives configure upstream TLS transport. They go in `internal/config/config.go` (ProxySSL struct) and `internal/proxy/proxy.go` (transport TLS config).

### Task 9.1: proxy_ssl_verify, proxy_ssl_certificate, proxy_ssl_trusted_certificate

**Files:**
- Modify: `internal/config/config.go` — add `ProxySSL` to `Upstream`
- Modify: `internal/proxy/proxy.go` — wire to transport TLS config
- Test: `internal/proxy/proxy_test.go`

Config struct:
```go
type ProxySSL struct {
    Verify             bool   `yaml:"verify,omitempty"`
    Certificate        string `yaml:"certificate,omitempty"`
    CertificateKey     string `yaml:"certificate_key,omitempty"`
    TrustedCertificate string `yaml:"trusted_certificate,omitempty"`
    Protocols          string `yaml:"protocols,omitempty"`
    Ciphers            string `yaml:"ciphers,omitempty"`
    ServerName         string `yaml:"server_name,omitempty"`
    SessionReuse       *bool  `yaml:"session_reuse,omitempty"`
    Name               string `yaml:"name,omitempty"`
}
```

Wire in `transport()`:
```go
if u.ProxySSL != nil {
    tlsConfig := &tls.Config{}
    if u.ProxySSL.Verify {
        // Load trusted cert
        caCert, _ := os.ReadFile(u.ProxySSL.TrustedCertificate)
        caPool := x509.NewCertPool()
        caPool.AppendCertsFromPEM(caCert)
        tlsConfig.RootCAs = caPool
    }
    if u.ProxySSL.Certificate != "" {
        cert, _ := tls.LoadX509KeyPair(u.ProxySSL.Certificate, u.ProxySSL.CertificateKey)
        tlsConfig.Certificates = []tls.Certificate{cert}
    }
    if u.ProxySSL.ServerName != "" {
        tlsConfig.ServerName = u.ProxySSL.ServerName
    }
    transport.TLSClientConfig = tlsConfig
}
```

### Task 9.2: proxy_ssl_protocols, proxy_ssl_ciphers

Wire protocol/cipher config to tls.Config:
```go
if u.ProxySSL.Protocols != "" {
    // Parse "TLSv1.2 TLSv1.3" → set MinVersion
}
if len(u.ProxySSL.Ciphers) > 0 {
    // Map cipher names to IDs
}
```

### Task 9.3: proxy_ssl_server_name, proxy_ssl_name, proxy_ssl_session_reuse

- ServerName/Name: set `tlsConfig.ServerName`
- SessionReuse: control `tlsConfig.SessionTicketsDisabled`

---

## Batch 10: Proxy Misc (8 directives)

### Task 10.1: proxy_request_buffering

Config: `Proxy.RequestBuffering *bool`
In director: if false, stream body directly (already default in Go). If true, buffer fully.

### Task 10.2: proxy_intercept_errors

Config: `Proxy.InterceptErrors bool`, `Proxy.ErrorPages map[int]string`
ModifyResponse: if status matches error pages, replace body.

### Task 10.3: proxy_cookie_domain, proxy_cookie_path

ModifyResponse: rewrite Set-Cookie Domain/Path attributes.

### Task 10.4: proxy_hide_header

Config: `Upstream.HideHeaders []string`
ModifyResponse: delete specified headers from response.

### Task 10.5: proxy_method

Config: `Proxy.Method string`
Director: override request method.

### Task 10.6: proxy_socket_keepalive

Config: `Upstream.SocketKeepalive bool`
Transport: enable `net.Dialer.KeepAlive`.

### Task 10.7: proxy_buffers, proxy_busy_buffers_size

Config: `Proxy.Buffers string`, `Proxy.BusyBuffersSize string`
These control buffering behavior — wire to ReverseProxy.BufferPool size.

---

## Batch 11: Proxy Cache Extras (6 directives)

Extend existing cache system in `internal/proxy/cache/`.

### Task 11.1: proxy_cache_valid

Config: `CacheConfig.Valid map[int]int` (status code → TTL seconds)

### Task 11.2: proxy_cache_use_stale

Config: `CacheConfig.UseStale bool`
Serve stale entry on upstream error/timeout.

### Task 11.3: proxy_cache_lock

Config: `CacheConfig.Lock bool`
Prevent cache stampede — only one request populates cache.

### Task 11.4: proxy_cache_key

Config: `CacheConfig.Key string`
Custom cache key format.

### Task 11.5: proxy_no_cache, proxy_cache_bypass

Config: `CacheConfig.NoCache string`, `CacheConfig.Bypass string`
Conditional cache bypass.

---

## Batch 12: Core + Auth (7 directives)

### Task 12.1: alias

Config: `Location.Alias string`
Like root but replaces the matched path prefix.

### Task 12.2: limit_except

Config: `Location.LimitExcept []string`
Restrict location to specific HTTP methods.

### Task 12.3: internal

Config: `Location.Internal bool`
Mark location as internal-only (return 404 for external requests).

### Task 12.4: satisfy

Config: `Location.Satisfy string` ("all" or "any")
Combine auth_basic + access rules.

### Task 12.5: auth_request_set

Config: `Location.AuthRequestSet map[string]string`
Map auth response headers to request headers.

### Task 12.6: auth_basic_user_file (encrypted)

Support bcrypt/sha passwords in basic_auth.users.

### Task 12.7: resolver

Config: `ModulesConfig.Resolver string`
DNS resolver for upstream resolution.

---

## Batch 13: SSL Extras (5 directives)

### Task 13.1: ssl_session_timeout

Config: `TLS.SessionTimeout int`

### Task 13.2: ssl_client_certificate, ssl_verify_client, ssl_verify_depth

Config: `TLS.ClientCertificate string`, `TLS.VerifyClient bool`, `TLS.VerifyDepth int`

### Task 13.3: ssl_reject_handshake

Config: `TLS.RejectHandshake bool`
Reject TLS handshake immediately.

---

## Batch 14: Small Wins (6 directives)

### Task 14.1: gzip_static

Config: `GzipConfig.Static bool`
Serve pre-compressed .gz files.

### Task 14.2: log_subrequest

Config: `AccessLogConfig.Subrequest bool`

### Task 14.3: rewrite_log

Config: `Rewrite.Log bool`

### Task 14.4: break

Config: add to rewrite processing.

### Task 14.5: proxy_protocol

Config: `Upstream.ProxyProtocol bool`

### Task 14.6: proxy_max_temp_file_size

Config: `Proxy.MaxTempFileSize string`

---

## Verification

After all batches:
- `go test -race ./...`
- `golangci-lint run`
- `go build ./cmd/gore`
- Update `docs/directive-audit.md`
