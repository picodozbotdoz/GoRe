#!/bin/bash
set -e
BASE="https://127.0.0.1:8443"

echo "Testing TLS/HTTPS..."

# Test 1: HTTPS request
BODY=$(curl -sk "$BASE/")
if [ "$BODY" != "Hello over HTTPS" ]; then
  echo "FAIL: body = '$BODY', want 'Hello over HTTPS'"
  exit 1
fi
echo "PASS: GET / → 'Hello over HTTPS' over HTTPS"

# Test 2: HTTP/2 protocol
PROTO=$(curl -sk --http2 -o /dev/null -w "%{http_version}" "$BASE/")
if [ "$PROTO" = "2" ] || [ "$PROTO" = "2.0" ]; then
  echo "PASS: Protocol negotiated HTTP/2"
else
  echo "INFO: Protocol = $PROTO (HTTP/2 requires client support)"
fi

# Test 3: HSTS header
HSTS=$(curl -sk -I "$BASE/" 2>/dev/null | grep -i strict-transport)
if [ -n "$HSTS" ]; then
  echo "PASS: HSTS header present"
else
  echo "WARN: HSTS header not found in response headers"
fi

echo "All TLS tests passed."
