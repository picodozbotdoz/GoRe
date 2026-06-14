# Modules

GoRe implements nginx-compatible modules as Go middleware.

## Module Chain

Modules are executed in order:
1. Access Control
2. Rate Limiting
3. Headers
4. Gzip Compression

## Access Control

IP-based allow/deny rules.

```yaml
modules:
  access:
    rules:
      - allow: 192.168.0.0/16
      - deny: all
```

**Behavior:**
- Rules processed in order
- First matching rule determines action
- Default: allow if no rules match

## Gzip Compression

Compresses responses based on Content-Type.

```yaml
modules:
  gzip:
    enabled: true
    level: 6
    types:
      - text/html
      - text/plain
      - application/json
```

**Behavior:**
- Only compresses if client sends `Accept-Encoding: gzip`
- Filters by Content-Type
- Recycles gzip writers via sync.Pool

## Headers

Add or remove response headers.

```yaml
modules:
  headers:
    add:
      X-Frame-Options: DENY
      X-Content-Type-Options: nosniff
    remove:
      - Server
      - X-Powered-By
```

**Behavior:**
- Headers added/removed before response sent
- Works with any response status

## Rate Limiting

Token bucket rate limiting per IP.

```yaml
modules:
  rate_limit:
    rate: 10r/s
    burst: 20
```

**Rate formats:**
- `10/s` - 10 requests per second
- `100/m` - 100 requests per minute
- `1000/h` - 1000 requests per hour

**Behavior:**
- Tokens refill at configured rate
- Burst allows temporary spikes
- Returns 429 with `Retry-After` header

## Static Files

Serves files from a directory.

```yaml
http:
  server:
    - locations:
        - path: /static/
          root: /var/www
          autoindex: false
```

**Behavior:**
- Path prefix stripped before file lookup
- Directory traversal blocked
- Optional directory listing (autoindex)

## Reverse Proxy

Proxies requests to upstream servers.

```yaml
upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:8080
      - addr: 127.0.0.1:8081

http:
  server:
    - locations:
        - path: /api/
          proxy:
            upstream: backend
```

**Strategies:**
- `round-robin` - Distribute evenly
- `least-conn` - Fewest active connections

## Custom Modules

Implement the middleware interface:

```go
type Middleware func(http.Handler) http.Handler

func MyMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Before
        next.ServeHTTP(w, r)
        // After
    })
}
```
