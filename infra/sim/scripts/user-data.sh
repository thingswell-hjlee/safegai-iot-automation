#!/bin/bash
# SafeGAI EC2 User-Data Script
# This script runs at instance launch via cloud-init to:
# 1. Install the SafeGAI gateway .deb package
# 2. Start all simulator services
# 3. Configure logging and health checks
#
# Environment variables expected from instance tags or SSM:
#   SAFEGAI_PROFILE=aws-sim
#   SAFEGAI_GATEWAY_ID=gw-sim-001
#   SAFEGAI_SITE_ID=site-sim-001
#   SAFEGAI_TENANT_ID=tenant-sim

set -euo pipefail

exec > >(tee /var/log/safegai-user-data.log) 2>&1
echo "=== SafeGAI user-data start: $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="

# --- System prerequisites ---
apt-get update -y
apt-get install -y --no-install-recommends \
  curl \
  jq \
  sqlite3 \
  awscli

# --- Download and install SafeGAI gateway package ---
SAFEGAI_VERSION="${SAFEGAI_VERSION:-0.1.0}"
SAFEGAI_S3_BUCKET="${SAFEGAI_S3_BUCKET:-safegai-artifacts}"
SAFEGAI_REGION="${SAFEGAI_REGION:-ap-northeast-2}"

DEB_FILE="safegai-edge_${SAFEGAI_VERSION}_amd64.deb"
DEB_PATH="/tmp/${DEB_FILE}"

echo "Downloading ${DEB_FILE} from S3..."
aws s3 cp "s3://${SAFEGAI_S3_BUCKET}/packages/${DEB_FILE}" "${DEB_PATH}" \
  --region "${SAFEGAI_REGION}" || {
  echo "WARN: S3 download failed, trying local build artifact..."
  if [ -f "/opt/safegai/${DEB_FILE}" ]; then
    cp "/opt/safegai/${DEB_FILE}" "${DEB_PATH}"
  else
    echo "ERROR: Cannot find .deb package"
    exit 1
  fi
}

echo "Installing SafeGAI gateway..."
dpkg -i "${DEB_PATH}" || apt-get install -f -y

# --- Configure gateway ---
mkdir -p /etc/safegai
cat > /etc/safegai/environment <<EOF
SAFEGAI_PROFILE=${SAFEGAI_PROFILE:-aws-sim}
SAFEGAI_GATEWAY_ID=${SAFEGAI_GATEWAY_ID:-gw-sim-001}
SAFEGAI_SITE_ID=${SAFEGAI_SITE_ID:-site-sim-001}
SAFEGAI_TENANT_ID=${SAFEGAI_TENANT_ID:-tenant-sim}
SAFEGAI_LISTEN_ADDR=:8080
SAFEGAI_LOG_LEVEL=info
EOF

# --- Enable and start services ---
systemctl daemon-reload

# Start gateway first
systemctl enable safegai-edge.service
systemctl start safegai-edge.service

# Wait for gateway to be ready
echo "Waiting for gateway to become ready..."
for i in $(seq 1 30); do
  if curl -sf http://localhost:8080/health/ready >/dev/null 2>&1; then
    echo "Gateway ready after ${i}s"
    break
  fi
  sleep 1
done

# Start simulators
SIMULATOR_SERVICES=(
  "safegai-camera-sim"
  "safegai-sensor-sim"
  "safegai-equipment-sim"
  "safegai-output-sim"
  "safegai-modbus-sim"
  "safegai-scenario-runner"
)

for svc in "${SIMULATOR_SERVICES[@]}"; do
  if systemctl list-unit-files | grep -q "${svc}.service"; then
    systemctl enable "${svc}.service"
    systemctl start "${svc}.service"
    echo "Started ${svc}"
  else
    echo "WARN: ${svc}.service not found, skipping"
  fi
done

# --- Health check ---
echo "Running initial health check..."
sleep 2

for svc in safegai-edge "${SIMULATOR_SERVICES[@]}"; do
  status=$(systemctl is-active "${svc}.service" 2>/dev/null || echo "inactive")
  echo "  ${svc}: ${status}"
done

# --- Signal completion to CloudFormation/CDK ---
if [ -n "${SAFEGAI_CFN_SIGNAL_URL:-}" ]; then
  curl -X PUT -H 'Content-Type:' \
    --data-binary '{"Status":"SUCCESS","Reason":"SafeGAI initialized","UniqueId":"user-data","Data":"ok"}' \
    "${SAFEGAI_CFN_SIGNAL_URL}"
fi

echo "=== SafeGAI user-data complete: $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
