#!/bin/bash
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'backend-9001 response')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9001), H).serve_forever()
" &
echo $! > /tmp/gore-example-subfilter.pid
sleep 1
echo "Sub-filter backend started."
