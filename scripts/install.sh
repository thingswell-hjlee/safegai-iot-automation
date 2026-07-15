#!/usr/bin/env bash
# Install SafeGAI Gateway from .deb package
# Usage: sudo ./install.sh <package.deb> [profile]
# Profiles: aws-sim, local-sim, local-lab, local-pilot
set -euo pipefail

DEB_FILE="${1:-}"
PROFILE="${2:-local-sim}"

if [ -z "$DEB_FILE" ]; then
    echo "Usage: $0 <safegai-gateway_VERSION_amd64.deb> [profile]"
    echo "Profiles: aws-sim, local-sim, local-lab, local-pilot"
    exit 1
fi

if [ ! -f "$DEB_FILE" ]; then
    echo "ERROR: Package file not found: $DEB_FILE"
    exit 1
fi

echo "=== Installing SafeGAI Gateway ==="
echo "Package: $DEB_FILE"
echo "Profile: $PROFILE"

# Install package
dpkg -i "$DEB_FILE" || apt-get install -f -y

# Set profile symlink
if [ -f "/etc/safegai/${PROFILE}.yaml" ]; then
    ln -sf "/etc/safegai/${PROFILE}.yaml" /etc/safegai/config.yaml
    echo "Profile set to: $PROFILE"
else
    echo "WARNING: Profile config /etc/safegai/${PROFILE}.yaml not found"
    echo "Using default local-sim profile"
fi

echo "=== Installation complete ==="
echo "Next steps:"
echo "  1. Edit /etc/safegai/config.yaml"
echo "  2. sudo systemctl start safegai-edge"
echo "  3. curl http://localhost:8080/health/live"
