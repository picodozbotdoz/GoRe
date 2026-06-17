# Reverse Proxy Example

Proxies requests to a backend server with round-robin load balancing.

## How to run

```bash
./before.sh        # starts mock backend on :9001
gore -c gore.yaml &
./test.sh
./after.sh
```

## What it demonstrates

- Reverse proxy to upstream backend
- Round-robin load balancing (2 backends)
- X-Forwarded-For, X-Forwarded-Host, X-Forwarded-Proto headers
- Health check on backend
