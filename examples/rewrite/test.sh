#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing URL rewrite..."

# Test 1: /old/foo should redirect to /new/foo
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/old/foo")
if [ "$STATUS" != "301" ]; then
  echo "FAIL: /old/foo returned $STATUS, want 301"
  exit 1
fi
echo "PASS: /old/foo → 301"

# Test 2: follow redirect should hit /new/foo
LOCATION=$(curl -s -o /dev/null -w "%{redirect_url}" "$BASE/old/foo")
if echo "$LOCATION" | grep -q "/new/foo"; then
  echo "PASS: redirect location = $LOCATION"
else
  echo "FAIL: location = '$LOCATION', want '/new/foo'"
  exit 1
fi

# Test 3: /new directly returns content
BODY=$(curl -s "$BASE/new")
if [ "$BODY" = "redirected content" ]; then
  echo "PASS: /new → 'redirected content'"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

echo "All rewrite tests passed."
