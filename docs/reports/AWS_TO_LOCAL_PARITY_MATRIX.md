# AWS to Local Parity Matrix

## Overview

This matrix documents the feature-by-feature equivalence between the AWS simulation
environment and local Ubuntu deployment. The SafeGAI gateway is designed as a
single binary that operates identically in both environments with only configuration
profile changes.

## Parity Dimensions

| Dimension | AWS Sim | Local Ubuntu | Parity Status |
|-----------|---------|--------------|---------------|
| Binary | Same Go binary | Same Go binary | IDENTICAL |
| Configuration | `aws-sim` profile | `local-sim/lab/pilot` profile | CONFIG ONLY |
| Safety Rules | Fixed rules R-01 to R-05 | Fixed rules R-01 to R-05 | IDENTICAL |
| Event Processing | In-process pipeline | In-process pipeline | IDENTICAL |
| Output Commands | HTTP to sim adapter | Modbus/DIO to hardware | ADAPTER ONLY |
| Camera Input | HTTP from sim | HTTP/RTSP from camera | ADAPTER ONLY |
| Sensor Input | HTTP from sim | Modbus from sensor | ADAPTER ONLY |
| Storage | SQLite WAL | SQLite WAL | IDENTICAL |
| Health API | :8080/health/* | :8080/health/* | IDENTICAL |
| Local REST API | :8080/api/* | :8080/api/* | IDENTICAL |
| Audit Logging | JSON to file + outbox | JSON to file + outbox | IDENTICAL |
| Cloud Sync | IoT Core MQTT | IoT Core MQTT (if connected) | IDENTICAL |
| Offline Operation | Full local operation | Full local operation | IDENTICAL |
| Graceful Shutdown | Signal handling | Signal handling | IDENTICAL |
| systemd Integration | ExecStart/Type=simple | ExecStart/Type=simple | IDENTICAL |

## Architecture Invariants

1. **No AWS SDK in domain/ports layer** - Verified by `go vet` and grep
2. **No conditional compilation for environment** - No `//go:build aws` tags in core
3. **Adapter selection via config only** - Profile YAML determines which adapters load
4. **Safety rules are environment-independent** - Same decisions regardless of deployment
5. **Storage schema is identical** - Same SQLite migrations everywhere

## Verification Commands

```bash
# Run portability test
./tests/portability/run-portability-test.sh

# Verify parity
./tests/portability/verify-parity.sh http://localhost:8080

# Check for AWS SDK in core (should return nothing)
grep -r "github.com/aws" services/gateway-server/internal/domain/
grep -r "github.com/aws" services/gateway-server/internal/ports/
```

## Adapters by Environment

| Adapter | aws-sim | local-sim | local-lab | local-pilot |
|---------|---------|-----------|-----------|-------------|
| CameraPort | SimulatedCamera | SimulatedCamera | GenericHttp | VendorSpecific |
| SensorPort | SimulatedSensor | SimulatedSensor | Modbus | Modbus |
| EquipmentInputPort | SimulatedEquipment | SimulatedEquipment | Modbus | Modbus |
| OutputPort | SimulatedOutput | SimulatedOutput | ModbusTcp | ModbusTcp |
| MediaPort | SimulatedMedia | SimulatedMedia | MediaMTX | MediaMTX |
| CloudSyncPort | AwsIoT | Disabled | Disabled | AwsIoT |
| StoragePort | SQLite | SQLite | SQLite | SQLite |

## Migration Path

To move from AWS simulation to local Ubuntu:
1. Install same .deb package
2. Change `SAFEGAI_PROFILE` from `aws-sim` to target profile
3. Connect hardware (cameras, Modbus devices)
4. Start service with `systemctl start safegai-edge`

No code changes, no recompilation required.
