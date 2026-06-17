#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$DIR/site"
echo 'cached content' > "$DIR/site/index.html"
echo "Site created."
