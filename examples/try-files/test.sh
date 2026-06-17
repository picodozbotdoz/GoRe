#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing try_files..."

# Test 1: /index.html should serve the file
BODY=$(curl -s "$BASE/index.html")
if echo "$BODY" | grep -q "SPA App"; then
  echo "PASS: /index.html serves file"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 2: /unknown should fallback to /index.html (SPA fallback)
BODY=$(curl -s "$BASE/unknown")
if echo "$BODY" | grep -q "SPA App"; then
  echo "PASS: /unknown falls back to /index.html"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

echo "All try_files tests passed."
