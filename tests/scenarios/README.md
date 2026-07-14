# SafeGAI Scenario Tests

## Overview

Scenario tests validate end-to-end safety decision logic by defining
sequences of events and their expected outcomes. Each scenario file
is a JSON document conforming to `schema.json` in this directory.

## Scenario File Format

Each scenario file contains:

- **name**: Human-readable scenario identifier
- **description**: What safety rule or behavior is being tested
- **preconditions**: Initial state of zones, equipment, and cameras
- **steps**: Ordered list of events with timestamps
- **expected_states**: Expected zone/equipment states after steps execute
- **expected_outputs**: Expected safety decisions or actuation commands

## Step Structure

Each step has:
- `timestamp`: ISO 8601 time of the event
- `event_type`: Type of input event (camera_event, equipment_state, etc.)
- `payload`: Event data conforming to the relevant contract schema

## Running Scenarios

Scenarios are validated by the contract test suite:

```bash
make test-contract
```

The scenario runner verifies:
1. Each scenario file is valid JSON
2. Each scenario conforms to the scenario schema
3. Steps reference valid event types
4. Expected outputs reference valid decision types

## Safety Rules Encoded

- OCCUPIED + RUNNING = STOP_REQUEST_REQUIRED
- STALE state blocks equipment restart
- VACANT_CONFIRMED is the only state that permits normal operation
- Camera offline transitions zone to UNKNOWN
- Duplicate events within suppression window are ignored
- UNKNOWN and STALE are never treated as vacancy (non-occupancy)

## Adding New Scenarios

1. Create a new JSON file in this directory
2. Follow the schema defined in `schema.json`
3. Include clear preconditions and expected outcomes
4. Run `make test-contract` to validate
