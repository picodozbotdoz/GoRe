# Configuration

GoRe uses YAML configuration files.

## Structure

```yaml
worker_processes: auto    # Number of worker processes (auto = NumCPU)
listen:                   # List of listening addresses
  - addr: ":80"
  - addr: ":443"
    tls:
      cert: /path/to/cert.pem
      key: /path/to/key.pem

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
    strategy: round-robin  # round-robin | least-conn
    servers:
      - addr: 127.0.0.1:8080
        weight: 1
      - addr: 127.0.0.1:8081
        weight: 2
    keepalive: 32

modules:
  gzip:
    enabled: true
    level: 6
    types:
      - text/html
      - text/plain
      - application/json
  access:
    rules:
      - allow: 192.168.0.0/16
      - deny: all
  headers:
    add:
      X-Frame-Options: DENY
      X-Content-Type-Options: nosniff
    remove:
      - Server
  rate_limit:
    rate: 10r/s
    burst: 20
```

## Options

### listen

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| addr | string | `:80` | Listen address (host:port) |
| tls.cert | string | - | TLS certificate path |
| tls.key | string | - | TLS private key path |

### upstreams

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| strategy | string | round-robin | Load balancing strategy |
| servers[].addr | string | - | Backend address (host:port) |
| servers[].weight | int | 1 | Server weight |
| keepalive | int | 0 | Max keepalive connections |

### modules.gzip

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| enabled | bool | false | Enable compression |
| level | int | 6 | Compression level (1-9) |
| types | []string | all | MIME types to compress |

### modules.access

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| rules[].allow | string | - | CIDR or IP to allow |
| rules[].deny | string | - | CIDR or IP to deny |

### modules.headers

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| add | map | - | Headers to add |
| remove | []string | - | Headers to remove |

### modules.rate_limit

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| rate | string | 10/s | Requests per second (e.g., 10/s, 100/m) |
| burst | int | rate | Burst size |
