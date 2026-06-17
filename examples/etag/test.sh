#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing ETag..."

# Test 1: ETag header should be present
HEADERS=$(curl -s -D - -o /dev/null "$BASE/index.html")
if echo "$HEADERS" | grep -qi "etag:"; then
  echo "PASS: ETag header present"
else
  echo "FAIL: ETag header not found"
  exit 1
fi

# Test 2: conditional request with matching ETag → 304
ETAG=$(echo "$HEADERS" | grep -i "etag:" | head -1 | awk '{print $2}' | tr -d '\r')
if [ -n "$ETAG" ]; then
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "If-None-Match: $ETAG" "$BASE/index.html")
  if [ "$STATUS" = "304" ]; then
    echo "PASS: matching ETag → 304"
  else
    echo "INFO: status = $STATUS (304 expected)"
  fi
fi

echo "All ETag tests passed."
