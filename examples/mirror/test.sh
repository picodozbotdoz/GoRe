#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing mirror..."

# Test 1: primary response should be returned
BODY=$(curl -s "$BASE/")
if [ "$BODY" = "primary response" ]; then
  echo "PASS: primary response returned"
else
  echo "FAIL: body = '$BODY', want 'primary response'"
  exit 1
fi

sleep 1
echo "All mirror tests passed."
