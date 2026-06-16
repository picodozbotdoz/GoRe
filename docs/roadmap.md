# GoRe Roadmap

Implementation status of all features, compared against nginx. Last updated: 2026-06-15.

## Status Legend

- ✅ Implemented — working, tested
- 🔧 Partial — partially working or stub
- 📋 Planned — designed but not started
- ❌ Not started
- ⛔ Out of scope — will not implement

---

## Implemented Features

| # | Feature | Status | Tests | Notes |
|---|---------|--------|-------|-------|
| 1 | HTTP/1.1 | ✅ | ✅ | Cleartext and TLS |
| 2 | HTTP/2 cleartext (h2c) | ✅ | ✅ | `golang.org/x/net/http2/h2c` |
| 3 | HTTP/2 over TLS | ✅ | ✅ | ALPN negotiation via `http2.ConfigureServer` |
| 4 | HTTP/3 (QUIC) | ✅ | ✅ | `quic-go/http3`, Alt-Svc header auto-injected |
| 5 | TLS/HTTPS | ✅ | ✅ | PEM cert+key, ALPN h2/http1.1 |
| 6 | Static file serving | ✅ | ✅ | Path prefix, index.html, range requests |
| 7 | Directory listing (autoindex) | ✅ | ✅ | HTML output |
| 8 | Path traversal protection | ✅ | ✅ | Blocks `../../../` escape |
| 9 | Reverse proxy | ✅ | ✅ | `httputil.RoundTripper`, X-Forwarded-* headers |
| 10 | Round-robin load balancing | ✅ | ✅ | Atomic counter, unhealthy server skip |
| 11 | Inline return (200/301/403/405) | ✅ | ✅ | Status code + body, Location header for 3xx |
| 12 | Gzip compression | ✅ | ✅ | Content-Type filtering, sync.Pool reuse |
| 13 | Response headers (add/remove) | ✅ | ✅ | Global middleware |
| 14 | Rate limiting (token bucket) | ✅ | ✅ | Per-IP, N/s or N/m format, 429 response |
| 15 | Access control (allow/deny) | ✅ | ✅ | CIDR rules, order matters, `"all"` = 0.0.0.0/0 |
| 16 | YAML configuration | ✅ | ✅ | Full config loading with defaults |
| 17 | Graceful shutdown | ✅ | ✅ | SIGINT/SIGTERM/SIGQUIT, 5s timeout |
| 18 | Multi-listener | ✅ | ✅ | Multiple ports, per-listener protocol config |
| 19 | Request body forwarding | ✅ | ✅ | POST body proxied to upstream |
| 20 | Connection reuse (keepalive) | ✅ | ✅ | HTTP/1.1 and HTTP/2 connection pooling |

---

## Missing Features — High Priority

| # | Feature | nginx Equivalent | Status | Effort | Description |
|---|---------|-----------------|--------|--------|-------------|
| 21 | Access logging | `access_log` | ❌ | Small | Log request method, path, status, latency, client IP |
| 22 | Error logging levels | `error_log` | 🔧 | Small | Replace `log.Printf` with leveled logger |
| 23 | URL rewrite rules | `rewrite` | ❌ | Medium | Regex-based URL rewriting with flags (permanent/temporary) |
| 24 | `try_files` | `try_files` | ❌ | Medium | Fallback chain: `$uri $uri/ =404` |
| 25 | WebSocket upgrade | `proxy_pass` + Upgrade | ❌ | Medium | Handle `Upgrade: websocket` header, bidirectional streaming |
| 26 | Proxy timeouts | `proxy_connect_timeout`, `proxy_read_timeout` | ❌ | Small | Per-location timeout configuration |
| 27 | Proxy buffering | `proxy_buffering`, `proxy_buffer_size` | ❌ | Small | Buffer upstream response before sending to client |
| 28 | Proxy retry | `proxy_next_upstream` | ❌ | Medium | Retry on next upstream on failure |
| 29 | Active health checks | `health_check` | ❌ | Large | Periodic backend health probing |
| 30 | Upstream keepalive | `keepalive` in upstream | ❌ | Small | Configurable connection pool to backends |
| 31 | Request body size limit | `client_max_body_size` | ❌ | Small | Reject requests larger than limit |
| 32 | Concurrent connection limit | `limit_conn` | ❌ | Medium | Per-IP connection count limit |
| 33 | Request header manipulation | `proxy_set_header` | ❌ | Small | Set/add/remove headers on proxied request |
| 34 | ETag generation | Automatic | ❌ | Small | Generate ETag for static files |
| 35 | Cache-Control headers | `expires` | ❌ | Small | Configurable cache headers per location |
| 36 | HTTP Basic Auth | `auth_basic` | ❌ | Medium | Username/password authentication |
| 37 | Auth subrequest | `auth_request` | ❌ | Large | Delegate auth to external service |
| 38 | TLS cipher config | `ssl_ciphers` | ❌ | Small | Configurable cipher suites |
| 39 | Stub status endpoint | `stub_status` | ❌ | Small | `/nginx_status` equivalent for monitoring |

---

## Missing Features — Medium Priority

