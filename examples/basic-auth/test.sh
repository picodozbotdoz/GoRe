#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing basic auth..."

# Test 1: no credentials → 401
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
if [ "$STATUS" != "401" ]; then
  echo "FAIL: no auth returned $STATUS, want 401"
  exit 1
fi
echo "PASS: no credentials → 401"

# Test 2: wrong credentials → 401
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -u "admin:wrong" "$BASE/")
if [ "$STATUS" != "401" ]; then
  echo "FAIL: wrong password returned $STATUS, want 401"
  exit 1
fi
echo "PASS: wrong password → 401"

# Test 3: correct credentials → 200
BODY=$(curl -s -u "admin:secret123" "$BASE/")
if [ "$BODY" != "Welcome, authenticated user" ]; then
  echo "FAIL: body = '$BODY', want 'Welcome, authenticated user'"
  exit 1
fi
echo "PASS: correct credentials → 200"

echo "All basic auth tests passed."
