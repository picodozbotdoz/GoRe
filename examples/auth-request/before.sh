#!/bin/bash
set -e
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        if 'Authorization' in self.headers:
            self.send_response(200)
            self.end_headers()
        else:
            self.send_response(401)
            self.end_headers()
            self.wfile.write(b'unauthorized')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9000), H).serve_forever()
" &
echo $! > /tmp/gore-example-auth.pid
sleep 1
echo "Mock auth server started on :9000"
