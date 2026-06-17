#!/bin/bash
set -e

echo "Testing TLS cipher configuration..."

# Test 1: HTTPS request works
BODY=$(curl -sk "https://127.0.0.1:8443/")
if [ "$BODY" = "Hello over TLS" ]; then
  echo "PASS: HTTPS request with configured TLS"
else
  echo "FAIL: body = '$BODY'"
  exit 1
fi

# Test 2: verify TLS version from verbose output
TLS_INFO=$(curl -sk -v "https://127.0.0.1:8443/" 2>&1 | grep "SSL connection using")
if echo "$TLS_INFO" | grep -q "TLSv1.3"; then
  echo "PASS: TLS 1.3 negotiated ($TLS_INFO)"
elif echo "$TLS_INFO" | grep -q "TLSv1.2"; then
  echo "PASS: TLS 1.2 negotiated ($TLS_INFO)"
else
  echo "FAIL: TLS info = '$TLS_INFO'"
  exit 1
fi

# Test 3: verify certificate
CERT_INFO=$(curl -sk -v "https://127.0.0.1:8443/" 2>&1 | grep "subject:")
if echo "$CERT_INFO" | grep -q "localhost"; then
  echo "PASS: certificate subject is localhost"
else
  echo "FAIL: cert info = '$CERT_INFO'"
  exit 1
fi

echo "All TLS cipher tests passed."
