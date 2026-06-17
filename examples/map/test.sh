#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing map directive..."

# Map sets header on request, not response. Verify map runs without errors.
# Full verification requires a backend that echoes request headers.

# Test 1: mobile user agent should get 200 (map module processes without error)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "User-Agent: Mozilla/5.0 Mobile Safari" "$BASE/")
if [ "$STATUS" = "200" ]; then
  echo "PASS: map processes mobile UA without error"
else
  echo "FAIL: status = $STATUS"
  exit 1
fi

# Test 2: bot user agent
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "User-Agent: Googlebot/2.1" "$BASE/")
if [ "$STATUS" = "200" ]; then
  echo "PASS: map processes bot UA without error"
else
  echo "FAIL: status = $STATUS"
  exit 1
fi

# Test 3: desktop (default)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "User-Agent: Chrome/120" "$BASE/")
if [ "$STATUS" = "200" ]; then
  echo "PASS: map processes desktop UA without error"
else
  echo "FAIL: status = $STATUS"
  exit 1
fi

echo "All map tests passed."
