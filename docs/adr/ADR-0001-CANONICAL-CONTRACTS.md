# ADR-0001: Canonical Contracts for SafeGAI IoT Platform

## Status

Accepted

## Date

2024-07-14

## Context

Multiple PR branches (PR #2 through #13) were developed in parallel, each defining
their own versions of domain types, event schemas, and error models. This resulted
in conflicting definitions of OccupancyState, EquipmentState, SafetyDecision,
ActuationCommand, EventEnvelope, and ErrorModel across modules.

A single canonical source of truth is required to ensure:
- Safety-critical decisions are based on a well-defined, auditable state model
- API consumers see consistent JSON schemas
- Internal domain code uses validated state transitions
- No ambiguity exists about what constitutes "vacancy" or "safe to restart"

## Decision

PR #3 (M00 Foundation) contracts are designated as the canonical baseline. All other
modules MUST reference or comply with these schemas rather than defining their own.

### External JSON and API Conventions

- All external-facing JSON uses **camelCase** field names
- API versioning: `/api/v1`
- Timestamps: RFC 3339 UTC (e.g., `2024-01-01T12:00:00.000Z`)
- Event IDs: UUID v4

### OccupancyState (Canonical Enum)

| Value | Meaning | Vacancy? |
|-------|---------|----------|
| OCCUPIED | Person(s) detected in zone | No |
| VACANT_PENDING | Exit detected, awaiting confirmation | No |
| VACANT_CONFIRMED | Vacancy confirmed by required cameras | **Yes (only this)** |
| UNKNOWN | Camera offline or conflicting data | No |
| STALE | No update received within staleness threshold | No |

**Rules:**
- Only `VACANT_CONFIRMED` satisfies vacancy for safety decisions
- `UNKNOWN` is NEVER treated as vacancy
- `STALE` is NEVER treated as vacancy
- Camera offline produces `UNKNOWN`
- Stale events MUST NOT trigger state changes or output commands
- No automatic restart is permitted

### EquipmentState (Canonical Enum)

| Value | Meaning |
|-------|---------|
| RUNNING | Equipment is actively operating |
| STOPPED | Equipment is at rest |
| STARTING | Equipment is in startup sequence |
| STOPPING | Equipment is in shutdown sequence |
| FAULT | Equipment has a fault condition |
| OFFLINE | Equipment communication lost |
| UNKNOWN | State cannot be determined |

**`RESTART_REQUESTED` is NOT an EquipmentState.** It is managed as a separate
Operator Request or Audit Event.

### SafetyDecision (Canonical Enum)

| Value | Meaning |
|-------|---------|
| SAFE | Normal operation, no safety concerns |
| WARNING | Potential risk, advisory only |
| STOP_REQUEST_REQUIRED | Must send stop request to PLC/Safety Relay |
| RESTART_INTERLOCK | Restart blocked until conditions are met |
| SAFETY_CONFIRMATION_UNAVAILABLE | Cannot confirm safety (camera offline, stale) |
| MAINTENANCE_MONITORING | Approved maintenance window active |

**Banned values (MUST NOT appear in any schema or code):**
- `OPERATION_PERMITTED`
- `ALLOW_START`
- `EMERGENCY_STOP`
- Any command that implies automatic restart
- Any command that implies direct power cutoff

### ActuationCommand (Canonical Enum)

| Value | Target |
|-------|--------|
| STOP_REQUEST | PLC or SAFETY_RELAY only |
| WARNING_LIGHT | Visual alarm |
| WARNING_SIREN | Audible alarm |
| VOICE_ANNOUNCE | PA system announcement |
| DIGITAL_OUTPUT_TEST | Test mode only |

**`STOP_REQUEST` target is restricted to PLC or SAFETY_RELAY.**
No direct machine power switching is permitted.

## Consequences

- All modules that previously defined their own enums are updated to reference
  the canonical `contracts/events/` schemas
- Go code uses `services/gateway-server/internal/domain/events` package as
  the single source of truth for type constants
- Contract validation tests (`make test-contract`) enforce canonical values
- Internal packages (safety, occupancy, actuation) may use local type aliases
  but values MUST match canonical definitions
- Any future PR that introduces non-canonical values will fail contract tests

## References

- `contracts/events/occupancy-state-v1.schema.json`
- `contracts/events/equipment-state-v1.schema.json`
- `contracts/events/safety-decision-v1.schema.json`
- `contracts/events/actuation-result-v1.schema.json`
- `contracts/events/event-envelope-v1.schema.json`
- `services/gateway-server/internal/domain/events/types.go`
- `tests/contract/validate_contracts.py`
