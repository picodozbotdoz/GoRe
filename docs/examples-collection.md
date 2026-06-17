# GoRe Examples Collection

Tracked, verified, runnable configuration examples. Each example is in `examples/<name>/` with:
- `gore.yaml` — directly runnable with `gore -c`
- `before.sh` — prerequisites (certs, dirs, mock backends)
- `after.sh` — cleanup
- `test.sh` — curl assertions verifying expected behavior

## Status Legend

- ⬜ Pending — planned, not started
- 🔧 In Progress — being written
- ✅ Done — written, executed, verified
- 🔨 Logging — logging improved during verification

## Priority 1 — Core Features

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 1 | Static files | `examples/static-files/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Serve HTML/CSS/JS from directory |
| 2 | Reverse proxy | `examples/reverse-proxy/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Proxy to upstream with load balancing |
| 3 | TLS/HTTPS | `examples/tls-https/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Self-signed cert, HTTP/2 ALPN |

## Priority 2 — Protocol Features

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 4 | HTTP/2 h2c | `examples/http2-cleartext/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Cleartext HTTP/2 upgrade |
| 5 | HTTP/3 QUIC | `examples/http3-quic/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | QUIC with Alt-Svc |
| 6 | Gzip compression | `examples/gzip/` | ✅ | ✅ | ✅ | ✅ | ✅ | 🔨 | Compress text/JSON responses |
| 7 | Brotli compression | `examples/brotli/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Better ratio than gzip |

## Priority 3 — Security Features

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 8 | Rate limiting | `examples/rate-limit/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Per-IP token bucket |
| 9 | Access control | `examples/access-control/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | CIDR allow/deny rules |
| 10 | Basic auth | `examples/basic-auth/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Username/password protection |
| 11 | Auth subrequest | `examples/auth-request/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Delegate auth to external service |
| 12 | Body size limit | `examples/body-limit/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Reject oversized POST bodies |
| 13 | Connection limit | `examples/conn-limit/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Per-IP concurrent connections |

## Priority 4 — Proxy Features

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 14 | Load balancing | `examples/load-balancing/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Round-robin across backends |
| 15 | Proxy timeouts | `examples/proxy-timeouts/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Connect/read/send timeout config |
| 16 | Proxy retry | `examples/proxy-retry/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Automatic failover on 5xx |
| 17 | WebSocket proxy | `examples/websocket-proxy/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Bidirectional streaming |
| 18 | Proxy cache | `examples/proxy-cache/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | In-memory response caching |
| 19 | Proxy buffering | `examples/proxy-buffering/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Streaming vs buffered mode |
| 20 | Upstream keepalive | `examples/upstream-keepalive/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Connection pooling |
| 21 | Set headers | `examples/proxy-headers/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Custom upstream headers |
| 22 | Health checks | `examples/health-check/` | ✅ | ✅ | ✅ | ✅ | ✅ | ⬜ | Active backend probing |

## Priority 5 — Advanced Features

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 23 | URL rewrite | `examples/rewrite/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Regex path rewriting |
| 24 | try_files | `examples/try-files/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | SPA fallback chain |
| 25 | Cache-Control | `examples/cache-control/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Browser/CDN cache headers |
| 26 | ETag | `examples/etag/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Automatic weak ETags |
| 27 | Map directive | `examples/map/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Regex header mapping |
| 28 | Split clients | `examples/split-clients/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | A/B testing |
| 29 | Mirror | `examples/mirror/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Traffic shadowing |
| 30 | Sub filter | `examples/sub-filter/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Response body rewriting |
| 31 | Real IP | `examples/real-ip/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | X-Forwarded-For extraction |
| 32 | Gunzip | `examples/gunzip/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Decompress for old clients |
| 33 | TLS ciphers | `examples/tls-ciphers/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Cipher suite configuration |

## Priority 6 — Observability Features

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 34 | Access logging | `examples/access-log/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | nginx-format request logs |
| 35 | Status endpoint | `examples/status/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | /status for monitoring |
| 36 | Error logging levels | `examples/error-log/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Debug/info/warn/error |

## Priority 7 — Full Stack

| # | Feature | Dir | Config | Before | After | Test | Verified | Logging | Notes |
|---|---------|-----|--------|--------|-------|------|----------|---------|-------|
| 37 | Production stack | `examples/production/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | All modules combined |
| 38 | API gateway | `examples/api-gateway/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Auth + rate limit + proxy |
| 39 | Multi-port | `examples/multi-port/` | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | ⬜ | Multiple listeners, protocols |

## Progress Summary

| Priority | Total | Done | % |
|----------|-------|------|---|
| P1 Core | 3 | 3 | 100% |
| P2 Protocol | 4 | 4 | 100% |
| P3 Security | 6 | 6 | 100% |
| P4 Proxy | 9 | 9 | 100% |
| P4 Proxy | 9 | 0 | 0% |
| P5 Advanced | 11 | 0 | 0% |
| P6 Observability | 3 | 0 | 0% |
| P7 Full Stack | 3 | 0 | 0% |
| **Total** | **39** | **22** | **56%** |