| # | Feature | nginx Equivalent | Status | Effort | Description |
|---|---------|-----------------|--------|--------|-------------|
| 40 | Content replacement | `sub_filter` | ❌ | Medium | Replace strings in response body |
| 41 | Server Side Includes | `ssi` | ❌ | Large | Parse SSI directives in HTML |
| 42 | Proxy cache | `proxy_cache` | ❌ | Large | Cache upstream responses on disk |
| 43 | Brotli compression | `brotli` | ❌ | Medium | Brotli compression middleware |
| 44 | Gunzip (decompress) | `gunzip` | ❌ | Small | Decompress upstream for old clients |
| 45 | WebDAV | `dav_methods` | ❌ | Large | PUT, DELETE, MKCOL, COPY, MOVE |
| 46 | GeoIP | `geoip` | ❌ | Medium | IP-to-country lookup |
| 47 | Map directive | `map` | ❌ | Medium | Variable mapping based on conditions |
| 48 | Split clients | `split_clients` | ❌ | Small | A/B testing by IP/header hash |
| 49 | Traffic mirroring | `mirror` | ❌ | Medium | Clone requests to shadow backend |
| 50 | Real IP extraction | `real_ip` | ❌ | Small | Parse X-Forwarded-For for client IP |
| 51 | TCP/UDP stream proxy | `stream` | ❌ | Large | L4 proxy (no HTTP parsing) |
| 52 | Mail proxy | `mail` | ❌ | Large | IMAP/POP3/SMTP proxy |

---

## Missing Features — Low Priority

| # | Feature | nginx Equivalent | Status | Effort | Description |
|---|---------|-----------------|--------|--------|-------------|
| 53 | Perl/Lua scripting | `perl`, `ngx_http_lua` | ⛔ | — | Out of scope for GoRe |
| 54 | Image processing | `image_filter` | ⛔ | — | Resize/rotate/watermark images |
| 55 | FLV/MP4 streaming | `flv`, `mp4` | ⛔ | — | Byte-range media streaming |
| 56 | Autoindex JSON/size/date | `autoindex_format` | ❌ | Small | Enhanced directory listing |
| 57 | Open file cache | `open_file_cache` | ❌ | Medium | FD + stat caching |
| 58 | Worker CPU affinity | `worker_cpu_affinity` | ⛔ | — | Single-process model, N/A |
| 59 | Conditional logging | `access_log ... if=` | ❌ | Small | Log based on condition |

---

## Statistics

| Category | Total | Implemented | Partial | Planned | Out of Scope |
|----------|-------|-------------|---------|---------|--------------|
| Core HTTP | 7 | 7 | 0 | 0 | 0 |
| Proxy | 9 | 2 | 0 | 7 | 0 |
| Static | 5 | 3 | 0 | 2 | 0 |
| Security | 5 | 1 | 0 | 4 | 0 |
| Compression | 4 | 1 | 0 | 3 | 0 |
| Headers | 3 | 2 | 0 | 1 | 0 |
| Limits | 3 | 1 | 0 | 2 | 0 |
| Logging | 4 | 0 | 1 | 3 | 0 |
| TLS | 3 | 1 | 0 | 2 | 0 |
| Rewrite/Routing | 3 | 0 | 0 | 3 | 0 |
| Advanced | 13 | 0 | 0 | 9 | 4 |
| **Total** | **59** | **18** | **1** | **37** | **4** |

**Completion: 31% (18/59 features)**
**Remaining: 37 features to implement, 4 out of scope**

---

## Priority Roadmap

### Phase 1 — Production Essentials (next)

| # | Feature | Why |
|---|---------|-----|
| 21 | Access logging | Every production server needs request logs |
| 26 | Proxy timeouts | Without this, slow backends hang the proxy |
| 31 | Request body size limit | Prevents abuse (large POST uploads) |
| 33 | Request header manipulation | Needed for custom auth headers, tracing |
| 39 | Stub status endpoint | Needed for monitoring (Prometheus scrape) |
| 22 | Error logging levels | Debug/Info/Warn/Error separation |

### Phase 2 — Core Completeness

| # | Feature | Why |
|---|---------|-----|
| 23 | URL rewrite rules | Complex routing requires regex rewrites |
| 25 | WebSocket upgrade | Modern apps need real-time communication |
| 30 | Upstream keepalive | Connection pooling reduces latency |
| 32 | Concurrent connection limit | DDoS protection beyond rate limiting |
| 34 | ETag generation | Browser caching for static files |
| 35 | Cache-Control headers | CDN and browser cache control |
| 50 | Real IP extraction | Accurate client IP behind load balancers |

### Phase 3 — Authentication & Security

| # | Feature | Why |
|---|---------|-----|
| 36 | HTTP Basic Auth | Simple auth for admin panels |
| 24 | try_files | Elegant fallback chain for SPAs |
| 27 | Proxy buffering | Control memory usage for large responses |
| 28 | Proxy retry | Improve availability with automatic failover |
| 37 | Auth subrequest | External auth service delegation |
| 38 | TLS cipher config | Security hardening |

### Phase 4 — Advanced

| # | Feature | Why |
|---|---------|-----|
| 29 | Active health checks | Automatic backend failover |
| 40 | Content replacement | Response body rewriting |
| 43 | Brotli compression | Better compression than gzip |
| 47 | Map directive | Variable-based config flexibility |
| 48 | Split clients | A/B testing |
| 49 | Traffic mirroring | Safe deployment testing |
