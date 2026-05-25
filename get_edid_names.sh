#!/bin/bash

for port in /sys/class/drm/*/status; do
    if grep -q "^connected" "$port"; then
        echo -e "\n--- $(basename $(dirname $port)) ---"
        edid-decode "$(dirname $port)/edid" | grep -i "Display Product Name"
    fi
done
