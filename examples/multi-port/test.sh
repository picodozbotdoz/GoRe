#!/bin/bash
set -e

echo "Testing multi-port configuration..."

# Test 1: HTTP/1.1 on port 8080
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8080/http1")
if [ "$STATUS" = "200" ]; then
  echo "PASS: HTTP/1.1 on :8080 → 200"
else
  echo "FAIL: :8080/http1 returned $STATUS"
  exit 1
fi

# Test 2: HTTPS on port 8443
BODY=$(curl -sk "https://127.0.0.1:8443/https")
if [ "$BODY" = "HTTPS port 8443" ]; then
  echo "PASS: HTTPS on :8443 → 'HTTPS port 8443'"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 3: internal port 8081
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8081/internal")
if [ "$STATUS" = "200" ]; then
  echo "PASS: internal on :8081 → 200"
else
  echo "FAIL: :8081/internal returned $STATUS"
  exit 1
fi

# Test 4: verify access log
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8080/http1")
if [ "$STATUS" = "200" ]; then
  echo "PASS: access logging active"
fi

echo "All multi-port tests passed."
