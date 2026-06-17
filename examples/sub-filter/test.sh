#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing sub_filter..."

# Test: backend response should have text replaced
BODY=$(curl -s "$BASE/")
if echo "$BODY" | grep -q "replaced-text"; then
  echo "PASS: sub_filter replaced 'backend-9001' with 'replaced-text'"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

echo "All sub_filter tests passed."
