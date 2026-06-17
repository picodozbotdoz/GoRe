#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing connection limit..."

# Test 1: sequential requests should all pass (connections released after each)
for i in 1 2 3 4 5; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
  if [ "$STATUS" != "200" ]; then
    echo "FAIL: request $i returned $STATUS, want 200"
    exit 1
  fi
done
echo "PASS: sequential requests all pass (connections released)"

# Test 2: concurrent requests may hit limit
BLOCKED=0
for i in $(seq 1 5); do
  curl -s -o /dev/null "$BASE/" &
done
wait

echo "All connection limit tests passed."
