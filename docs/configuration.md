# GoRe Configuration Reference

Complete reference for all GoRe configuration options. YAML format.

## Top-Level Structure

```yaml
worker_processes: auto          # informational only (single-process, multi-goroutine)
listen: []                      # list of listeners (see Listen Block)
http:                           # virtual server blocks
  server: []
upstreams: {}                   # named backend groups
modules: {}                     # global middleware configuration
```

## Listen Block

Each listener binds to an address and optionally enables TLS, HTTP/2, and HTTP/3.

```yaml
listen:
  - addr: ":80"                          # cleartext HTTP/1.1 + HTTP/2 (h2c)
  - addr: "0.0.0.0:443"                  # bind to all interfaces
    tls:
      cert: /etc/ssl/certs/server.pem    # PEM certificate
      key: /etc/ssl/private/server.key   # PEM private key
    http2:
      enabled: true
      max_concurrent_streams: 250        # default: 250
      max_frame_size: 1048576            # default: 1MB (1048576 bytes)
    http3:
      enabled: true                      # requires TLS
      max_streams: 100                   # default: 100
      idle_timeout: 30                   # seconds, default: 30
```

### Protocol Behavior

| Listener Config | Result |
|---|---|
| `addr` only (no TLS) | HTTP/1.1 + HTTP/2 cleartext (h2c) |
| `addr` + `tls` | HTTP/1.1 + HTTP/2 over TLS (ALPN) |
| `addr` + `tls` + `http3.enabled` | Above + HTTP/3 on UDP same port (Alt-Svc advertised) |

## HTTP Server / Locations

```yaml
http:
  server:
    - name: example.com          # server name (informational, not used for routing)
      locations:
        - path: /                # URL prefix to match
          return: "200 'OK'"     # inline response
        - path: /static/
          root: /var/www/html    # serve files from filesystem
          autoindex: true        # directory listing
        - path: /api/
          proxy:
            upstream: backend    # proxy to named upstream
```

### Location Fields

| Field | Type | Description |
|---|---|---|
| `path` | string | URL prefix match (radix trie). `/api/` matches `/api/anything`. `/api` matches only `/api`. |
| `root` | string | Filesystem root for static files |
| `proxy.upstream` | string | Name of upstream group |
| `return` | string | Inline response (see Return Syntax below) |
| `autoindex` | bool | Enable directory listing (default: false) |

### Return Syntax

| Value | Behavior |
|---|---|
| `"200"` | 200 OK, empty body |
| `"200 'Hello World'"` | 200 OK, body is `Hello World` (quotes stripped) |
| `"301"` | 301 Moved Permanently, no Location header |
| `"301 /new-path"` | 301 with `Location: /new-path` |
| `"403"` | 403 Forbidden |
| `"405"` | 405 Method Not Allowed |
| `"/other-path"` (no numeric prefix) | 301 redirect to that path |

### Location Priority

Only the first matching handler is used: `Proxy` > `Root` > `Return` > 404.

## Upstreams

```yaml
upstreams:
  backend:
    strategy: round-robin        # load balancing (currently only round-robin)
    servers:
      - addr: 127.0.0.1:9001
        weight: 1                # parsed but not used in scheduling
      - addr: 127.0.0.1:9002
        weight: 2
    keepalive: 32                # parsed but not used in transport config
```

### Upstream Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `strategy` | string | `"round-robin"` | Load balancing algorithm |
| `servers` | []UpstreamServer | required | Backend server list |
| `keepalive` | int | — | Parsed but unused |

### UpstreamServer Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `addr` | string | required | Backend address `host:port` |
| `weight` | int | 1 | Parsed but unused in scheduling |

## Modules

All modules are optional. When configured, they wrap every location handler.

### Middleware Execution Order

```
Request  → Access → RateLimit → Handler → Headers → Gzip → Response
```

Access control is outermost (short-circuits rejected requests). Gzip is innermost for responses (compresses final output).

### Gzip

```yaml
modules:
  gzip:
    enabled: true
    level: 6                     # compression level 1-9 (0 = default)
    types:                       # MIME type prefixes to compress
      - text/plain
      - text/html
      - application/json
      - application/javascript
```

Behavior: Checks `Accept-Encoding: gzip` header. Compresses response only if `Content-Type` matches a type in the list (or list is empty = compress all). Uses `sync.Pool` for writer reuse.

### Headers

```yaml
modules:
  headers:
    add:
      X-Frame-Options: DENY
      X-Content-Type-Options: nosniff
      Server: GoRe/1.0
    remove:
      - X-Powered-By
      - Server
```

### Rate Limiting

```yaml
modules:
  rate_limit:
    zone: global                 # zone name (parsed but unused)
    rate: "100/s"                # tokens per second (format: N/s or N/m)
    burst: 200                   # token bucket size (default: same as rate)
```

Behavior: Token bucket per client IP. Returns HTTP 429 with `Retry-After: 1` when exceeded.

Rate format: `"N/s"` (per second) or `"N/m"` (per minute). Integer values only (no decimals).

### Access Control

```yaml
modules:
  access:
    rules:
      - allow: 192.168.0.0/16    # allow private network
      - allow: 10.0.0.0/8        # allow internal network
      - deny: all                # deny everything else
```

Rules are processed in order, first match wins. Special values:
- `"all"` → `0.0.0.0/0` (matches everything)
- Plain IPs like `"10.0.0.1"` → auto-converted to `/32`
- If no rules match, request is allowed (open policy)

## Global Defaults

| Setting | Default | Notes |
|---|---|---|
| `worker_processes` | `"auto"` | Informational, not functional |
| HTTP listen address | `":80"` | When no listen block provided |
| HTTP2 max_concurrent_streams | 250 | Per-connection stream limit |
| HTTP2 max_frame_size | 1048576 | 1 MiB |
| HTTP3 max_streams | 100 | Per-connection QUIC streams |
| HTTP3 idle_timeout | 30s | QUIC idle timeout |
| ReadTimeout | 30s | Hardcoded |
| WriteTimeout | 30s | Hardcoded |
| IdleTimeout | 120s | Hardcoded |
| Shutdown timeout | 5s | Graceful shutdown |
| Proxy dial timeout | 5s | Upstream connection |
| Proxy idle timeout | 90s | Upstream keepalive |
| Proxy max idle conns | 100 | Connection pool |
| Proxy max idle conns/host | 10 | Per-host pool |

## Signal Handling

| Signal | Action |
|---|---|
| SIGINT | Graceful shutdown |
| SIGTERM | Graceful shutdown |
| SIGQUIT | Graceful shutdown |

Shutdown order: HTTP/3 servers first, then HTTP/1+2 servers.

## CLI Usage

```
gore -c config.yaml
```

| Flag | Default | Description |
|---|---|---|
| `-c` | `gore.yaml` | Config file path |
