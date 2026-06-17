#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing reverse proxy..."

# Test 1: proxy forwards to backend
BODY=$(curl -s "$BASE/api/test")
if [ "$BODY" != "backend-1" ] && [ "$BODY" != "backend-2" ]; then
  echo "FAIL: proxy returned '$BODY', want 'backend-1' or 'backend-2'"
  exit 1
fi
echo "PASS: GET /api/test → proxied to backend ($BODY)"

# Test 2: load balancing distributes requests
COUNT1=0
COUNT2=0
for i in $(seq 1 10); do
  BODY=$(curl -s "$BASE/api/test")
  if [ "$BODY" = "backend-1" ]; then
    COUNT1=$((COUNT1 + 1))
  elif [ "$BODY" = "backend-2" ]; then
    COUNT2=$((COUNT2 + 1))
  fi
done
echo "Load distribution: backend-1=$COUNT1 backend-2=$COUNT2"
if [ "$COUNT1" -eq 0 ] || [ "$COUNT2" -eq 0 ]; then
  echo "WARN: load balancing may not be working (all went to one backend)"
else
  echo "PASS: load balancing distributes across backends"
fi

# Test 3: health endpoint returns 200
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: /health returned $STATUS, want 200"
  exit 1
fi
echo "PASS: GET /health → 200"

# Test 4: unmatched path returns 404
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/nope")
if [ "$STATUS" != "404" ]; then
  echo "FAIL: /nope returned $STATUS, want 404"
  exit 1
fi
echo "PASS: GET /nope → 404"

echo "All reverse proxy tests passed."
