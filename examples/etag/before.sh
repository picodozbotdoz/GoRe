#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$DIR/site"
echo 'etag test content' > "$DIR/site/index.html"
echo "Site created."
