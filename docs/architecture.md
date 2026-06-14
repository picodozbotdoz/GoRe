# Architecture

GoRe follows a middleware-based architecture inspired by nginx's filter chain.

## Request Flow

```
Client Request
     │
     ▼
net/http.Server
     │
     ▼
Router (Radix Tree)
     │
     ▼
Middleware Chain
  ├── Access Control → 403 if denied
  ├── Rate Limit → 429 if exceeded
  ├── Headers → add/remove
  ├── Gzip → compress response
  ├── Static → serve file
  └── Proxy → forward to upstream
     │
     ▼
Response to Client
```

## Core Components

### Router (`internal/router/`)

Radix tree (htrie) for O(log n) location matching.

```go
router := router.NewRouter()
router.AddRoute("/api/", apiHandler)
router.AddRoute("/static/", staticHandler)
router.ServeHTTP(w, r)
```

**Features:**
- Exact matching: `/api` matches `/api`
- Prefix matching: `/static/` matches `/static/file.txt`
- No regex overhead for common patterns

### Proxy (`internal/proxy/`)

Reverse proxy with connection pooling.

```go
upstream := proxy.NewUpstream("backend", servers, "round-robin")
upstream.ServeHTTP(w, r)
```

**Features:**
- `httputil.ReverseProxy` foundation
- Connection pooling via `http.Transport`
- Load balancing strategies
- Automatic retry on failure

### Modules (`internal/modules/`)

Middleware chain builder.

```go
handler := modules.BuildChain(&cfg.Modules, finalHandler)
```

**Pattern:**
```go
func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Before
        next.ServeHTTP(w, r)
        // After
    })
}
```

### Config (`internal/config/`)

YAML configuration parser.

```go
cfg, err := config.Load("config.yaml")
```

**Features:**
- Struct-based parsing
- Default values
- Validation

## Data Structures

### ngx_listening_t equivalent

```go
type Listen struct {
    Addr string
    TLS  *TLS
}
```

### ngx_connection_t equivalent

Go's `net.Conn` handles connection state.

### ngx_http_request_t equivalent

Go's `http.Request` + `http.ResponseWriter`.

## Performance Considerations

| Aspect | nginx | GoRe |
|--------|-------|------|
| Event loop | epoll/kqueue | goroutine-per-connection |
| Memory | Zero-copy | GC-managed |
| TLS | OpenSSL | crypto/tls |
| Config parsing | Custom | YAML library |

**Trade-offs:**
- GoRe: Simpler code, GC overhead
- nginx: Maximum performance, complex code
