#!/usr/bin/env bash
# Rollback SafeGAI Gateway to a previous backup
# Usage: sudo ./rollback.sh [backup-timestamp]
# Without timestamp, uses the most recent backup.
set -euo pipefail

TIMESTAMP="${1:-}"
BACKUP_DIR="/var/lib/safegai/backups"
DB_PATH="/var/lib/safegai/data/safegai.db"

echo "=== Rolling Back SafeGAI Gateway ==="

if [ ! -d "$BACKUP_DIR" ]; then
    echo "ERROR: No backup directory found at $BACKUP_DIR"
    exit 1
fi

# Find backup file
if [ -z "$TIMESTAMP" ]; then
    BACKUP_FILE=$(ls -t "$BACKUP_DIR"/safegai_*.db 2>/dev/null | head -1)
    if [ -z "$BACKUP_FILE" ]; then
        echo "ERROR: No backup files found"
        exit 1
    fi
else
    BACKUP_FILE="$BACKUP_DIR/safegai_${TIMESTAMP}.db"
    if [ ! -f "$BACKUP_FILE" ]; then
        echo "ERROR: Backup not found: $BACKUP_FILE"
        echo "Available backups:"
        ls "$BACKUP_DIR"/safegai_*.db 2>/dev/null || echo "  (none)"
        exit 1
    fi
fi

echo "Backup file: $BACKUP_FILE"

# Stop service
if systemctl is-active safegai-edge.service >/dev/null 2>&1; then
    echo "Stopping service..."
    systemctl stop safegai-edge.service
fi

# Restore database
echo "Restoring database..."
cp "$BACKUP_FILE" "$DB_PATH"
WAL_FILE="${BACKUP_FILE}-wal"
if [ -f "$WAL_FILE" ]; then
    cp "$WAL_FILE" "${DB_PATH}-wal"
else
    rm -f "${DB_PATH}-wal" "${DB_PATH}-shm"
fi
chown safegai:safegai "$DB_PATH" "${DB_PATH}-wal" 2>/dev/null || true

# Start service
echo "Starting service..."
systemctl start safegai-edge.service

sleep 3
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/ready 2>/dev/null || echo "000")

if [ "$HEALTH_STATUS" = "200" ]; then
    echo "=== Rollback successful ==="
else
    echo "WARNING: Service may not be healthy (HTTP $HEALTH_STATUS)"
    echo "Check: systemctl status safegai-edge"
fi
