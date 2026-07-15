#!/usr/bin/env bash
# Build Debian package for SafeGAI Gateway
# Produces: packages/safegai-gateway_<version>_amd64.deb
#
# Requirements: dpkg-deb, fakeroot (or run as root)
# The same package is installed on AWS EC2 and local Ubuntu IPC.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VERSION="${VERSION:-0.1.0}"
ARCH="amd64"
PKG_NAME="safegai-gateway"
PKG_DIR="${REPO_ROOT}/dist/deb-staging/${PKG_NAME}_${VERSION}_${ARCH}"
OUTPUT_DIR="${REPO_ROOT}/packages"

echo "=== Building SafeGAI Gateway Debian Package ==="
echo "Version: ${VERSION}"
echo "Architecture: ${ARCH}"

# Ensure binary exists
BINARY="${REPO_ROOT}/dist/safegai-edge"
if [ ! -f "$BINARY" ]; then
    echo "Binary not found at ${BINARY}. Building..."
    (cd "${REPO_ROOT}/services/gateway-server" && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o "${BINARY}" ./cmd/safegai-edge)
fi

# Create package directory structure
rm -rf "$PKG_DIR"
mkdir -p "$PKG_DIR/DEBIAN"
mkdir -p "$PKG_DIR/opt/safegai/current/bin"
mkdir -p "$PKG_DIR/etc/safegai"
mkdir -p "$PKG_DIR/var/lib/safegai/data"
mkdir -p "$PKG_DIR/var/log/safegai"
mkdir -p "$PKG_DIR/lib/systemd/system"
mkdir -p "$PKG_DIR/etc/logrotate.d"

# Copy binary
cp "$BINARY" "$PKG_DIR/opt/safegai/current/bin/safegai-edge"
chmod 755 "$PKG_DIR/opt/safegai/current/bin/safegai-edge"

# Copy systemd unit
cp "${REPO_ROOT}/infra/edge/systemd/safegai-edge.service" "$PKG_DIR/lib/systemd/system/"

# Copy configuration files
cp "${REPO_ROOT}/configs/common.yaml" "$PKG_DIR/etc/safegai/"
cp "${REPO_ROOT}/configs/local-sim.yaml" "$PKG_DIR/etc/safegai/"
# Default config symlink
ln -sf "/etc/safegai/local-sim.yaml" "$PKG_DIR/etc/safegai/config.yaml"

# Copy logrotate config
cp "${SCRIPT_DIR}/../logrotate/safegai-gateway" "$PKG_DIR/etc/logrotate.d/" 2>/dev/null || \
    echo '/var/log/safegai/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    create 0640 safegai safegai
    postrotate
        systemctl reload safegai-edge 2>/dev/null || true
    endscript
}' > "$PKG_DIR/etc/logrotate.d/safegai-gateway"

# Create DEBIAN/control
cat > "$PKG_DIR/DEBIAN/control" << EOF
Package: ${PKG_NAME}
Version: ${VERSION}
Section: misc
Priority: optional
Architecture: ${ARCH}
Depends: sqlite3, systemd
Maintainer: ThingsWell <dev@thingswell.com>
Description: SafeGAI Industrial Safety Gateway
 Edge gateway for industrial safety monitoring with zone-based
 occupancy detection, equipment state management, and safety
 rule enforcement. Supports SQLite WAL storage, local REST API,
 WebSocket, and optional AWS IoT cloud sync.
EOF

# Create DEBIAN/conffiles
cat > "$PKG_DIR/DEBIAN/conffiles" << EOF
/etc/safegai/common.yaml
/etc/safegai/local-sim.yaml
/etc/safegai/config.yaml
/etc/logrotate.d/safegai-gateway
EOF

# Create DEBIAN/preinst
cat > "$PKG_DIR/DEBIAN/preinst" << 'EOF'
#!/bin/bash
set -e
# Create safegai user/group if not exists
if ! getent group safegai >/dev/null 2>&1; then
    groupadd --system safegai
fi
if ! getent passwd safegai >/dev/null 2>&1; then
    useradd --system --gid safegai --home /var/lib/safegai --shell /usr/sbin/nologin safegai
fi
EOF
chmod 755 "$PKG_DIR/DEBIAN/preinst"

# Create DEBIAN/postinst
cat > "$PKG_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
set -e
# Set ownership
chown -R safegai:safegai /var/lib/safegai
chown -R safegai:safegai /var/log/safegai
chmod 750 /var/lib/safegai/data

# Reload systemd
systemctl daemon-reload
# Enable but do not start (user must configure and start manually)
systemctl enable safegai-edge.service || true
echo "SafeGAI Gateway installed. Configure /etc/safegai/config.yaml then run:"
echo "  sudo systemctl start safegai-edge"
EOF
chmod 755 "$PKG_DIR/DEBIAN/postinst"

# Create DEBIAN/prerm
cat > "$PKG_DIR/DEBIAN/prerm" << 'EOF'
#!/bin/bash
set -e
# Stop service before removal
if systemctl is-active safegai-edge.service >/dev/null 2>&1; then
    systemctl stop safegai-edge.service
fi
EOF
chmod 755 "$PKG_DIR/DEBIAN/prerm"

# Create DEBIAN/postrm
cat > "$PKG_DIR/DEBIAN/postrm" << 'EOF'
#!/bin/bash
set -e
if [ "$1" = "purge" ]; then
    rm -rf /var/lib/safegai
    rm -rf /var/log/safegai
    userdel safegai 2>/dev/null || true
    groupdel safegai 2>/dev/null || true
fi
systemctl daemon-reload || true
EOF
chmod 755 "$PKG_DIR/DEBIAN/postrm"

# Build the package
mkdir -p "$OUTPUT_DIR"
if command -v dpkg-deb >/dev/null 2>&1; then
    dpkg-deb --build "$PKG_DIR" "$OUTPUT_DIR/${PKG_NAME}_${VERSION}_${ARCH}.deb"
else
    # Fallback: build .deb manually using ar + tar (for environments without dpkg-deb)
    echo "dpkg-deb not found, building .deb manually with ar..."
    DEB_FILE="$OUTPUT_DIR/${PKG_NAME}_${VERSION}_${ARCH}.deb"

    # Create debian-binary
    echo "2.0" > /tmp/debian-binary

    # Create control.tar.gz
    (cd "$PKG_DIR/DEBIAN" && tar czf /tmp/control.tar.gz ./*)

    # Create data.tar.gz (everything except DEBIAN)
    (cd "$PKG_DIR" && tar czf /tmp/data.tar.gz --exclude='./DEBIAN' ./)

    # Assemble .deb using ar
    rm -f "$DEB_FILE"
    ar rcs "$DEB_FILE" /tmp/debian-binary /tmp/control.tar.gz /tmp/data.tar.gz

    rm -f /tmp/debian-binary /tmp/control.tar.gz /tmp/data.tar.gz
fi

echo "=== Package built: ${OUTPUT_DIR}/${PKG_NAME}_${VERSION}_${ARCH}.deb ==="

# Generate checksum
(cd "$OUTPUT_DIR" && sha256sum "${PKG_NAME}_${VERSION}_${ARCH}.deb" > "${PKG_NAME}_${VERSION}_${ARCH}.deb.sha256")
echo "=== SHA-256: $(cat "${OUTPUT_DIR}/${PKG_NAME}_${VERSION}_${ARCH}.deb.sha256") ==="

# Cleanup staging
rm -rf "${REPO_ROOT}/dist/deb-staging"
