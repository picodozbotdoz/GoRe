#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$DIR/site"
echo '<!DOCTYPE html><html><body><h1>SPA App</h1></body></html>' > "$DIR/site/index.html"
echo "Site created."
