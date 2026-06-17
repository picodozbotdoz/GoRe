#!/bin/bash
kill $(cat /tmp/gore-example-gw.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-gw.pid
echo "API gateway mock services stopped."
