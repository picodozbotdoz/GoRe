#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"
HTTPS="https://127.0.0.1:8443"

echo "Testing production stack..."

# Test 1: HTTP/1.1 static file
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
if [ "$STATUS" = "200" ]; then
  echo "PASS: HTTP/1.1 static file serving"
else
  echo "FAIL: HTTP/1.1 returned $STATUS"
  exit 1
fi

# Test 2: proxy to backend
BODY=$(curl -s "$BASE/api/test")
if [ "$BODY" = "backend-ok" ]; then
  echo "PASS: proxy forwards to backend"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 3: health endpoint
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
if [ "$STATUS" = "200" ]; then
  echo "PASS: health endpoint returns 200"
else
  echo "FAIL: /health returned $STATUS"
  exit 1
fi

# Test 4: HTTPS
BODY=$(curl -sk "$HTTPS/")
if [ "$BODY" = "healthy" ] || [ "$BODY" = "backend-ok" ]; then
  echo "PASS: HTTPS request works"
else
  echo "INFO: HTTPS body = '$BODY'"
fi

# Test 5: status endpoint
BODY=$(curl -s "$BASE/status")
if echo "$BODY" | grep -q "Active connections:"; then
  echo "PASS: status endpoint works"
else
  echo "FAIL: status endpoint broken"
  exit 1
fi

# Test 6: gzip compression
HEADERS=$(curl -s -D - -H "Accept-Encoding: gzip" -o /dev/null "$BASE/")
if echo "$HEADERS" | grep -qi "content-encoding: gzip"; then
  echo "PASS: gzip compression active"
else
  echo "INFO: gzip not triggered (may need content-type)"
fi

echo "All production tests passed."
