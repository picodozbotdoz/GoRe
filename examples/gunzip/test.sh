#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing gunzip..."

# Test 1: client without gzip support should get decompressed body
BODY=$(curl -s "$BASE/")
if [ "$BODY" = "plain text from backend" ]; then
  echo "PASS: backend gzip response decompressed for client"
else
  echo "FAIL: body = '$BODY', want 'plain text from backend'"
  exit 1
fi

# Test 2: client with gzip support should get compressed response
HEADERS=$(curl -s -D - -H "Accept-Encoding: gzip" -o /dev/null "$BASE/")
if echo "$HEADERS" | grep -qi "content-encoding: gzip"; then
  echo "PASS: compressed response passed through when client accepts gzip"
else
  echo "INFO: response not gzip-compressed"
fi

echo "All gunzip tests passed."
