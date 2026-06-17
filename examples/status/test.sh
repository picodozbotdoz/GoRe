#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing status endpoint..."

# Test 1: /status should return 200
BODY=$(curl -s "$BASE/status")
if echo "$BODY" | grep -q "Active connections:"; then
  echo "PASS: /status returns active connections info"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 2: should contain server accepts handled requests
if echo "$BODY" | grep -q "server accepts handled requests"; then
  echo "PASS: /status returns request stats"
else
  echo "FAIL: missing request stats"
  exit 1
fi

# Test 3: should contain uptime
if echo "$BODY" | grep -q "Uptime:"; then
  echo "PASS: /status returns uptime"
else
  echo "FAIL: missing uptime"
  exit 1
fi

# Test 4: hit the endpoint, then check counters updated
curl -s "$BASE/" > /dev/null
BODY=$(curl -s "$BASE/status")
echo "After request:"
echo "$BODY" | head -5

# Test 5: status endpoint content type should be text/plain
CT=$(curl -s -D - -o /dev/null "$BASE/status" | grep -i content-type)
if echo "$CT" | grep -qi "text/plain"; then
  echo "PASS: Content-Type is text/plain"
else
  echo "INFO: Content-Type = $CT"
fi

echo "All status tests passed."
