#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing auth request..."

# Test 1: public path → always 200
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/public")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: /public returned $STATUS, want 200"
  exit 1
fi
echo "PASS: GET /public → 200 (no auth needed)"

# Test 2: protected path without auth → 401
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/protected")
if [ "$STATUS" != "401" ]; then
  echo "FAIL: /protected without auth returned $STATUS, want 401"
  exit 1
fi
echo "PASS: GET /protected → 401 (no auth)"

# Test 3: protected path with auth → 200
BODY=$(curl -s -H "Authorization: Bearer token" "$BASE/protected")
if [ "$BODY" != "protected content" ]; then
  echo "FAIL: body = '$BODY', want 'protected content'"
  exit 1
fi
echo "PASS: GET /protected with auth → 200"

echo "All auth request tests passed."
