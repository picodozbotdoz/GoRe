#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing proxy buffering..."

# Test: streaming mode should work (response forwarded)
BODY=$(curl -s "$BASE/")
if [ "$BODY" = "backend-9001" ]; then
  echo "PASS: streaming proxy forwards response"
else
  echo "FAIL: body = '$BODY', want 'backend-9001'"
  exit 1
fi

echo "All proxy buffering tests passed."
