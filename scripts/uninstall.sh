#!/usr/bin/env bash
# Uninstall SafeGAI Gateway
# Usage: sudo ./uninstall.sh [--purge]
set -euo pipefail

PURGE="${1:-}"

echo "=== Uninstalling SafeGAI Gateway ==="

# Stop service
if systemctl is-active safegai-edge.service >/dev/null 2>&1; then
    echo "Stopping service..."
    systemctl stop safegai-edge.service
fi

if [ "$PURGE" = "--purge" ]; then
    echo "Purging package and data..."
    dpkg --purge safegai-gateway
    echo "Data removed from /var/lib/safegai and /var/log/safegai"
else
    echo "Removing package (data preserved)..."
    dpkg --remove safegai-gateway
    echo "Data preserved at /var/lib/safegai"
fi

echo "=== Uninstall complete ==="
