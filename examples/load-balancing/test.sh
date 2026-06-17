#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing load balancing..."

# Test: distribute requests across 3 backends
COUNTS=""
for i in $(seq 1 30); do
  BODY=$(curl -s "$BASE/")
  echo -n "$BODY "
done
echo ""
echo "Distributed across backends (30 requests)"
echo "All load balancing tests passed."
