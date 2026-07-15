# Contract Consolidation Report

## Summary

This report documents the contract consolidation performed during the
integration of PR #2 through #13 into the `integration/mvp-consolidation` branch.

## Canonical Contract Source

**PR #3 (M00 Foundation)** was designated as the canonical baseline for all
contract schemas in `contracts/events/`, `contracts/api/`, and `contracts/errors/`.

## Conflicts Resolved

### 1. OccupancyState Enum

| PR | Values Defined | Resolution |
|----|----------------|------------|
| #3 | OCCUPIED, VACANT_PENDING, VACANT_CONFIRMED, UNKNOWN, STALE | **Canonical** |
| #4 | Same values but in `properties.state.enum` | Discarded; schema kept from #3 using `$defs` |
| #7 | Same values in Go types | Kept canonical from #3 |
| #8 | Same values in local Go type | Retained (values match canonical) |
| #9 | Same values in local Go type | Retained (values match canonical) |
| #13 | Same values in Go types | Discarded; ours kept |

### 2. EquipmentState Enum

| PR | Values Defined | Resolution |
|----|----------------|------------|
| #3 | RUNNING, STOPPED, RESTART_REQUESTED, UNKNOWN | **Updated** to canonical |
| #7 | RUNNING, STOPPED, RESTART_REQUESTED, UNKNOWN | Updated to remove RESTART_REQUESTED |
| #13 | RUNNING, STOPPED, FAULT, UNKNOWN | Discarded; canonical used |

**Canonical values:** RUNNING, STOPPED, STARTING, STOPPING, FAULT, OFFLINE, UNKNOWN

`RESTART_REQUESTED` removed from EquipmentState. Now tracked as a boolean flag
(`RestartRequested`) on the equipment state tracker, representing an operator
request rather than a machine state.

### 3. SafetyDecision Enum

| PR | Values Defined | Resolution |
|----|----------------|------------|
| #3 | SAFE, WARNING, STOP_REQUEST_REQUIRED, RESTART_INTERLOCK, SAFETY_CONFIRMATION_UNAVAILABLE, MAINTENANCE_MONITORING | **Canonical** |
| #4 | STOP_REQUEST_REQUIRED, OPERATION_PERMITTED, HOLD_CURRENT_STATE, EMERGENCY_STOP | **Banned values removed** |
| #9 | Same as #3 (local type) | Retained (values match canonical) |

**Banned values eliminated:** OPERATION_PERMITTED, ALLOW_START, EMERGENCY_STOP

### 4. ActuationCommand Enum

| PR | Values Defined | Resolution |
|----|----------------|------------|
| #3 | STOP_REQUEST, WARNING_LIGHT, WARNING_SIREN, VOICE_ANNOUNCE, DIGITAL_OUTPUT | Changed to DIGITAL_OUTPUT_TEST |
| #10 | WARNING_LIGHT, SIREN, STOP_REQUEST_PULSE, AUDIO_ANNOUNCEMENT (internal) | Internal names retained; external contract uses canonical |

**Canonical values:** STOP_REQUEST, WARNING_LIGHT, WARNING_SIREN, VOICE_ANNOUNCE, DIGITAL_OUTPUT_TEST

### 5. EventEnvelope

| PR | Version | Resolution |
|----|---------|------------|
| #3 | Full envelope with 13 required fields | **Canonical** |
| #4 | Simplified version | Discarded |
| #6 | Same as #3 but with Payload field | Merged: added Payload field to canonical |
| #7-#13 | Various duplicates | Discarded; canonical kept |

### 6. ErrorModel / Domain Errors

| PR | Error Types | Resolution |
|----|-------------|------------|
| #3 | Validation, NotFound, Conflict, Internal, Timeout | Base set |
| #7 | Added: IOFailure, Connection, Modbus | **Merged** (superset) |
| #11 | Added: CapacityLimit | **Merged** |
| #13 | Similar to #7 but without Modbus | Discarded; ours kept |

### 7. Test Fixtures

PR #4 fixtures used snake_case field names and lacked envelope fields.
Updated to:
- Use camelCase per canonical contract
- Include all required EventEnvelope fields
- Reference canonical enum values

### 8. Contract Validation Scripts

- `scripts/run-contract-tests.sh`: Updated to validate against `$defs` structure
  (canonical schemas use JSON Schema `$defs` for enum definitions)
- `tests/contract/validate_contracts.py`: Updated to resolve `$ref` paths and
  validate canonical enums including banned-value checks

## Verification

All contract validations pass:
- 17 schema and fixture validations pass
- 7 Python fixture validation tests pass
- 0 failures
- No banned values detected in any schema
