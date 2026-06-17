#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing access control..."

# Test 1: allowed request from localhost
BODY=$(curl -s "$BASE/")
if [ "$BODY" != "allowed" ]; then
  echo "FAIL: body = '$BODY', want 'allowed'"
  exit 1
fi
echo "PASS: GET / → 200 'allowed'"

# Test 2: admin path also allowed (from localhost)
BODY=$(curl -s "$BASE/admin")
if [ "$BODY" != "admin panel" ]; then
  echo "FAIL: body = '$BODY', want 'admin panel'"
  exit 1
fi
echo "PASS: GET /admin → 200 'admin panel'"

echo "All access control tests passed."
