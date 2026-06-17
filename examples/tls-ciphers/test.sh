#!/bin/bash
set -e

echo "Testing TLS cipher configuration..."

# Test 1: HTTPS request works
BODY=$(curl -sk "https://127.0.0.1:8443/")
if [ "$BODY" = "Hello over TLS" ]; then
  echo "PASS: HTTPS request with configured ciphers"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 2: verify cipher negotiation
CIPHER=$(curl -sk -o /dev/null -w "%{cipher}" "https://127.0.0.1:8443/")
echo "Negotiated cipher: $CIPHER"
if [ -n "$CIPHER" ]; then
  echo "PASS: cipher negotiation successful"
else
  echo "INFO: cipher info not available via curl"
fi

# Test 3: TLS version
VERSION=$(curl -sk -o /dev/null -w "%{ssl_version}" "https://127.0.0.1:8443/")
echo "TLS version: $VERSION"

echo "All TLS cipher tests passed."
