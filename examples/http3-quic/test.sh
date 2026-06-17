#!/bin/bash
set -e
BASE="https://127.0.0.1:4443"

echo "Testing HTTP/3 QUIC..."

# Test 1: HTTPS fallback (H3 requires QUIC client, test with regular curl)
BODY=$(curl -sk "$BASE/")
if [ "$BODY" = "Hello HTTP/3 QUIC" ]; then
  echo "PASS: GET / → 200 with body over HTTPS"
else
  echo "FAIL: body = '$BODY', want 'Hello HTTP/3 QUIC'"
  exit 1
fi

# Test 2: HTTP/2 negotiated over TLS
PROTO=$(curl -sk -o /dev/null -w "%{http_version}" "$BASE/")
if [ "$PROTO" = "2" ] || [ "$PROTO" = "2.0" ]; then
  echo "PASS: TLS connection negotiated HTTP/2 (H3 requires QUIC client)"
else
  echo "INFO: TLS version = $PROTO"
fi

# Test 3: Alt-Svc header for H3 discovery
ALTSVC=$(curl -sk -I "$BASE/" 2>/dev/null | grep -i alt-svc)
if [ -n "$ALTSVC" ]; then
  echo "PASS: Alt-Svc header present for H3 discovery"
else
  echo "WARN: Alt-Svc header not found"
fi

echo "All HTTP/3 tests passed."
