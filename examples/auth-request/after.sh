#!/bin/bash
kill $(cat /tmp/gore-example-auth.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-auth.pid
echo "Mock auth server stopped."
