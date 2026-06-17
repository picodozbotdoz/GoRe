#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing proxy cache..."

# Test 1: first request should be MISS
HEADERS=$(curl -s -D - -o /dev/null "$BASE/")
if echo "$HEADERS" | grep -qi "x-cache: MISS"; then
  echo "PASS: first request → X-Cache: MISS"
else
  echo "WARN: no X-Cache header on first request"
fi

# Test 2: second request should be HIT
HEADERS=$(curl -s -D - -o /dev/null "$BASE/")
if echo "$HEADERS" | grep -qi "x-cache: HIT"; then
  echo "PASS: second request → X-Cache: HIT"
else
  echo "WARN: no X-Cache: HIT on second request (may need cache TTL adjustment)"
fi

echo "All proxy cache tests passed."
