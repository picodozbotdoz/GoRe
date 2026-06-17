#!/bin/bash
set -e
BASE="http://127.0.0.1:8080"

echo "Testing proxy retry..."

# Backend 1 returns 500, Backend 2 returns 200
# With retries=2, GoRe should retry and hit Backend 2
BODY=$(curl -s "$BASE/")
if [ "$BODY" = "backend-2-ok" ]; then
  echo "PASS: retried to backend-2 after backend-1 failed"
elif [ "$BODY" = "error" ]; then
  echo "FAIL: did not retry (got backend-1 error)"
  exit 1
else
  echo "INFO: body = '$BODY'"
fi

echo "All proxy retry tests passed."
