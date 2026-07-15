#!/usr/bin/env bash
# Backup SafeGAI Gateway database
# Usage: sudo ./backup.sh [output-dir]
set -euo pipefail

OUTPUT_DIR="${1:-/var/lib/safegai/backups}"
DB_PATH="/var/lib/safegai/data/safegai.db"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "=== Backing Up SafeGAI Gateway ==="

if [ ! -f "$DB_PATH" ]; then
    echo "ERROR: Database not found at $DB_PATH"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"
BACKUP_FILE="$OUTPUT_DIR/safegai_${TIMESTAMP}.db"

# Use sqlite3 backup command for safe WAL backup
if command -v sqlite3 >/dev/null 2>&1; then
    echo "Using sqlite3 online backup..."
    sqlite3 "$DB_PATH" ".backup '$BACKUP_FILE'"
else
    echo "sqlite3 not found, copying files directly..."
    cp "$DB_PATH" "$BACKUP_FILE"
    if [ -f "${DB_PATH}-wal" ]; then
        cp "${DB_PATH}-wal" "${BACKUP_FILE}-wal"
    fi
fi

# Generate checksum
sha256sum "$BACKUP_FILE" > "${BACKUP_FILE}.sha256"

echo "Backup saved: $BACKUP_FILE"
echo "Checksum: $(cat "${BACKUP_FILE}.sha256")"

# Cleanup old backups (keep last 7)
BACKUP_COUNT=$(ls -t "$OUTPUT_DIR"/safegai_*.db 2>/dev/null | wc -l)
if [ "$BACKUP_COUNT" -gt 7 ]; then
    echo "Cleaning up old backups (keeping 7)..."
    ls -t "$OUTPUT_DIR"/safegai_*.db | tail -n +8 | while read -r f; do
        rm -f "$f" "${f}-wal" "${f}.sha256"
    done
fi

echo "=== Backup complete ==="
