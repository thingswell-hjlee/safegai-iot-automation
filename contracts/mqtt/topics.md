# SafeGAI MQTT Topic Contract

Version: 1.0
Region: ap-northeast-2
Protocol: MQTT 3.1.1 over mTLS (port 8883)
QoS: 1 (at least once delivery)

## Topic Structure

```
safegai/v1/{tenant}/{site}/{gateway}/status
safegai/v1/{tenant}/{site}/{gateway}/events
safegai/v1/{tenant}/{site}/{gateway}/images/{eventId}
safegai/v1/{tenant}/{site}/{gateway}/acks
```

## Topic Descriptions

### `safegai/v1/{tenant}/{site}/{gateway}/status`

**Direction:** Gateway -> Cloud
**Payload:** JSON
**Purpose:** Periodic heartbeat with gateway health data.
**Frequency:** Configurable (default 60s, range 30-300s).

Fields:
- `gatewayId` (string, required)
- `timestamp` (ISO 8601, required)
- `online` (boolean, required)
- `version` (string, required)
- `hardwareProfileId` (string, required)
- `cpuPercent` (number, required)
- `ramPercent` (number, required)
- `diskPercent` (number, required)
- `temperatureCelsius` (number, optional)
- `outboxDepth` (number, required)
- `cameras` (array of camera summaries)
- `ssdHealth` (object with model, capacityGiB, healthPercent)
- `nicStates` (array of NIC link states)

### `safegai/v1/{tenant}/{site}/{gateway}/events`

**Direction:** Gateway -> Cloud
**Payload:** JSON (event metadata ONLY, no image data)
**Purpose:** Safety event notification with metadata.

Fields (see contracts/schemas/ for full schema):
- `eventId` (UUID v4, required)
- `idempotencyKey` (string, required)
- `detectedAt` (ISO 8601, required)
- `tenantId` (string, required)
- `siteId` (string, required)
- `gatewayId` (string, required)
- `cameraId` (string, required)
- `zoneId` (string, required)
- `severity` (enum: critical|high|medium|low|info)
- `occupancy` (enum: OCCUPIED|VACANT_CONFIRMED|UNKNOWN|STALE)
- `equipmentState` (enum: RUNNING|STOPPED|FAULT|UNKNOWN)
- `actions` (array of action objects)
- `imageKey` (string, optional - S3 key reference)
- `description` (string, optional)
- `schemaVersion` (literal "1.0")

**CRITICAL:**
- Image data is NEVER embedded in this JSON payload.
- No base64-encoded image fields.
- Images are sent separately on the `images/{eventId}` topic.

### `safegai/v1/{tenant}/{site}/{gateway}/images/{eventId}`

**Direction:** Gateway -> Cloud
**Payload:** Raw JPEG binary (NOT JSON, NOT base64)
**Maximum size:** 96KB
**Purpose:** Event thumbnail image stored directly to S3 via IoT Rule.

Constraints:
- Content type: image/jpeg
- Maximum payload: 96KB (98304 bytes)
- One image per eventId
- Stored to S3 key: `{tenant}/{site}/{yyyy}/{mm}/{dd}/{eventId}.jpg`
- Local full-resolution images are NOT auto-uploaded to cloud.

### `safegai/v1/{tenant}/{site}/{gateway}/acks`

**Direction:** Cloud -> Gateway
**Payload:** JSON
**Purpose:** Event acknowledgment and setting updates from cloud.

Fields:
- `type` (enum: event_ack|settings_update)
- `eventId` (string, for event_ack type)
- `ackStatus` (enum: acknowledged|resolved)
- `ackBy` (string)
- `ackAt` (ISO 8601)
- `settings` (object, for settings_update type, non-safety allowlist ONLY)

## Topic Policy Scope

Each gateway's IoT Policy restricts:
- **Connect:** Only with its own Thing Name as client ID
- **Publish:** Only to its own tenant/site/gateway path
- **Subscribe:** Only to its own acks topic
- **Receive:** Only from its own acks topic
- **Shadow:** Only its own `health` and `settings` named shadows

## Forbidden Topics and Commands

The following DO NOT EXIST and MUST NEVER be created:

- `safegai/v1/{tenant}/{site}/{gateway}/control/*` - NO machine control
- `safegai/v1/{tenant}/{site}/{gateway}/actuator/*` - NO actuator commands
- `safegai/v1/{tenant}/{site}/{gateway}/command/*` - NO remote commands
- `safegai/v1/{tenant}/{site}/{gateway}/stop` - NO cloud-initiated stop
- `safegai/v1/{tenant}/{site}/{gateway}/restart` - NO cloud-initiated restart
- Any topic that sends safety I/O mapping changes
- Any topic that sends BIOS/OS update commands
- Any topic that sends occupancy override commands

## Device Shadow Contract

### Named Shadow: `health` (Reported Only)

Gateway reports health telemetry. Cloud reads only.
Cloud CANNOT write desired state to the health shadow.

### Named Shadow: `settings` (Desired/Reported)

**Allowlist (permitted in desired/reported):**
- `heartbeatIntervalSec` (number, 30-300)
- `cloudThumbnailEnabled` (boolean)
- `logLevel` (enum: debug|info|warn|error)
- `logLevelExpiresAt` (ISO 8601)
- `notificationPolicyVersion` (string)

**Forbidden (NEVER in shadow):**
- Safety rules
- I/O mapping
- Stop-request behavior
- Restart interlock
- BIOS version changes
- OS image changes
- Occupancy override
- Equipment state override
- Camera credentials

## Security Invariants

1. Gateway identity is bound to X.509 certificate via mTLS.
2. Topic policy enforces per-gateway publish/subscribe scope.
3. Certificate ID is verified against registered gateway in ingest handler.
4. No long-lived AWS access keys are used.
5. Cloud CANNOT send safety commands to gateway.
6. Hardware inventory is informational; never used for remote command execution.
