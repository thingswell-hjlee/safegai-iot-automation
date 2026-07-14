# SafeGAI MVP Module Plan

## Overview

This document defines the modular decomposition, dependency graph, wave execution plan,
and safety boundaries for the SafeGAI MVP platform. All modules are developed as part of
a single Go modular-monolith gateway binary, with a TypeScript cloud backend and React frontend.

Each module has a clear boundary, explicit interfaces, and declared dependencies.
Shared contracts reside in the `contracts/` directory and must not be duplicated.

---

## Dependency Graph (ASCII)

```
                         +------------------+
                         |   M07 Gateway    |
                         |   Core (Wave 2)  |
                         +--------+---------+
                                  |
            +---------------------+---------------------+
            |         |           |           |         |
            v         v           v           v         v
  +-------+---+ +----+----+ +----+----+ +----+----+ +--+------+
  | M02       | | M03     | | M04     | | M05     | | M06     |
  | Camera    | | Device  | | Zone    | | Activity| | Output  |
  | Adapter   | | Sensor  | | State   | | Engine  | | Alarm   |
  | (Wave 1)  | | (Wave 1)| | (Wave 1)| | (Wave 1)| | (Wave 1)|
  +-+---+-----+ +--+------+ +--+--+---+ +--+------+ +--+------+
    |   |           |           |  |        |            |
    |   |           |           |  +--------+            |
    |   |           |           |  (zone+equip)          |
    |   |           +-----+-----+                        |
    |   |                 |                              |
    v   v                 v                              v
  +-+---+-----+   +------+------+               +------+------+
  | M08       |   | M10         |               | M10         |
  | Media     |   | Operations  |               | Operations  |
  | Gateway   |   | Maintenance |               | Maintenance |
  | (Wave 1)  |   | (Wave 1)   |               | (Wave 1)    |
  +-----------+   +------+------+               +------+------+
                         |                              |
                         v                              v
                  +------+------+               +------+------+
                  | M09         |               | M01         |
                  | AWS Cloud   |               | Hybrid App  |
                  | (Wave 1)    |               | (Wave 0)    |
                  +------+------+               +------+------+
                         |                              |
                         |                              |
    +--------------------+------------------------------+
    |                    |                              |
    v                    v                              v
  +-+--------------------+------------------------------+--+
  |                  M00 Foundation (Wave 0)                |
  |    contracts, schemas, event envelope, error model      |
  +-----------+--------------------------------------------+
              |
              v
  +-----------+--------------------------------------------+
  |                  M11 Quality (Wave 0)                   |
  |    test infra, contract validation, scenario runner     |
  +---------------------------------------------------------+
```

### Simplified Layer Diagram

```
Wave 2:  [M07 Gateway Core - Integration]
              |
Wave 1:  [M02 Camera] [M03 Device] [M04 Zone] [M05 Activity] [M06 Output]
         [M08 Media]  [M09 AWS]    [M10 Ops/Maint]
              |
Wave 0:  [M00 Foundation] [M01 Hybrid App Mock] [M11 Quality]
```

---

## Wave Execution Plan

### Wave 0 - Foundation (No dependencies)

| Module | Purpose |
|--------|---------|
| M00 Foundation | Shared contracts, schemas, event envelope, error model, Go project scaffold |
| M01 Hybrid App Mock | React/TypeScript frontend skeleton with mock API, role-based views |
| M11 Quality Framework | Test infrastructure, contract validation, scenario testing framework |

Wave 0 establishes the shared language, build system, and verification baseline.
All subsequent waves depend on Wave 0 contracts.

### Wave 1 - Domain Modules (Depends on Wave 0)

| Module | Purpose |
|--------|---------|
| M02 Camera Adapter | Camera event adapter with vendor abstraction and event normalization |
| M03 Device and Sensor Input | DI/DO Modbus interface, equipment state management |
| M04 Zone State Engine | Occupancy state machine (R3 safety-critical) |
| M05 Activity Engine | Fixed safety rule evaluator combining zone + equipment state (R3) |
| M06 Output and Alarm | Actuation service for warning lights, sirens, stop requests (R3) |
| M08 Media Gateway | MediaMTX stream proxy management for up to 4 cameras |
| M09 AWS Cloud | Lambda handlers, CDK infrastructure, IoT Core, DynamoDB |
| M10 Operations and Maintenance | SQLite storage, event/audit store, cloud outbox, health, local API, RBAC |

