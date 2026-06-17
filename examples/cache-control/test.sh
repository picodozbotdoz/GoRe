#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing Cache-Control..."

# Test 1: Cache-Control header should be present
HEADERS=$(curl -s -D - -o /dev/null "$BASE/index.html")
if echo "$HEADERS" | grep -qi "cache-control: public, max-age=3600"; then
  echo "PASS: Cache-Control header present"
else
  echo "FAIL: Cache-Control header not found"
  echo "$HEADERS"
  exit 1
fi

echo "All Cache-Control tests passed."
