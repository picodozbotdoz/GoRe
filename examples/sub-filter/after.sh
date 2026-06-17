#!/bin/bash
kill $(cat /tmp/gore-example-subfilter.pid 2>/dev/null) 2>/dev/null || true
rm -f /tmp/gore-example-subfilter.pid
echo "Sub-filter backend stopped."
