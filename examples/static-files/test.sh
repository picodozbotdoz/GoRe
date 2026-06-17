#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing static file serving..."

# Test 1: index.html
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: index.html returned $STATUS, want 200"
  exit 1
fi
echo "PASS: GET / → 200"

# Test 2: specific file
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/about.html")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: about.html returned $STATUS, want 200"
  exit 1
fi
echo "PASS: GET /about.html → 200"

# Test 3: CSS file
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/css/style.css")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: css/style.css returned $STATUS, want 200"
  exit 1
fi
echo "PASS: GET /css/style.css → 200"

# Test 4: JS file
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/js/app.js")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: js/app.js returned $STATUS, want 200"
  exit 1
fi
echo "PASS: GET /js/app.js → 200"

# Test 5: 404 for missing file
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/nonexistent.html")
if [ "$STATUS" != "404" ]; then
  echo "FAIL: nonexistent.html returned $STATUS, want 404"
  exit 1
fi
echo "PASS: GET /nonexistent.html → 404"

# Test 6: directory traversal blocked
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/../../../etc/passwd")
if [ "$STATUS" != "403" ] && [ "$STATUS" != "404" ]; then
  echo "FAIL: path traversal returned $STATUS, want 403 or 404"
  exit 1
fi
echo "PASS: path traversal blocked → $STATUS"

# Test 7: ETag present
ETAG=$(curl -s -o /dev/null -w "%{header_json}" "$BASE/about.html" | grep -i etag)
if [ -z "$ETAG" ]; then
  echo "WARN: no ETag header (may need static module update)"
else
  echo "PASS: ETag header present"
fi

echo "All static file tests passed."
