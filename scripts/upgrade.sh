#!/usr/bin/env bash
# Upgrade SafeGAI Gateway to a new version
# Usage: sudo ./upgrade.sh <new-package.deb>
#
# Process:
#   1. Backup current database
#   2. Stop service
#   3. Install new package
#   4. Run migrations
#   5. Start service
#   6. Health check
#   7. On failure: rollback
set -euo pipefail

DEB_FILE="${1:-}"

if [ -z "$DEB_FILE" ]; then
    echo "Usage: $0 <safegai-gateway_VERSION_amd64.deb>"
    exit 1
fi

if [ ! -f "$DEB_FILE" ]; then
    echo "ERROR: Package file not found: $DEB_FILE"
    exit 1
fi

BACKUP_DIR="/var/lib/safegai/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_PATH="/var/lib/safegai/data/safegai.db"

echo "=== Upgrading SafeGAI Gateway ==="
echo "Package: $DEB_FILE"

# Step 1: Backup database
if [ -f "$DB_PATH" ]; then
    mkdir -p "$BACKUP_DIR"
    echo "Backing up database..."
    cp "$DB_PATH" "$BACKUP_DIR/safegai_${TIMESTAMP}.db"
    if [ -f "${DB_PATH}-wal" ]; then
        cp "${DB_PATH}-wal" "$BACKUP_DIR/safegai_${TIMESTAMP}.db-wal"
    fi
    echo "Backup saved to: $BACKUP_DIR/safegai_${TIMESTAMP}.db"
fi

# Step 2: Record current version for rollback
CURRENT_VERSION=$(dpkg-query -W -f='${Version}' safegai-gateway 2>/dev/null || echo "none")
echo "Current version: $CURRENT_VERSION"

# Step 3: Stop service
if systemctl is-active safegai-edge.service >/dev/null 2>&1; then
    echo "Stopping service..."
    systemctl stop safegai-edge.service
fi

# Step 4: Install new package
echo "Installing new package..."
dpkg -i "$DEB_FILE" || apt-get install -f -y

# Step 5: Start service
echo "Starting service..."
systemctl start safegai-edge.service

# Step 6: Health check
echo "Waiting for health check..."
sleep 3
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/ready 2>/dev/null || echo "000")

if [ "$HEALTH_STATUS" = "200" ]; then
    NEW_VERSION=$(dpkg-query -W -f='${Version}' safegai-gateway 2>/dev/null || echo "unknown")
    echo "=== Upgrade successful: $CURRENT_VERSION -> $NEW_VERSION ==="
else
    echo "ERROR: Health check failed (HTTP $HEALTH_STATUS)"
    echo "Attempting rollback..."
    # Restore database backup
    if [ -f "$BACKUP_DIR/safegai_${TIMESTAMP}.db" ]; then
        cp "$BACKUP_DIR/safegai_${TIMESTAMP}.db" "$DB_PATH"
    fi
    echo "Database restored. Manual intervention may be required."
    echo "To rollback the binary, install the previous .deb package."
    exit 1
fi
