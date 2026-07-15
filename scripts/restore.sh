#!/usr/bin/env bash
# Restore SafeGAI Gateway database from backup
# Usage: sudo ./restore.sh <backup-file>
set -euo pipefail

BACKUP_FILE="${1:-}"
DB_PATH="/var/lib/safegai/data/safegai.db"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file.db>"
    echo "Available backups:"
    ls -lt /var/lib/safegai/backups/safegai_*.db 2>/dev/null || echo "  (none found)"
    exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
    echo "ERROR: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "=== Restoring SafeGAI Gateway Database ==="
echo "From: $BACKUP_FILE"
echo "To: $DB_PATH"

# Verify checksum if available
if [ -f "${BACKUP_FILE}.sha256" ]; then
    echo "Verifying checksum..."
    if sha256sum -c "${BACKUP_FILE}.sha256" >/dev/null 2>&1; then
        echo "Checksum OK"
    else
        echo "ERROR: Checksum verification failed!"
        exit 1
    fi
fi

# Stop service
if systemctl is-active safegai-edge.service >/dev/null 2>&1; then
    echo "Stopping service..."
    systemctl stop safegai-edge.service
fi

# Remove WAL/SHM files
rm -f "${DB_PATH}-wal" "${DB_PATH}-shm"

# Copy backup
cp "$BACKUP_FILE" "$DB_PATH"
chown safegai:safegai "$DB_PATH"
chmod 640 "$DB_PATH"

# Start service
echo "Starting service..."
systemctl start safegai-edge.service

sleep 3
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/ready 2>/dev/null || echo "000")

if [ "$HEALTH_STATUS" = "200" ]; then
    echo "=== Restore successful ==="
else
    echo "WARNING: Service may not be healthy (HTTP $HEALTH_STATUS)"
    echo "Check: systemctl status safegai-edge"
fi
