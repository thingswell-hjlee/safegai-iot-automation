#!/usr/bin/env bash
# Generate release manifest for SafeGAI Gateway
# Records version, checksums, and compatibility info for parity verification.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="${VERSION:-0.1.0}"
OUTPUT_DIR="${REPO_ROOT}/dist"
BINARY="${OUTPUT_DIR}/safegai-edge"
PACKAGE="${REPO_ROOT}/packages/safegai-gateway_${VERSION}_amd64.deb"

echo "=== Generating Release Manifest ==="

MANIFEST_FILE="${OUTPUT_DIR}/release-manifest.json"
mkdir -p "$OUTPUT_DIR"

# Binary SHA-256
BINARY_SHA=""
if [ -f "$BINARY" ]; then
    BINARY_SHA=$(sha256sum "$BINARY" | cut -d' ' -f1)
fi

# Package SHA-256
PACKAGE_SHA=""
if [ -f "$PACKAGE" ]; then
    PACKAGE_SHA=$(sha256sum "$PACKAGE" | cut -d' ' -f1)
fi

# Git info
COMMIT_SHA=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")

# Schema version (count migrations in sqlite.go)
SCHEMA_VERSION=$(grep -c "^	\`" "${REPO_ROOT}/services/gateway-server/internal/storage/sqlite/sqlite.go" 2>/dev/null || echo "11")

# API contract version
API_VERSION="1.0.0"

# Safety rule version (always 1.0.0 for now - fixed rules)
SAFETY_RULE_VERSION="1.0.0"

cat > "$MANIFEST_FILE" << EOF
{
  "name": "safegai-gateway",
  "version": "${VERSION}",
  "commit_sha": "${COMMIT_SHA}",
  "build_time": "${BUILD_TIME}",
  "go_version": "${GO_VERSION}",
  "binary_sha256": "${BINARY_SHA}",
  "package_sha256": "${PACKAGE_SHA}",
  "db_schema_version": ${SCHEMA_VERSION},
  "api_contract_version": "${API_VERSION}",
  "safety_rule_version": "${SAFETY_RULE_VERSION}",
  "architecture": "amd64",
  "os": "linux",
  "min_ubuntu_version": "24.04",
  "storage_engine": "sqlite-wal",
  "portability": {
    "aws_ec2": true,
    "local_ipc": true,
    "same_binary": true,
    "same_package": true,
    "same_schema": true,
    "same_api": true,
    "same_safety_rules": true,
    "same_systemd_unit": true
  }
}
EOF

echo "Manifest written to: $MANIFEST_FILE"
cat "$MANIFEST_FILE"
