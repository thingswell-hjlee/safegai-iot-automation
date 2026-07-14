package simulator

import (
	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
)

// Scenario represents a predefined I/O state for testing.
// Each scenario configures the simulator DI/DO values and
// any fault injection settings.
type Scenario struct {
	// Name is the human-readable scenario identifier.
	Name string

	// Description explains what this scenario represents.
	Description string

	// DIValues are the digital input states for points 0-7.
	DIValues [ioAdapter.NumDIPoints]bool

	// Config is the simulator fault injection configuration.
	Config Config
}

// EquipmentRunning returns a scenario where DI[0]=true (equipment running signal).
// DI[0] is the primary equipment run/stop indicator from the PLC.
func EquipmentRunning() Scenario {
	di := [ioAdapter.NumDIPoints]bool{}
	di[0] = true // Running signal from PLC
	return Scenario{
		Name:        "equipment_running",
		Description: "Equipment is running. DI[0] (run signal from PLC) is active.",
		DIValues:    di,
		Config:      DefaultConfig(),
	}
}

// EquipmentStopped returns a scenario where DI[0]=false (equipment stopped).
// This represents a normal stopped state confirmed by the PLC.
func EquipmentStopped() Scenario {
	di := [ioAdapter.NumDIPoints]bool{}
	// DI[0] = false (no running signal)
	return Scenario{
		Name:        "equipment_stopped",
		Description: "Equipment is stopped. DI[0] (run signal from PLC) is inactive.",
		DIValues:    di,
		Config:      DefaultConfig(),
	}
}

// RestartRequested returns a scenario where DI[1]=true (restart request signal).
// DI[1] is the restart request indicator. Equipment may be running or stopped.
func RestartRequested() Scenario {
	di := [ioAdapter.NumDIPoints]bool{}
	di[1] = true // Restart request signal
	return Scenario{
		Name:        "restart_requested",
		Description: "Restart has been requested. DI[1] (restart request from operator panel) is active.",
		DIValues:    di,
		Config:      DefaultConfig(),
	}
}

// ModbusOffline returns a scenario that simulates Modbus communication failure.
// This tests the requirement that I/O failure must not be treated as normal state.
func ModbusOffline() Scenario {
	return Scenario{
		Name:        "modbus_offline",
		Description: "Modbus I/O module is offline. Connection fails. I/O failure must NOT be treated as normal state.",
		DIValues:    [ioAdapter.NumDIPoints]bool{},
		Config: Config{
			SimulatedLatencyMs: 0,
			FailOnConnect:      true,
			FailOnRead:         true,
			FailOnWrite:        true,
			TimeoutOnRead:      false,
		},
	}
}

// OutputFeedback returns a scenario where DI[2]=true after DO write.
// DI[2] represents output feedback confirmation from the PLC/Safety Relay.
// This verifies the closed-loop pattern: DO write -> PLC executes -> DI confirms.
func OutputFeedback() Scenario {
	di := [ioAdapter.NumDIPoints]bool{}
	di[0] = true // Equipment running
	di[2] = true // Feedback confirmation: output was acknowledged by PLC/Safety Relay
	return Scenario{
		Name:        "output_feedback",
		Description: "Output feedback confirmed. DI[2] (PLC/Safety Relay acknowledge) is active after DO write.",
		DIValues:    di,
		Config:      DefaultConfig(),
	}
}

// Timeout returns a scenario that simulates Modbus timeout.
// This tests the requirement that timeout must produce degraded quality,
// which in turn produces UNKNOWN equipment state.
func Timeout() Scenario {
	return Scenario{
		Name:        "timeout",
		Description: "Modbus communication timeout. Reads will fail with timeout error.",
		DIValues:    [ioAdapter.NumDIPoints]bool{},
		Config: Config{
			SimulatedLatencyMs: 0,
			FailOnConnect:      false,
			FailOnRead:         false,
			FailOnWrite:        false,
			TimeoutOnRead:      true,
		},
	}
}

// ApplyScenario configures the simulator with the given scenario.
func ApplyScenario(sim *Simulator, scenario Scenario) {
	sim.SetConfig(scenario.Config)
	sim.SetAllDI(scenario.DIValues)
}
