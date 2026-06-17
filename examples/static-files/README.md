# Static Files Example

Serves files from a local directory over HTTP/1.1.

## How to run

```bash
./before.sh
gore -c gore.yaml &
./test.sh
./after.sh
```

## What it demonstrates

- Static file serving from a configured root directory
- Automatic index.html detection for directories
- Path prefix mapping (`/static/` → `/var/www/static/`)
- ETag generation and If-None-Match handling