Wave 1 modules can be developed in parallel. Each module depends on M00 contracts
and M11 test infrastructure but not on each other (interface via contracts).

### Wave 2 - Integration (Depends on Wave 0 + Wave 1)

| Module | Purpose |
|--------|---------|
| M07 Gateway Core | Integration module connecting all gateway modules into single binary |

Wave 2 wires all modules together, validates the vertical function end-to-end,
and runs integration, performance, and failure recovery tests.

---

## Module Details

### M00 Foundation

| Attribute | Value |
|-----------|-------|
| **Purpose** | Shared contracts, JSON schemas, event envelope format, error model, Go project scaffold, build system |
| **Wave** | 0 |
| **Risk Class** | R1 (infrastructure) |
| **Dependencies** | None (root module) |
| **Inputs** | Product spec requirements, safety rules, API definitions |
| **Outputs** | JSON Schema files, Go types, TypeScript types, Makefile targets, CI config |
| **Key Interfaces** | Event envelope struct, error codes, contract validation functions |

**Responsibilities:**
- Define `contracts/events/camera-event-v1.schema.json`
- Define `contracts/events/occupancy-state-v1.schema.json`
- Define `contracts/events/equipment-state-v1.schema.json`
- Define `contracts/events/safety-decision-v1.schema.json`
- Define `contracts/events/actuation-result-v1.schema.json`
- Define `contracts/events/cloud-event-v1.schema.json`
- Define `contracts/api/local-openapi.yaml`
- Define Go event envelope with common fields: `schemaVersion`, `eventId`, `correlationId`, `tenantId`, `siteId`, `gatewayId`, `deviceId`, `zoneId`, `observedAt`, `receivedAt`, `sequenceNo`, `source`, `quality`, `rawReference`
- Define standard error model and error codes
- Provide Go module scaffold (`services/gateway-server/`)
- Provide Makefile with `check-prereqs`, `format`, `lint`, `test`, `build`, `verify-fast`, `verify`

---

### M01 Hybrid App

| Attribute | Value |
|-----------|-------|
| **Purpose** | React/TypeScript frontend with role-based views (User/Operator/Maintainer), mock API for development |
| **Wave** | 0 (mock), Wave 2 (real API connection) |
| **Risk Class** | R1 (UI, no safety logic) |
| **Dependencies** | M00 (API contract, event schemas) |
| **Inputs** | Local REST API responses, WebSocket real-time events, video stream URLs |
| **Outputs** | User actions (ACK, resolve, work window), rendered safety status display |
| **Key Interfaces** | REST client to `/api/v1/*`, WebSocket client to `/api/v1/realtime`, role-based route guards |

**Responsibilities:**
- Implement User Mode: safety status, 1/2/4-split video, zone occupancy, equipment state, active alarms
- Implement Operator Mode: event queue, ACK/resolve/classify, work window management, reports
- Implement Maintainer Mode: camera registration, zone mapping, DI/DO state, DO test, diagnostics
- Role-based navigation and API permission enforcement
- Mock API server for Wave 0 development (returns contract-compliant responses)
- Responsive layout for control room display and mobile

---

### M02 Camera Adapter

| Attribute | Value |
|-----------|-------|
| **Purpose** | Camera event adapter with vendor abstraction layer, event normalization, health monitoring |
| **Wave** | 1 |
| **Risk Class** | R2 (input to safety chain) |
| **Dependencies** | M00 (event envelope, camera-event schema) |
| **Inputs** | Vendor-specific camera AI events (HTTP push / MQTT / ONVIF), camera health status |
| **Outputs** | Normalized `CameraEvent` conforming to `camera-event-v1.schema.json` |
| **Key Interfaces** | `CameraAdapter` interface: `Connect()`, `Health()`, `SubscribeEvents()`, `GetSnapshot()`, `Close()` |

**Responsibilities:**
- Vendor abstraction: each vendor in `internal/adapters/camera/<vendor>/`
- Event normalization: zone ID, person count, enter/exit, dwell, confidence, timestamp
- Duplicate suppression (2-second window)
- Sequence and timestamp validation
- Reject queue for malformed payloads
- Camera offline detection (30-second timeout)
- Camera failure produces `UNKNOWN` state, never `VACANT`
- Snapshot acquisition for event evidence

---

### M03 Device and Sensor Input

