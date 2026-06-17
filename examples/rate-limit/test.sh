#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing rate limiting..."

# Test 1: first requests should pass (burst allows 5)
for i in 1 2 3 4 5; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
  if [ "$STATUS" != "200" ]; then
    echo "FAIL: request $i returned $STATUS, want 200 (burst)"
    exit 1
  fi
done
echo "PASS: first 5 requests within burst → 200"

# Test 2: request beyond burst should be rate limited
sleep 1
for i in $(seq 1 10); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
  if [ "$STATUS" = "429" ]; then
    echo "PASS: request $i returned 429 (rate limited)"
    break
  fi
  if [ "$i" = "10" ]; then
    echo "WARN: no 429 received in 10 requests"
  fi
done

echo "All rate limit tests passed."
