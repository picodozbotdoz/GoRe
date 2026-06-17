#!/bin/bash
python3 -c "
import gzip
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        body = gzip.compress(b'plain text from backend')
        self.send_response(200)
        self.send_header('Content-Encoding', 'gzip')
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9001), H).serve_forever()
" &
echo $! > /tmp/gore-example-gunzip.pid
sleep 1
echo "Gunzip backend started."