| Attribute | Value |
|-----------|-------|
| **Purpose** | DI/DO Modbus TCP/RTU interface, equipment running/stopped state management |
| **Wave** | 1 |
| **Risk Class** | R2 (input to safety chain) |
| **Dependencies** | M00 (equipment-state schema, event envelope) |
| **Inputs** | Modbus TCP/RTU coils and discrete inputs (8 DI / 8 DO) |
| **Outputs** | Normalized `EquipmentState` events: RUNNING, STOPPED, RESTART_REQUESTED, UNKNOWN |
| **Key Interfaces** | `ModbusClient` interface, `EquipmentStateProvider` interface, register map configuration |

**Responsibilities:**
- Modbus TCP and RTU polling with configurable interval
- Equipment state mapping: DI bit patterns to logical states
- TTL and quality management for input freshness
- Stale input transitions to `UNKNOWN`
- Output feedback reading (DO state confirmation)
- Device offline detection and error reporting
- Register map documentation per device profile
- No direct 24V GPIO usage (isolated Modbus only)

---

### M04 Zone State Engine

| Attribute | Value |
|-----------|-------|
| **Purpose** | Occupancy state machine managing zone vacancy confirmation with fail-safe behavior |
| **Wave** | 1 |
| **Risk Class** | R3 (SAFETY-CRITICAL) |
| **Dependencies** | M00 (occupancy-state schema, event envelope), M02 (camera events - via contract) |
| **Inputs** | Normalized camera events from M02 |
| **Outputs** | Zone occupancy state transitions: OCCUPIED, VACANT_PENDING, VACANT_CONFIRMED, UNKNOWN, STALE |
| **Key Interfaces** | `ZoneStateMachine` interface, state transition events, configurable timing parameters |

**Safety Rules:**
- `VACANT_CONFIRMED` is the ONLY valid vacancy state for safety decisions
- `UNKNOWN` and `STALE` must fail-safe (no restart allowed, no vacancy claim)
- Camera failure = `UNKNOWN`, never `VACANT`
- Camera offline (30s) transitions any zone to `STALE`
- Data timeout cannot directly produce `VACANT_CONFIRMED`

**State Transitions:**
```
UNKNOWN -> OCCUPIED (person detected)
UNKNOWN -> VACANT_PENDING (no person, camera healthy)
OCCUPIED -> VACANT_PENDING (no person for threshold start)
VACANT_PENDING -> VACANT_CONFIRMED (3s or 3 consecutive samples confirmed)
VACANT_PENDING -> OCCUPIED (person detected during pending)
ANY -> STALE (no fresh data for 10s)
STALE -> OCCUPIED or VACANT_PENDING (fresh data received)
```

**Forbidden Transitions:**
- Data timeout directly to `VACANT_CONFIRMED`
- Camera offline to `VACANT_CONFIRMED`
- Count parsing failure to `VACANT_CONFIRMED`

---

### M05 Activity Engine

| Attribute | Value |
|-----------|-------|
| **Purpose** | Fixed safety rule evaluator combining zone occupancy and equipment state to produce safety decisions |
| **Wave** | 1 |
| **Risk Class** | R3 (SAFETY-CRITICAL) |
| **Dependencies** | M00 (safety-decision schema), M04 (zone state - via contract), M03 (equipment state - via contract) |
| **Inputs** | Zone occupancy state (from M04), equipment state (from M03) |
| **Outputs** | Safety decisions: WARNING, STOP_REQUEST_REQUIRED, RESTART_INTERLOCK, SAFETY_CONFIRMATION_UNAVAILABLE |
| **Key Interfaces** | `SafetyRuleEvaluator` interface, rule templates R-01 through R-05 |

**Safety Rules Implemented:**

| Rule | Condition | Decision |
|------|-----------|----------|
| R-01 | Zone=OCCUPIED AND Equipment=RUNNING | WARNING + STOP_REQUEST_REQUIRED |
| R-02 | RestartRequested AND Zone != VACANT_CONFIRMED | RESTART_INTERLOCK |
| R-03 | Camera/Occupancy=UNKNOWN or STALE | SAFETY_CONFIRMATION_UNAVAILABLE |
| R-04 | ApprovedWorkWindow AND Equipment=STOPPED | MAINTENANCE_MONITORING |
| R-05 | Duplicate event within time window | Suppress (single output only) |

**Critical Constraints:**
- Rules are fixed templates, not user-configurable in MVP
- No cloud-originated machine control
- Stop requests go to PLC/Safety Relay only
- No ambiguous safety requirements resolved without human approval

---

### M06 Output and Alarm

