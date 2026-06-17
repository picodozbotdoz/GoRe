#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
kill $(cat /tmp/gore-example-prod.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-prod.pid
rm -rf "$DIR/site" "$DIR/cert.pem" "$DIR/key.pem"
echo "Production cleaned up."
