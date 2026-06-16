# GoRe Feature Reference

Complete catalog of GoRe capabilities with working configuration examples.

## Table of Contents

1. [Basic HTTP/1.1 Server](#1-basic-http11-server)
2. [Static File Serving](#2-static-file-serving)
3. [Directory Listing (Autoindex)](#3-directory-listing-autoindex)
4. [Reverse Proxy](#4-reverse-proxy)
5. [Load Balancing](#5-load-balancing)
6. [Inline Responses (Return)](#6-inline-responses-return)
7. [TLS / HTTPS](#7-tls--https)
8. [HTTP/2 Cleartext (h2c)](#8-http2-cleartext-h2c)
9. [HTTP/2 over TLS](#9-http2-over-tls)
10. [HTTP/3 (QUIC)](#10-http3-quic)
11. [Gzip Compression](#11-gzip-compression)
12. [Response Headers](#12-response-headers)
13. [Rate Limiting](#13-rate-limiting)
14. [Access Control](#14-access-control)
15. [Complete Production Config](#15-complete-production-config)
16. [Multi-Listener Config](#16-multi-listener-config)
17. [API Gateway Pattern](#17-api-gateway-pattern)

---

## 1. Basic HTTP/1.1 Server

Minimal server serving static files.

```yaml
# examples/01-basic-http.yaml
listen:
  - addr: ":8080"

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

Run: `gore -c examples/01-basic-http.yaml`

---

## 2. Static File Serving

Serve files from a directory with path prefix mapping.

```yaml
# examples/02-static-files.yaml
listen:
  - addr: ":8080"

http:
  server:
    - locations:
        - path: /
          root: /var/www/public
        - path: /assets/
          root: /var/www/static
        - path: /downloads/
          root: /data/files
```

- `GET /index.html` → serves `/var/www/public/index.html`
- `GET /assets/style.css` → serves `/var/www/static/style.css`
- `GET /downloads/report.pdf` → serves `/data/files/report.pdf`

Path traversal (`../../../etc/passwd`) is blocked (returns 403).

---

## 3. Directory Listing (Autoindex)

Enable HTML directory listing when no `index.html` exists.

```yaml
# examples/03-autoindex.yaml
listen:
  - addr: ":8080"

http:
  server:
    - locations:
        - path: /
          root: /var/www/files
          autoindex: true
```

---

## 4. Reverse Proxy

Proxy requests to backend services.

```yaml
# examples/04-reverse-proxy.yaml
listen:
  - addr: ":8080"

upstreams:
  api:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:3000
      - addr: 127.0.0.1:3001

http:
  server:
    - locations:
        - path: /api/
          proxy:
            upstream: api
        - path: /
          root: /var/www/static
```

- `GET /api/users` → proxied to `127.0.0.1:3000/api/users` or `:3001`
- `GET /index.html` → served from static files

Forwarded headers: `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`.

---

## 5. Load Balancing

Round-robin across multiple backends.

```yaml
# examples/05-load-balancing.yaml
listen:
  - addr: ":8080"

upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 10.0.0.1:8080
        weight: 1
      - addr: 10.0.0.2:8080
        weight: 1
      - addr: 10.0.0.3:8080
        weight: 1

http:
  server:
    - locations:
        - path: /
          proxy:
            upstream: backend
```

Note: `weight` is parsed but currently not used in scheduling. All backends receive equal traffic.

---

## 6. Inline Responses (Return)

Return status codes and bodies directly without a backend.

```yaml
# examples/06-return.yaml
listen:
  - addr: ":8080"

http:
  server:
    - locations:
        # Simple redirect
        - path: /old-page
          return: "301 /new-page"

        # Status with body
        - path: /health
          return: "200 'OK'"

        # Error responses
        - path: /forbidden
          return: "403"
        - path: /maintenance
          return: "503 'Service Unavailable'"

        # 404 for unknown paths
        - path: /admin
          return: "404"
```

---

## 7. TLS / HTTPS

Serve over TLS with automatic HTTP/2 negotiation.

```yaml
# examples/07-tls.yaml
listen:
  - addr: "0.0.0.0:443"
    tls:
      cert: /etc/letsencrypt/live/example.com/fullchain.pem
      key: /etc/letsencrypt/live/example.com/privkey.pem

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

ALPN negotiation advertises `h2` and `http/1.1`. Clients automatically negotiate the best protocol.

---

## 8. HTTP/2 Cleartext (h2c)

HTTP/2 over plaintext (no TLS). Useful for internal services.

```yaml
# examples/08-h2c.yaml
listen:
  - addr: ":8080"

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

When TLS is not configured, GoRe automatically enables h2c. Clients use HTTP/2 Upgrade mechanism or prior knowledge.

---

## 9. HTTP/2 over TLS

HTTP/2 with TLS encryption (standard deployment).

```yaml
# examples/09-h2-tls.yaml
listen:
  - addr: "0.0.0.0:443"
    tls:
      cert: /etc/ssl/certs/server.pem
      key: /etc/ssl/private/server.key
    http2:
      enabled: true
      max_concurrent_streams: 250
      max_frame_size: 1048576

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

---

## 10. HTTP/3 (QUIC)

HTTP/3 over QUIC (UDP). Requires TLS. Advertises via `Alt-Svc` header on HTTP/2 responses.

```yaml
# examples/10-h3.yaml
listen:
  - addr: "0.0.0.0:443"
    tls:
      cert: /etc/ssl/certs/server.pem
      key: /etc/ssl/private/server.key
    http2:
      enabled: true
    http3:
      enabled: true
      max_streams: 100
      idle_timeout: 30

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

The server listens on TCP:443 (HTTP/2) and UDP:443 (HTTP/3). HTTP/2 responses include `Alt-Svc: h3=":443"; ma=86400` for client discovery.

---

## 11. Gzip Compression

Compress responses based on MIME type.

```yaml
# examples/11-gzip.yaml
listen:
  - addr: ":8080"

modules:
  gzip:
    enabled: true
    level: 6
    types:
      - text/plain
      - text/html
      - text/css
      - text/javascript
      - application/json
      - application/javascript
      - application/xml

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

Requires `Accept-Encoding: gzip` from client. Responses are compressed only if `Content-Type` matches one of the configured types.

---

## 12. Response Headers

Add or remove response headers globally.

```yaml
# examples/12-headers.yaml
listen:
  - addr: ":8080"

modules:
  headers:
    add:
      X-Frame-Options: DENY
      X-Content-Type-Options: nosniff
      X-XSS-Protection: "1; mode=block"
      Strict-Transport-Security: "max-age=31536000; includeSubDomains"
      Cache-Control: "public, max-age=3600"
    remove:
      - X-Powered-By
      - Server

http:
  server:
    - locations:
        - path: /
          root: /var/www/html
```

---

## 13. Rate Limiting

Per-IP token bucket rate limiting.

```yaml
# examples/13-ratelimit.yaml
listen:
  - addr: ":8080"

modules:
  rate_limit:
    zone: api
    rate: "100/s"
    burst: 200

http:
  server:
    - locations:
        - path: /api/
          proxy:
            upstream: backend
        - path: /
          root: /var/www/html
```

Rate format: `"N/s"` (per second) or `"N/m"` (per minute). Returns HTTP 429 with `Retry-After: 1` when exceeded. Buckets are per client IP.

---

## 14. Access Control

IP-based allow/deny rules. Processed in order, first match wins.

```yaml
# examples/14-access-control.yaml
listen:
  - addr: ":8080"

modules:
  access:
    rules:
      - allow: 192.168.0.0/16     # allow private network
      - allow: 10.0.0.0/8         # allow internal network
      - deny: all                 # deny everything else

http:
  server:
    - locations:
        - path: /admin
          return: "200 'Admin Panel'"
        - path: /
          root: /var/www/html
```

Special values: `"all"` = `0.0.0.0/0`. Plain IPs auto-converted to `/32`. If no rules match, request is allowed.

---

## 15. Complete Production Config

All features combined for a production-like setup.

```yaml
# examples/15-production.yaml
listen:
  - addr: "0.0.0.0:80"
  - addr: "0.0.0.0:443"
    tls:
      cert: /etc/letsencrypt/live/example.com/fullchain.pem
      key: /etc/letsencrypt/live/example.com/privkey.pem
    http2:
      enabled: true
      max_concurrent_streams: 250
    http3:
      enabled: true
      max_streams: 100

upstreams:
  app:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:3000
      - addr: 127.0.0.1:3001

modules:
  gzip:
    enabled: true
    level: 6
    types:
      - text/
      - application/json
      - application/javascript
  headers:
    add:
      X-Frame-Options: DENY
      X-Content-Type-Options: nosniff
      Strict-Transport-Security: "max-age=31536000"
    remove:
      - X-Powered-By
  rate_limit:
    zone: global
    rate: "1000/s"
    burst: 2000
  access:
    rules:
      - deny: all

http:
  server:
    - locations:
        - path: /api/
          proxy:
            upstream: app
        - path: /static/
          root: /var/www/static
        - path: /health
          return: "200 'OK'"
        - path: /
          root: /var/www/html
```

---

## 16. Multi-Listener Config

Multiple ports with different configurations.

```yaml
# examples/16-multi-listener.yaml
listen:
  - addr: ":80"                           # HTTP redirect
  - addr: ":8080"                         # internal API (h2c)
  - addr: ":443"                          # public HTTPS
    tls:
      cert: /etc/ssl/certs/server.pem
      key: /etc/ssl/private/server.key
    http2:
      enabled: true
    http3:
      enabled: true

http:
  server:
    - locations:
        - path: /health
          return: "200 'OK'"
        - path: /api/
          proxy:
            upstream: backend
        - path: /
          root: /var/www/html

upstreams:
  backend:
    strategy: round-robin
    servers:
      - addr: 127.0.0.1:3000
```

---

## 17. API Gateway Pattern

Rate limiting + access control + reverse proxy for an API.

```yaml
# examples/17-api-gateway.yaml
listen:
  - addr: "0.0.0.0:443"
    tls:
      cert: /etc/ssl/certs/server.pem
      key: /etc/ssl/private/server.key
    http2:
      enabled: true

upstreams:
  users-service:
    strategy: round-robin
    servers:
      - addr: 10.0.1.10:8080
      - addr: 10.0.1.11:8080
  orders-service:
    strategy: round-robin
    servers:
      - addr: 10.0.2.10:8080
      - addr: 10.0.2.11:8080

modules:
  gzip:
    enabled: true
    types:
      - application/json
  headers:
    add:
      X-Request-ID: ""          # placeholder
      X-Content-Type-Options: nosniff
    remove:
      - Server
  rate_limit:
    zone: api
    rate: "500/s"
    burst: 1000
  access:
    rules:
      - allow: 10.0.0.0/8
      - allow: 172.16.0.0/12
      - deny: all

http:
  server:
    - locations:
        - path: /api/users/
          proxy:
            upstream: users-service
        - path: /api/orders/
          proxy:
            upstream: orders-service
        - path: /health
          return: "200 'OK'"
        - path: /
          return: "404"
```
