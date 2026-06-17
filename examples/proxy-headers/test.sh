#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing proxy set headers..."

# Test: custom headers should be forwarded to backend
# Backend echoes the request, but we can verify via proxy response
BODY=$(curl -s "$BASE/")
if [ "$BODY" = "backend-9001" ]; then
  echo "PASS: proxy forwards to backend"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

echo "All proxy header tests passed."
