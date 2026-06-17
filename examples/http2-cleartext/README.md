# HTTP/2 Cleartext (h2c) Example

HTTP/2 over plaintext without TLS. GoRe automatically enables h2c when TLS is not configured.

## How to run

```bash
gore -c gore.yaml &
./test.sh
```

## What it demonstrates

- HTTP/2 upgrade on cleartext connections
- Multiplexed streams over a single TCP connection
- Backward compatibility with HTTP/1.1 clients
