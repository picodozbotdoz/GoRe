#!/bin/bash
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Creating site directory and files..."
mkdir -p "$DIR/site/css"
mkdir -p "$DIR/site/js"

cat > "$DIR/site/index.html" << 'HTML'
<!DOCTYPE html>
<html><head><title>GoRe Static</title>
<link rel="stylesheet" href="/css/style.css">
</head><body>
<h1>Hello from GoRe</h1>
<p>This is a static site served by GoRe.</p>
<script src="/js/app.js"></script>
</body></html>
HTML

cat > "$DIR/site/css/style.css" << 'CSS'
body { font-family: sans-serif; margin: 2em; }
h1 { color: #333; }
CSS

cat > "$DIR/site/js/app.js" << 'JS'
console.log("GoRe static site loaded");
JS

cat > "$DIR/site/about.html" << 'HTML'
<!DOCTYPE html>
<html><head><title>About</title></head>
<body><h1>About GoRe</h1><p>A Go nginx reimplementation.</p></body></html>
HTML

echo "Site files created in $DIR/site/"
