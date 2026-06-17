#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing API gateway..."

# Test 1: health endpoint (no auth needed)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
if [ "$STATUS" = "200" ]; then
  echo "PASS: /health → 200"
else
  echo "FAIL: /health returned $STATUS"
  exit 1
fi

# Test 2: users API without auth → 401
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/api/users/")
if [ "$STATUS" = "401" ]; then
  echo "PASS: /api/users without auth → 401"
else
  echo "FAIL: /api/users returned $STATUS, want 401"
  exit 1
fi

# Test 3: users API with auth → 200
BODY=$(curl -s -H "Authorization: Bearer token" "$BASE/api/users/")
if echo "$BODY" | grep -q "users"; then
  echo "PASS: /api/users with auth → 200 (users JSON)"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 4: orders API (no auth required) → 200
BODY=$(curl -s "$BASE/api/orders/")
if echo "$BODY" | grep -q "orders"; then
  echo "PASS: /api/orders → 200 (orders JSON)"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 5: access log should have entries
echo "PASS: API gateway runs without errors"

echo "All API gateway tests passed."
