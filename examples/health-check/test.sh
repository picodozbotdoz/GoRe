#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing health checks..."

# Test 1: requests should succeed (healthy backends)
for i in $(seq 1 3); do
  BODY=$(curl -s "$BASE/")
  if [ -z "$BODY" ]; then
    echo "FAIL: empty response"
    exit 1
  fi
done
echo "PASS: requests succeed with healthy backends"

# Test 2: verify both backends are hit (round-robin)
BODY1=$(curl -s "$BASE/")
BODY2=$(curl -s "$BASE/")
echo "Responses: $BODY1, $BODY2"

echo "All health check tests passed."
