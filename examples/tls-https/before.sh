#!/bin/bash
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Generating self-signed certificate..."
openssl req -x509 -newkey rsa:2048 -keyout "$DIR/key.pem" -out "$DIR/cert.pem" \
  -days 1 -nodes -subj '/CN=localhost' \
  -addext "subjectAltName=IP:127.0.0.1" 2>/dev/null

echo "Certificate generated: $DIR/cert.pem, $DIR/key.pem"
