# I/O Simulator

The I/O simulator provides a test replacement for real Modbus TCP/RTU
digital I/O modules. It emulates an isolated 8DI/8DO module that
communicates equipment status from a PLC or Safety Relay.

## Purpose

- Replace real Modbus hardware in development and testing environments
- Enable scenario-based testing of equipment state transitions
- Verify safety constraints without physical equipment
- Test I/O failure modes and staleness detection

## Architecture

```
+-------------------+       +----------------+       +------------------+
| Equipment Manager | <---- | IOAdapter I/F  | <---- | I/O Simulator    |
| (Domain)          |       | (Port)         |       | (Test Adapter)   |
+-------------------+       +----------------+       +------------------+
```

The simulator implements the `IOAdapter` interface defined in
`services/gateway-server/internal/adapters/io/interface.go`.

## Digital Input Mapping

| Address | Signal              | Description                               |
|---------|---------------------|-------------------------------------------|
| DI[0]   | Equipment Running   | true = running signal from PLC            |
| DI[1]   | Restart Requested   | true = restart request from operator      |
| DI[2]   | Output Feedback     | true = PLC/Safety Relay acknowledged DO   |
| DI[3]   | Reserved            | Future use                                |
| DI[4]   | Reserved            | Future use                                |
| DI[5]   | Reserved            | Future use                                |
| DI[6]   | Reserved            | Future use                                |
| DI[7]   | Reserved            | Future use                                |

## Digital Output Mapping

| Address | Signal              | Description                               |
|---------|---------------------|-------------------------------------------|
| DO[0]   | Stop Request        | Request PLC/Safety Relay to stop          |
| DO[1]   | Restart Permit      | Permit PLC/Safety Relay to restart        |
| DO[2]   | Reserved            | Future use                                |
| DO[3]   | Reserved            | Future use                                |
| DO[4]   | Reserved            | Future use                                |
| DO[5]   | Reserved            | Future use                                |
| DO[6]   | Reserved            | Future use                                |
| DO[7]   | Reserved            | Future use                                |

## Safety Constraints

- **No direct machine power control.** All DO outputs go to PLC or Safety Relay only.
- **I/O failure is never treated as normal state.** Communication errors produce UNKNOWN.
- **Stale DI input produces UNKNOWN equipment state.**
- **USB Relay is not acceptable** for production output.
- **Only external isolated I/O modules** are supported (no PC GPIO).

## Predefined Scenarios

See `scenarios/` directory for JSON scenario definitions:

| Scenario              | File                       | Description                        |
|-----------------------|----------------------------|------------------------------------|
| Equipment Running     | equipment_running.json     | DI[0]=true, equipment is running   |
| Equipment Stopped     | equipment_stopped.json     | DI[0]=false, equipment is stopped  |
| Restart Requested     | restart_requested.json     | DI[1]=true, restart pending        |
| Modbus Offline        | modbus_offline.json        | Communication failure              |
| Output Feedback       | output_feedback.json       | DI[2]=true after DO write          |

## Usage in Tests

```go
import (
    "context"
    ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
    "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io/simulator"
)

func TestEquipmentRunning(t *testing.T) {
    sim := simulator.New(simulator.DefaultConfig())
    scenario := simulator.EquipmentRunning()
    simulator.ApplyScenario(sim, scenario)

    ctx := context.Background()
    sim.Connect(ctx, ioAdapter.DefaultIOConfig())

    states, _ := sim.ReadAllDI(ctx)
    // states[0].Value == true (running)
}
```

## Equipment State Derivation

| DI[0] (Running) | DI[1] (Restart) | Quality | Derived State       |
|-----------------|-----------------|---------|---------------------|
| true            | false           | GOOD    | RUNNING             |
| false           | false           | GOOD    | STOPPED             |
| any             | true            | GOOD    | RESTART_REQUESTED   |
| any             | any             | BAD     | UNKNOWN             |
| any             | any             | STALE   | UNKNOWN             |
| (no data)       | (no data)       | -       | UNKNOWN             |
