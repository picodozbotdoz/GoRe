#!/bin/bash
# Mock auth server
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
            self.wfile.write(b'Unauthorized')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9000), H).serve_forever()
" &
echo $! > /tmp/gore-example-gw.pid
# Users backend
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(b'{\"users\":[{\"id\":1,\"name\":\"Alice\"}]}')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9001), H).serve_forever()
" &
echo $! >> /tmp/gore-example-gw.pid
# Orders backend
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(b'{\"orders\":[{\"id\":100,\"item\":\"Widget\"}]}')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9002), H).serve_forever()
" &
echo $! >> /tmp/gore-example-gw.pid
sleep 1
echo "API gateway mock services started."
