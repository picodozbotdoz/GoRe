# TLS/HTTPS Example

Serves over HTTPS with TLS 1.3 and HTTP/2.

## How to run

```bash
./before.sh        # generates self-signed cert
gore -c gore.yaml &
./test.sh
./after.sh
```

## What it demonstrates

- TLS termination with self-signed certificate
- HTTP/2 ALPN negotiation (h2 + http/1.1)
- HSTS header for browser security
