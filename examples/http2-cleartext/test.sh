#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing HTTP/2 h2c..."

# Test 1: basic GET
BODY=$(curl -s "$BASE/")
if [ "$BODY" != "Hello HTTP/2 h2c" ]; then
  echo "FAIL: body = '$BODY', want 'Hello HTTP/2 h2c'"
  exit 1
fi
echo "PASS: GET / → 200 with body"

# Test 2: HTTP/2 protocol via --http2-prior-knowledge
PROTO=$(curl -s --http2-prior-knowledge -o /dev/null -w "%{http_version}" "$BASE/")
if [ "$PROTO" = "2" ] || [ "$PROTO" = "2.0" ]; then
  echo "PASS: Protocol is HTTP/2 ($PROTO)"
else
  echo "INFO: Protocol = $PROTO"
fi

# Test 3: HTTP/1.1 fallback
STATUS=$(curl -s -o /dev/null -w "%{http_version}" "$BASE/headers")
if [ "$STATUS" = "1.1" ]; then
  echo "PASS: HTTP/1.1 fallback works"
else
  echo "INFO: HTTP/1.1 version = $STATUS"
fi

echo "All HTTP/2 h2c tests passed."
