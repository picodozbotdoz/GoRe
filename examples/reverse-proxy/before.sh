#!/bin/bash
set -e

echo "Starting mock backends..."

# Backend 1 on port 9001
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'backend-1')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9001), H).serve_forever()
" &
PID1=$!
echo $PID1 > /tmp/gore-example-backend1.pid

# Backend 2 on port 9002
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'backend-2')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9002), H).serve_forever()
" &
PID2=$!
echo $PID2 > /tmp/gore-example-backend2.pid

sleep 1
echo "Mock backends started on :9001 and :9002"
