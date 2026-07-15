# AWS to Local Migration Runbook

## Purpose

Step-by-step guide to migrate from the AWS simulation environment to a local
Ubuntu gateway installation. The same binary runs in both environments;
migration is a configuration change.

## Prerequisites

- Target machine: Ubuntu 22.04 LTS (amd64)
- Network access to target (SSH or physical)
- Hardware connections available (cameras, Modbus devices, outputs)
- SafeGAI .deb package (from S3 or CI artifact)

## Migration Steps

### Phase 1: Preparation

1. **Stop AWS simulation** (to avoid duplicate processing):
   ```bash
   gh workflow run aws-sim-stop.yml
   ```

2. **Download .deb package** to the local machine:
   ```bash
   scp dist/safegai-edge_0.1.0_amd64.deb target:/tmp/
   ```

3. **Verify target prerequisites**:
   ```bash
   ssh target "uname -m && lsb_release -a && dpkg --print-architecture"
   ```

### Phase 2: Installation

4. **Install the package**:
   ```bash
   ssh target "sudo dpkg -i /tmp/safegai-edge_0.1.0_amd64.deb"
   ```

5. **Configure the environment**:
   ```bash
   ssh target "sudo tee /etc/safegai/environment" <<EOF
   SAFEGAI_PROFILE=local-pilot
   SAFEGAI_GATEWAY_ID=gw-local-001
   SAFEGAI_SITE_ID=site-factory-01
   SAFEGAI_TENANT_ID=tenant-thingswell
   SAFEGAI_LISTEN_ADDR=:8080
   EOF
   ```

6. **Deploy profile configuration** (if custom):
   ```bash
   scp configs/local-pilot.yaml target:/etc/safegai/configs/
   ```

### Phase 3: Hardware Connection

7. **Configure camera adapter**:
   - Update camera URLs in the profile YAML
   - Verify network connectivity to cameras
   - Test: `curl http://<camera-ip>/api/status`

8. **Configure Modbus adapter**:
   - Set Modbus TCP endpoint in profile YAML
   - Verify PLC connectivity
   - Test: `modbus_client --tcp <plc-ip>:502 --read 0 8`

9. **Configure output adapter**:
   - Set output device addresses in profile YAML
   - Verify wiring (warning lights, sirens, relays)

### Phase 4: Startup

10. **Enable and start the service**:
    ```bash
    ssh target "sudo systemctl enable safegai-edge && sudo systemctl start safegai-edge"
    ```

11. **Verify health**:
    ```bash
    ssh target "curl -s http://localhost:8080/health/ready | jq"
    ```

12. **Check logs**:
    ```bash
    ssh target "sudo journalctl -u safegai-edge -f"
    ```

### Phase 5: Validation

13. **Run portability test** on the local machine:
    ```bash
    scp tests/portability/run-portability-test.sh target:/tmp/
    ssh target "SAFEGAI_PROFILE=local-pilot /tmp/run-portability-test.sh"
    ```

14. **Verify safety behavior**:
    - Trigger a person detection event via camera
    - Verify warning light activates
    - Verify audit log entry created
    - Verify event stored in SQLite

15. **Cloud sync** (optional):
    - If cloud connectivity desired, ensure IoT Core certificates are provisioned
    - Set cloud adapter to `aws-iot` in profile
    - Verify outbox sync resumes

## Rollback

If issues occur:

```bash
# Stop local service
ssh target "sudo systemctl stop safegai-edge"

# Restart AWS simulation
gh workflow run aws-sim-start.yml
```

## Validation Checklist

- [ ] Gateway binary starts successfully
- [ ] Health endpoints respond correctly
- [ ] Camera events received and processed
- [ ] Safety rules evaluate correctly
- [ ] Output commands reach hardware
- [ ] Audit log entries created
- [ ] Graceful shutdown preserves state
- [ ] Cloud sync works (if configured)

## Troubleshooting

### Gateway fails to start
- Check `journalctl -u safegai-edge` for error messages
- Verify SQLite DB path is writable
- Verify config profile exists

### No camera events received
- Verify camera network connectivity
- Check camera adapter configuration in profile
- Review camera API compatibility

### Modbus communication fails
- Verify PLC IP and port
- Check firewall rules
- Test with standalone Modbus client tool

### Output commands not reaching devices
- Verify output device wiring
- Check Modbus coil addresses in configuration
- Test with manual Modbus write
