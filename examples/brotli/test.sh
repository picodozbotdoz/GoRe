#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing brotli compression..."

# Test 1: compressed response when client accepts br
HEADERS=$(curl -s -D - -H "Accept-Encoding: br" -o /dev/null "$BASE/text")
if echo "$HEADERS" | grep -qi "content-encoding: br"; then
  echo "PASS: Accept-Encoding: br → Content-Encoding: br"
else
  echo "FAIL: no brotli encoding in response"
  echo "$HEADERS"
  exit 1
fi

# Test 2: uncompressed when client doesn't accept br
HEADERS=$(curl -s -D - -o /dev/null "$BASE/text")
if echo "$HEADERS" | grep -qi "content-encoding: br"; then
  echo "FAIL: should not compress without Accept-Encoding"
  exit 1
else
  echo "PASS: no Accept-Encoding → no compression"
fi

echo "All brotli tests passed."
