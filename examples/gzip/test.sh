#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing gzip compression..."

# Test 1: compressed response when client accepts gzip
HEADERS=$(curl -s -D - -H "Accept-Encoding: gzip" -o /dev/null "$BASE/text")
if echo "$HEADERS" | grep -qi "content-encoding: gzip"; then
  echo "PASS: Accept-Encoding: gzip → Content-Encoding: gzip"
else
  echo "FAIL: no gzip encoding in response"
  echo "$HEADERS"
  exit 1
fi

# Test 2: uncompressed when client doesn't accept gzip
HEADERS=$(curl -s -D - -o /dev/null "$BASE/text")
if echo "$HEADERS" | grep -qi "content-encoding: gzip"; then
  echo "FAIL: should not compress without Accept-Encoding"
  exit 1
else
  echo "PASS: no Accept-Encoding → no compression"
fi

# Test 3: JSON endpoint with decompression
BODY=$(curl -s --compressed "$BASE/json")
if echo "$BODY" | grep -q "status"; then
  echo "PASS: JSON endpoint works with compression"
else
  echo "FAIL: JSON body = '$BODY'"
  exit 1
fi

echo "All gzip tests passed."
