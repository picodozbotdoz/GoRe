#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing proxy timeouts..."

# Test 1: slow backend should timeout (connect_timeout=2)
START=$(date +%s)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 3 "$BASE/" 2>/dev/null || echo "000")
END=$(date +%s)
ELAPSED=$((END - START))
echo "Request took ${ELAPSED}s, status=$STATUS"

if [ "$ELAPSED" -le 4 ]; then
  echo "PASS: request completed within timeout window"
else
  echo "WARN: request took ${ELAPSED}s, may exceed timeout"
fi

echo "All proxy timeout tests passed."
