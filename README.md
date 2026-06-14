# GoRe

A clean reimplementation of nginx in Go with modern idioms and middleware architecture.

## Features

- **Reverse Proxy** with load balancing (round-robin, least-conn)
- **Static File Serving** with directory listing
- **Gzip Compression** with type filtering
- **Header Manipulation** (add/remove)
- **IP Access Control** (allow/deny by CIDR)
- **Rate Limiting** (token bucket per-IP)
- **URL Rewriting** with redirects
- **TLS/SSL** support
- **YAML/TOML** configuration

## Architecture

```
gore/
├── cmd/gore/          # Entry point
├── internal/
│   ├── config/        # YAML config parsing
│   ├── router/        # Radix tree location matching
│   ├── proxy/         # Reverse proxy + load balancing
│   ├── modules/       # Middleware chain
│   │   ├── access/    # IP allow/deny
│   │   ├── gzip/      # Compression
│   │   ├── headers/   # Header manipulation
│   │   ├── ratelimit/ # Rate limiting
│   │   └── static/    # Static file serving
│   └── server/        # HTTP server + TLS
└── docs/              # Documentation
```

## Quick Start

```bash
# Build
go build -o gore ./cmd/gore

# Run with config
./gore -c config.yaml
```

## Configuration

```yaml
worker_processes: auto
listen:
  - addr: ":80"
  - addr: ":443"
    tls:
      cert: /etc/ssl/cert.pem
      key: /etc/ssl/key.pem

http:
  server:
    - name: example.com
      locations:
        - path: /static/
          root: /var/www
        - path: /api/
          proxy:
            upstream: backend
        - path: /
          return: "https://example.com"

upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:8080
      - addr: 127.0.0.1:8081

modules:
  gzip:
    enabled: true
    level: 6
  access:
    rules:
      - allow: 192.168.0.0/16
      - deny: all
  headers:
    add:
      X-Frame-Options: DENY
      X-Content-Type-Options: nosniff
  rate_limit:
    rate: 10r/s
    burst: 20
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific tests
go test ./internal/modules/access/ -v
go test ./internal/proxy/ -v
```

## Nginx Compatibility

GoRe implements core nginx features with compatible behavior:

| Feature | Status |
|---------|--------|
| IP allow/deny | ✅ |
| Gzip compression | ✅ |
| Custom headers | ✅ |
| Reverse proxy | ✅ |
| Static files | ✅ |
| URL rewriting | ✅ |
| Rate limiting | ✅ |
| Directory listing | ✅ |
| TLS/SSL | ✅ |
| Load balancing | ✅ |
| HTTP/2 | ❌ |
| HTTP/3 | ❌ |
| Stream proxy | ❌ |

## License

MIT
