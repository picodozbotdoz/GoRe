#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing body size limit..."

# Start a mock backend for the proxy
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get('Content-Length', 0)))
        self.send_response(200)
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9001), H).serve_forever()
" &
BACKEND_PID=$!
sleep 1

# Test 1: small body → 200
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -d "small" "$BASE/upload")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: small body returned $STATUS, want 200"
  kill $BACKEND_PID 2>/dev/null; exit 1
fi
echo "PASS: small POST body → 200"

# Test 2: oversized body → 413
OVERSIZED=$(python3 -c "print('x' * 2048)")
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -d "$OVERSIZED" "$BASE/upload")
if [ "$STATUS" != "413" ]; then
  echo "FAIL: oversized body returned $STATUS, want 413"
  kill $BACKEND_PID 2>/dev/null; exit 1
fi
echo "PASS: oversized POST body → 413"

# Test 3: GET works normally
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/")
if [ "$STATUS" != "200" ]; then
  echo "FAIL: GET / returned $STATUS, want 200"
  kill $BACKEND_PID 2>/dev/null; exit 1
fi
echo "PASS: GET / → 200"

kill $BACKEND_PID 2>/dev/null
echo "All body limit tests passed."
