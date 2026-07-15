#!/usr/bin/env bash
# Health check for SafeGAI Gateway
# Usage: ./health-check.sh [host:port]
# Exit code 0 = healthy, 1 = unhealthy
set -euo pipefail

ADDR="${1:-localhost:8080}"

echo "=== SafeGAI Gateway Health Check ==="
echo "Target: $ADDR"

# Check liveness
LIVE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${ADDR}/health/live" 2>/dev/null || echo "000")
echo "Liveness:  HTTP $LIVE_STATUS"

# Check readiness
READY_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${ADDR}/health/ready" 2>/dev/null || echo "000")
echo "Readiness: HTTP $READY_STATUS"

# Get version info
if [ "$LIVE_STATUS" = "200" ]; then
    VERSION_INFO=$(curl -s "http://${ADDR}/health/live" 2>/dev/null || echo "{}")
    echo "Response:  $VERSION_INFO"
fi

# Check systemd service status
if command -v systemctl >/dev/null 2>&1; then
    SERVICE_STATUS=$(systemctl is-active safegai-edge.service 2>/dev/null || echo "unknown")
    echo "Service:   $SERVICE_STATUS"
fi

# Check database
DB_PATH="/var/lib/safegai/data/safegai.db"
if [ -f "$DB_PATH" ]; then
    DB_SIZE=$(du -h "$DB_PATH" 2>/dev/null | cut -f1)
    echo "Database:  $DB_PATH ($DB_SIZE)"
else
    echo "Database:  not found"
fi

# Final verdict
if [ "$LIVE_STATUS" = "200" ] && [ "$READY_STATUS" = "200" ]; then
    echo "=== HEALTHY ==="
    exit 0
else
    echo "=== UNHEALTHY ==="
    exit 1
fi
