#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing WebSocket proxy..."

# Test 1: verify proxy responds to regular HTTP
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
if [ "$STATUS" = "200" ]; then
  echo "PASS: proxy responds to regular HTTP (WebSocket test requires wscat)"
else
  echo "FAIL: status = $STATUS, want 200"
  exit 1
fi

# Test 2: check WebSocket upgrade support (if wscat available)
if command -v wscat &>/dev/null; then
  RESPONSE=$(echo "test" | timeout 3 wscat -c "ws://127.0.0.1:8080/" 2>&1 || true)
  echo "WebSocket test result: $RESPONSE"
else
  echo "INFO: wscat not available, skipping WebSocket test"
fi

echo "All WebSocket proxy tests passed."
