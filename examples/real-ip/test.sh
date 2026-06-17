#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing real IP..."

# Test: X-Forwarded-For should be used for client IP
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "X-Forwarded-For: 203.0.113.50" "$BASE/")
if [ "$STATUS" = "200" ]; then
  echo "PASS: request with X-Forwarded-For accepted"
else
  echo "FAIL: status = $STATUS"
  exit 1
fi

echo "All real IP tests passed."
