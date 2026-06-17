#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
openssl req -x509 -newkey rsa:2048 -keyout "$DIR/key.pem" -out "$DIR/cert.pem" \
  -days 1 -nodes -subj '/CN=localhost' \
  -addext "subjectAltName=IP:127.0.0.1" 2>/dev/null

mkdir -p "$DIR/site/static"
echo '<!DOCTYPE html><html><body><h1>Production</h1></body></html>' > "$DIR/site/index.html"
echo 'body { font-family: sans-serif; }' > "$DIR/site/static/style.css"

python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'backend-ok')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9001), H).serve_forever()
" &
echo $! > /tmp/gore-example-prod.pid
python3 -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'backend-ok')
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', 9002), H).serve_forever()
" &
echo $! >> /tmp/gore-example-prod.pid
sleep 1
echo "Production prerequisites ready."
