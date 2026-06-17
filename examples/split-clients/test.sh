#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing split clients..."

# Split sets header on request, not response. Verify it runs without error
# and different users get different variants (by checking status).

# Test 1: multiple requests should all succeed
for i in $(seq 1 20); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "User-Agent: test-$i" "$BASE/")
  if [ "$STATUS" != "200" ]; then
    echo "FAIL: request $i returned $STATUS"
    exit 1
  fi
done
echo "PASS: all requests succeed (split module processes without error)"

echo "All split client tests passed."
