#!/usr/bin/env bash
# Collect SafeGAI Gateway logs and diagnostics
# Usage: ./log-collect.sh [output-dir]
set -euo pipefail

OUTPUT_DIR="${1:-/tmp/safegai-logs}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
COLLECT_DIR="$OUTPUT_DIR/safegai-diag-${TIMESTAMP}"

echo "=== Collecting SafeGAI Gateway Diagnostics ==="
mkdir -p "$COLLECT_DIR"

# System info
echo "Collecting system info..."
uname -a > "$COLLECT_DIR/system-info.txt" 2>/dev/null || true
cat /etc/os-release >> "$COLLECT_DIR/system-info.txt" 2>/dev/null || true
free -h >> "$COLLECT_DIR/system-info.txt" 2>/dev/null || true
df -h >> "$COLLECT_DIR/system-info.txt" 2>/dev/null || true

# Service status
echo "Collecting service status..."
systemctl status safegai-edge.service > "$COLLECT_DIR/service-status.txt" 2>/dev/null || true

# Journal logs (last 1000 lines)
echo "Collecting journal logs..."
journalctl -u safegai-edge.service --no-pager -n 1000 > "$COLLECT_DIR/journal.log" 2>/dev/null || true

# Application logs
echo "Collecting application logs..."
if [ -d /var/log/safegai ]; then
    cp -r /var/log/safegai "$COLLECT_DIR/app-logs/" 2>/dev/null || true
fi

# Database info
echo "Collecting database info..."
DB_PATH="/var/lib/safegai/data/safegai.db"
if [ -f "$DB_PATH" ] && command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 "$DB_PATH" "SELECT 'schema_version', MAX(version) FROM schema_migrations;" > "$COLLECT_DIR/db-info.txt" 2>/dev/null || true
    sqlite3 "$DB_PATH" "SELECT 'events', COUNT(*) FROM events;" >> "$COLLECT_DIR/db-info.txt" 2>/dev/null || true
    sqlite3 "$DB_PATH" "SELECT 'outbox_pending', COUNT(*) FROM cloud_outbox WHERE status='PENDING';" >> "$COLLECT_DIR/db-info.txt" 2>/dev/null || true
    sqlite3 "$DB_PATH" "SELECT 'boot_records', COUNT(*) FROM boot_records;" >> "$COLLECT_DIR/db-info.txt" 2>/dev/null || true
fi

# Health endpoint
echo "Collecting health data..."
curl -s http://localhost:8080/health/live > "$COLLECT_DIR/health-live.json" 2>/dev/null || true
curl -s http://localhost:8080/health/ready > "$COLLECT_DIR/health-ready.json" 2>/dev/null || true

# Configuration (without secrets)
echo "Collecting configuration..."
if [ -f /etc/safegai/config.yaml ]; then
    cat /etc/safegai/config.yaml > "$COLLECT_DIR/config.yaml" 2>/dev/null || true
fi

# Create tarball
TARBALL="$OUTPUT_DIR/safegai-diag-${TIMESTAMP}.tar.gz"
tar -czf "$TARBALL" -C "$OUTPUT_DIR" "safegai-diag-${TIMESTAMP}"
rm -rf "$COLLECT_DIR"

echo "=== Diagnostics collected: $TARBALL ==="
echo "Size: $(du -h "$TARBALL" | cut -f1)"
