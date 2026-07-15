# Local Gateway Install Runbook

## Purpose

Install the SafeGAI edge gateway on a local Ubuntu machine for
lab testing or pilot deployment.

## Target Environment

- Ubuntu 22.04 LTS (amd64)
- Minimum: 2 CPU cores, 2 GB RAM, 10 GB disk
- Network: Access to cameras, PLCs, and output devices

## Installation Steps

### 1. Get the Package

From CI artifacts:
```bash
# Download from GitHub release or CI
wget https://github.com/thingswell-hjlee/safegai-iot-automation/releases/download/v0.1.0/safegai-edge_0.1.0_amd64.deb
```

Or build locally:
```bash
make package-amd64
```

### 2. Install

```bash
sudo dpkg -i safegai-edge_0.1.0_amd64.deb
```

### 3. Configure

```bash
sudo tee /etc/safegai/environment <<EOF
SAFEGAI_PROFILE=local-pilot
SAFEGAI_GATEWAY_ID=gw-factory-001
SAFEGAI_SITE_ID=site-factory-01
SAFEGAI_TENANT_ID=tenant-thingswell
SAFEGAI_LISTEN_ADDR=:8080
EOF
```

### 4. Start

```bash
sudo systemctl enable safegai-edge
sudo systemctl start safegai-edge
```

### 5. Verify

```bash
curl -s http://localhost:8080/health/ready | jq
```

Expected:
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "5s"
}
```

## Configuration Profiles

| Profile | Use Case |
|---------|----------|
| local-sim | Development with simulators |
| local-lab | Lab with partial hardware |
| local-pilot | Factory with all hardware |

## File Locations

| Path | Contents |
|------|----------|
| /usr/local/bin/safegai-edge | Gateway binary |
| /etc/safegai/ | Configuration files |
| /var/lib/safegai/ | SQLite database |
| /var/log/safegai/ | Log files |
| /etc/systemd/system/ | Service unit file |

## Health Check

```bash
# Service status
systemctl status safegai-edge

# Liveness
curl http://localhost:8080/health/live

# Readiness
curl http://localhost:8080/health/ready

# Logs
journalctl -u safegai-edge -f
```

## Troubleshooting

### Service fails to start
```bash
journalctl -u safegai-edge -n 50
# Common issues: SQLite path not writable, port already in use
```

### Permission denied
```bash
# Ensure data directory exists and is writable
sudo mkdir -p /var/lib/safegai
sudo chown safegai:safegai /var/lib/safegai
```

### Port conflict
```bash
sudo ss -tlnp | grep 8080
# Change SAFEGAI_LISTEN_ADDR if needed
```