| Attribute | Value |
|-----------|-------|
| **Purpose** | Actuation service for warning lights, sirens, audio announcements, and PLC/Safety Relay stop requests |
| **Wave** | 1 |
| **Risk Class** | R3 (SAFETY-CRITICAL) |
| **Dependencies** | M00 (actuation-result schema), M05 (safety decisions - via contract), M03 (DO interface - via contract) |
| **Inputs** | Safety decisions from M05 |
| **Outputs** | Physical actuations via isolated Modbus DO: warning light, siren, stop request pulse |
| **Key Interfaces** | `ActuatorService` interface, `OutputCommand` with correlation ID, timeout, and retry policy |

**Safety Rules:**
- Stop requests go to PLC or Safety Relay input only
- No direct machine main power switching via DO
- No automatic re-execution of previous commands after reboot
- No infinite retry on missing ACK (limited retry with escalation)
- Duplicate output prevention within suppression window
- Command ID and correlation ID for full traceability
- Optional feedback input for result confirmation
- All actuations recorded in audit store

**Forbidden:**
- Boot-time replay of historical output commands
- General DO used for main power cutoff
- App-initiated direct equipment power shutdown

---

### M07 Gateway Core

| Attribute | Value |
|-----------|-------|
| **Purpose** | Integration module connecting all gateway modules into a single Go binary with lifecycle management |
| **Wave** | 2 |
| **Risk Class** | R2 (integration, relies on R3 modules for safety logic) |
| **Dependencies** | M00, M02, M03, M04, M05, M06, M08, M10 (all gateway modules) |
| **Inputs** | Configuration, hardware profile, module initialization order |
| **Outputs** | Running gateway process with health endpoints, graceful shutdown, systemd integration |
| **Key Interfaces** | Module registry, dependency injection, lifecycle hooks (Start/Stop/Health) |

**Responsibilities:**
- `cmd/safegai-edge/main.go` entry point
- Configuration loading and validation
- Module initialization in dependency order
- Graceful shutdown with SIGTERM handling
- systemd watchdog notification
- Health endpoints: `/health/live`, `/health/ready`
- Hardware profile verification at startup
- Build version embedding
- Resource budget monitoring (CPU, RAM, disk)
- Boot-to-safety-ready target: 3 minutes

---

### M08 Media Gateway

| Attribute | Value |
|-----------|-------|
| **Purpose** | MediaMTX RTSP-to-WebRTC/HLS stream proxy management for up to 4 cameras |
| **Wave** | 1 |
| **Risk Class** | R1 (video display, no safety logic) |
| **Dependencies** | M00 (configuration schema) |
| **Inputs** | Camera RTSP URLs, stream configuration |
| **Outputs** | WebRTC/HLS stream endpoints for frontend consumption |
| **Key Interfaces** | `StreamManager` interface: `AddStream()`, `RemoveStream()`, `StreamStatus()`, `Health()` |

**Responsibilities:**
- MediaMTX configuration management
- RTSP source pull from cameras (H.264 sub-stream)
- WebRTC delivery (preferred), HLS fallback
- Codec copy / pass-through only (no transcoding)
- On-demand connection (no persistent stream when no viewer)
- Up to 4 simultaneous camera streams
- Stream health monitoring
- Camera credentials stored in local encrypted configuration only
- Resource budget: MediaMTX + 4 streams under 1GB RAM

---

### M09 AWS Cloud

| Attribute | Value |
|-----------|-------|
| **Purpose** | Lambda handlers, CDK infrastructure, IoT Core MQTT, DynamoDB event storage, S3 thumbnails, SNS notifications |
| **Wave** | 1 |
| **Risk Class** | R1 (cloud telemetry, no machine control) |
| **Dependencies** | M00 (cloud-event schema, API contract) |
| **Inputs** | MQTT messages from gateway outbox (status, events, images, acks) |
| **Outputs** | DynamoDB records, S3 thumbnails, SNS notifications, Cloud REST API responses |
| **Key Interfaces** | `ingest-handler` Lambda, `admin-api-handler` Lambda, IoT Rules, Cognito auth |

