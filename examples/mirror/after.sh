#!/bin/bash
kill $(cat /tmp/gore-example-mirror.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-mirror.pid
echo "Mirror backend stopped."
