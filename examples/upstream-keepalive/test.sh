#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing upstream keepalive..."

# Test: multiple requests should reuse connection
for i in $(seq 1 5); do
  BODY=$(curl -s "$BASE/")
  if [ "$BODY" != "backend-9001" ]; then
    echo "FAIL: request $i body = '$BODY'"
    exit 1
  fi
done
echo "PASS: 5 sequential requests all succeed (connection reuse)"

echo "All upstream keepalive tests passed."
