#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
rm -f "$DIR/cert.pem" "$DIR/key.pem"
echo "Cleaned up certificates."
