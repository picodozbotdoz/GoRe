#!/bin/bash
kill $(cat /tmp/gore-example-backend1.pid 2>/dev/null) 2>/dev/null || true
kill $(cat /tmp/gore-example-backend2.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-backend*.pid
echo "Mock backends stopped."
