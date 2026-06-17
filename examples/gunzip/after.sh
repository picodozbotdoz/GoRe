#!/bin/bash
kill $(cat /tmp/gore-example-gunzip.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-gunzip.pid
echo "Gunzip backend stopped."