**Responsibilities:**
- CDK stacks: foundation, iot, data, api, web
- IoT Core: Thing, Certificate, Policy, Topic rules
- MQTT topics: `safegai/v1/{tenant}/{site}/{gateway}/{type}`
- DynamoDB: `Gateways` table (PK: tenantId, SK: siteId#gatewayId), `Events` table (PK: tenantId#siteId, SK: detectedAt#eventId)
- S3: event thumbnails (max 96KB JPEG), frontend hosting
- Cognito: operator and maintainer groups with tenant/site claims
- SNS: email and optional SMS notification
- Idempotent event ingestion
- Hardware profile allowlist validation
- No machine-control endpoints
- No cloud-to-device actuator commands

---

### M10 Operations and Maintenance

| Attribute | Value |
|-----------|-------|
| **Purpose** | SQLite WAL storage, event/audit store, cloud outbox queue, health monitoring, local REST/WebSocket API, RBAC |
| **Wave** | 1 |
| **Risk Class** | R2 (data integrity, audit trail) |
| **Dependencies** | M00 (API contract, event schemas, error model) |
| **Inputs** | Domain events from all gateway modules, API requests from frontend |
| **Outputs** | Persisted events/audit, API responses, WebSocket real-time updates, outbox queue for cloud sync |
| **Key Interfaces** | `EventStore`, `AuditStore`, `OutboxQueue`, `LocalAPI` (REST + WebSocket), `AuthMiddleware` |

**Responsibilities:**
- SQLite WAL database with migration versioning
- Tables: events, occupancy_states, equipment_states, safety_decisions, actuation_results, audit_logs, cloud_outbox, config_versions, users
- All timestamps stored as UTC
- Event ID uniqueness enforcement
- Cloud outbox: transactional insert, exponential backoff, max queue alert, dead letter state
- Local REST API: `/api/v1/*` endpoints per product spec
- WebSocket: `/api/v1/realtime` for live state updates
- RBAC: User, Operator, Maintainer roles with API-level enforcement
- Health monitoring: CPU, RAM, disk, temperature, SSD health, NIC status
- 30-day local event retention with disk quota management
- Backup and restore support

---

### M11 Quality and Reliability

| Attribute | Value |
|-----------|-------|
| **Purpose** | Test infrastructure, JSON Schema contract validation, scenario testing framework, CI verification targets |
| **Wave** | 0 |
| **Risk Class** | R1 (test tooling) |
| **Dependencies** | M00 (schemas to validate against) |
| **Inputs** | Contract schemas, scenario definitions, module outputs |
| **Outputs** | Test results, validation reports, CI artifacts |
| **Key Interfaces** | `ContractValidator`, `ScenarioRunner`, Makefile test targets |

**Responsibilities:**
- JSON Schema validation library for contract testing
- Scenario runner: YAML/JSON scenario files with timed event sequences
- Camera simulator: occupied, vacant, count, enter, exit, dwell, duplicate, out-of-order, delayed, offline, malformed
- I/O simulator: running, stopped, restart request, work window, stop output, feedback, timeout, modbus exception, network drop
- Contract test target: `make test-contract`
- Go unit test helpers and fixtures
- TypeScript test utilities for Lambda and frontend
- CI integration: `make verify-fast` (quick) and `make verify` (full pre-PR)

---

## Shared Contracts (contracts/ directory)

All modules reference the single `contracts/` directory as the source of truth.
No module may duplicate contract definitions locally.

```
contracts/
  events/
    camera-event-v1.schema.json       # M02 output, M04 input
    occupancy-state-v1.schema.json    # M04 output, M05 input
    equipment-state-v1.schema.json    # M03 output, M05 input
    safety-decision-v1.schema.json    # M05 output, M06 input
    actuation-result-v1.schema.json   # M06 output
    cloud-event-v1.schema.json        # M10 outbox, M09 input
  api/
    local-openapi.yaml                # M10 API, M01 client
    cloud-openapi.yaml                # M09 API, M01 cloud client
  mqtt/
    topics.md                         # M09/M10 MQTT topic contract
  safety/
    README.md                         # R3 change approval requirements
    occupancy-rules.md                # State machine rules (approved)
    safety-decision-rules.md          # Rule templates R-01 to R-05
```

### Common Event Envelope Fields

Every event schema includes:

| Field | Type | Description |
|-------|------|-------------|
| schemaVersion | string | Schema version identifier |
| eventId | string (UUID) | Unique event identifier |
| correlationId | string (UUID) | Correlation chain identifier |
| tenantId | string | Tenant identifier |
| siteId | string | Site identifier |
| gatewayId | string | Gateway identifier |
| deviceId | string | Source device identifier |
| zoneId | string | Zone identifier |
| observedAt | string (ISO 8601) | When the event was observed |
| receivedAt | string (ISO 8601) | When the gateway received the event |
| sequenceNo | integer | Monotonic sequence number |
| source | string | Event source module |
| quality | string | Data quality indicator |
| rawReference | string | Reference to raw payload (if stored) |

---

## Safety Boundaries

### R3 Safety-Critical Modules

The following modules implement safety logic and require T1 + T2 human approval
for any change. PRs modifying R3 modules must not be merged without explicit
human verification evidence.

| Module | Safety Function | Critical Rule |
|--------|----------------|---------------|
| M04 Zone State Engine | Occupancy determination | VACANT_CONFIRMED is the only valid vacancy |
| M05 Activity Engine | Safety rule evaluation | Combines zone + equipment for stop decisions |
| M06 Output and Alarm | Physical actuation | Stop request to PLC/Safety Relay only |

### Key Safety Invariants

1. **VACANT_CONFIRMED is the only valid vacancy state** - No other state permits restart or clears safety interlocks.
2. **UNKNOWN and STALE must fail-safe** - These states never allow restart and never claim vacancy.
3. **No cloud machine control** - AWS cannot send actuator commands to the gateway.
4. **Stop requests go to PLC/Safety Relay only** - No direct main power switching.
5. **Camera failure = UNKNOWN, not VACANT** - Camera offline or error never produces vacancy.
6. **No boot-time output replay** - Previous actuations are not automatically re-executed after restart.
7. **No app-initiated power cutoff** - Frontend cannot directly execute equipment power shutdown.

### Risk Classification

| Class | Definition | Modules | Approval |
|-------|-----------|---------|----------|
| R3 | Safety-critical: incorrect behavior causes physical harm risk | M04, M05, M06 | T1 + T2 human approval required |
| R2 | Safety-adjacent: provides inputs to safety chain or data integrity | M02, M03, M07, M10 | D1 review + T1 or T2 evidence |
| R1 | Standard: no direct safety impact | M00, M01, M08, M09, M11 | D1 review |

---

## Module Dependency Matrix

| Module | Depends On | Depended By |
|--------|-----------|-------------|
| M00 Foundation | (none) | All modules |
| M01 Hybrid App | M00, M10 (API contract) | (none - leaf consumer) |
| M02 Camera Adapter | M00 | M04, M08 |
| M03 Device/Sensor | M00 | M05, M06 |
| M04 Zone State Engine | M00, M02 (via contract) | M05 |
| M05 Activity Engine | M00, M04 (via contract), M03 (via contract) | M06 |
| M06 Output/Alarm | M00, M05 (via contract), M03 (DO interface) | (none - leaf actuator) |
| M07 Gateway Core | M00, M02, M03, M04, M05, M06, M08, M10 | (none - integration root) |
| M08 Media Gateway | M00 | M01 (stream URLs) |
| M09 AWS Cloud | M00, M10 (outbox contract) | (none - cloud leaf) |
| M10 Ops/Maintenance | M00 | M01, M07, M09 |
| M11 Quality | M00 | All modules (test support) |

Note: Wave 1 modules depend on each other only through contracts defined in M00.
They do not import each other's code directly. This enables parallel development.

---

## Development Rules

1. Each module uses a separate branch and PR.
2. Never push directly to `main`.
3. PRs are created but never merged by automation.
4. No real AWS deployment is performed.
5. No connection to real cameras, PLCs, NVRs, or equipment.
6. No real passwords, certificates, or API keys are generated or required.
7. External equipment is replaced by simulators.
8. Shared contracts use the `contracts/` directory as the single source.
9. The same contract must not be duplicated across modules.
10. Inter-module calls use explicit interfaces.
11. Untested items are not reported as successful.
12. Ambiguous safety requirements are not decided without human approval.
13. AWS does not directly control field equipment.
14. Only `VACANT_CONFIRMED` counts as vacancy.
15. `UNKNOWN` and `STALE` are never treated as vacant.
16. The app does not directly execute equipment power cutoff.
17. Outputs only send stop requests to PLC or Safety Relay.

---

## Conflict Resolution

If a Wave 1 module needs to modify shared files (contracts, Makefile, common types),
it must not change those files directly. Instead, record the proposed change in
`docs/gaps/` as a change proposal for coordinated integration in Wave 2.

---

## Verification Checklist Per Module

Before a module PR is considered complete:

- [ ] `make verify-fast` passes
- [ ] Contract tests pass (`make test-contract`)
- [ ] Unit tests pass (`make test`)
- [ ] No secrets committed
- [ ] R3 modules marked as requiring T1+T2 approval
- [ ] Known limitations documented
- [ ] Integration requirements for next wave documented
