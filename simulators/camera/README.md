# Camera Simulator

The camera simulator provides a software-only replacement for real camera hardware,
enabling end-to-end testing of the SafeGAI edge gateway without physical devices.

## Overview

The simulator implements the `CameraAdapter` interface and generates deterministic
camera events based on configurable scenarios. It supports:

- Person detection events (occupied zone)
- Person not-detected events (zone exit)
- Camera offline events
- Duplicate event generation (for dedup testing)
- Out-of-order event generation
- Malformed event generation (for error handling testing)

## Usage

### Go API

```go
import (
    "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera/simulator"
)

// Create simulator with default config
cfg := simulator.DefaultConfig()
cfg.Scenarios = []simulator.ScenarioFunc{
    simulator.Occupied("zone-A"),
    simulator.Vacant("zone-A"),
}

sim := simulator.New(cfg)
defer sim.Close()

// Connect and subscribe
ctx := context.Background()
sim.Connect(ctx)

ch := make(chan camera.RawCameraEvent, 10)
sim.SubscribeEvents(ctx, ch)

// Consume events
for evt := range ch {
    // Process event...
}
```

### Scenario Files

JSON scenario files in `scenarios/` directory define test sequences:

| File | Description |
|------|-------------|
| `occupied_single.json` | Single person enters and stays in zone |
| `vacant_exit.json` | Person leaves zone (person_not_detected) |
| `camera_offline.json` | Camera goes offline |
| `duplicate_events.json` | Rapid duplicate events within dedup window |
| `out_of_order.json` | Events arriving out of timestamp order |

### Scenario Format

Each scenario file is a JSON object:

```json
{
  "name": "scenario-name",
  "description": "What this scenario tests",
  "cameraId": "cam-sim-001",
  "events": [
    {
      "zoneId": "zone-A",
      "eventType": "person_detected",
      "personCount": 1,
      "confidence": 0.95,
      "timestampOffsetMs": 0
    }
  ]
}
```

#### Fields

- `name`: Unique scenario identifier
- `description`: Human-readable explanation
- `cameraId`: Simulated camera device ID
- `events`: Array of events to generate
  - `zoneId`: Target zone
  - `eventType`: One of `person_detected`, `person_not_detected`, `offline`
  - `personCount`: Number of persons detected
  - `confidence`: Detection confidence (0.0 to 1.0)
  - `timestampOffsetMs`: Milliseconds offset from scenario start time

## Safety Rules

The camera simulator enforces the following safety rules:

1. **NEVER produces `VACANT_CONFIRMED`**: Only the Zone State Engine determines vacancy
   after a configurable timeout period. The camera adapter only reports raw detection events.

2. **Camera offline maps to UNKNOWN**: When a camera goes offline, the system state
   becomes UNKNOWN, never VACANT. This ensures safety by assuming potential occupancy
   when detection is unavailable.

3. **UNKNOWN and STALE are not vacancy**: These states maintain the safety interlock
   to prevent equipment operation when occupancy cannot be confirmed.

## Testing

Run simulator tests:

```bash
cd services/gateway-server
go test ./internal/adapters/camera/simulator/...
```

Run all camera-related tests:

```bash
cd services/gateway-server
go test ./internal/adapters/camera/... ./internal/domain/normalizer/...
```
