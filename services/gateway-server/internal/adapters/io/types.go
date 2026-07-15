package io

import "time"

// IOConfig holds configuration for connecting to an I/O module.
type IOConfig struct {
	// Host is the network address of the Modbus I/O module.
	Host string `json:"host"`

	// Port is the TCP port of the Modbus I/O module (default 502).
	Port int `json:"port"`

	// UnitID is the Modbus unit/slave ID.
	UnitID byte `json:"unitId"`

	// Timeout is the maximum time to wait for a response.
	Timeout time.Duration `json:"timeout"`

	// PollInterval is the interval between consecutive DI poll cycles.
	PollInterval time.Duration `json:"pollInterval"`
}

// DefaultIOConfig returns a sensible default configuration.
func DefaultIOConfig() IOConfig {
	return IOConfig{
		Host:         "127.0.0.1",
		Port:         502,
		UnitID:       1,
		Timeout:      2 * time.Second,
		PollInterval: 100 * time.Millisecond,
	}
}

// DIQuality indicates the quality/reliability of a digital input reading.
type DIQuality string

const (
	// DIQualityGood indicates a fresh, valid reading.
	DIQualityGood DIQuality = "GOOD"

	// DIQualityStale indicates the reading has not been refreshed within the expected interval.
	// Stale DI input must produce UNKNOWN equipment state.
	DIQualityStale DIQuality = "STALE"

	// DIQualityBad indicates a communication error; the value is unreliable.
	// Bad quality must never be treated as normal operation.
	DIQualityBad DIQuality = "BAD"
)

// DIState represents the state of a single digital input point.
type DIState struct {
	// Address is the DI point address (0-7).
	Address int `json:"address"`

	// Value is the current reading (true=ON, false=OFF).
	Value bool `json:"value"`

	// Quality indicates the reliability of this reading.
	Quality DIQuality `json:"quality"`

	// LastUpdate is the timestamp of the last successful read for this point.
	LastUpdate time.Time `json:"lastUpdate"`
}

// DOCommand represents a command to write to a single digital output.
// All DO commands are routed to PLC or Safety Relay only.
// This does NOT provide direct machine power control.
type DOCommand struct {
	// Address is the DO point address (0-7).
	Address int `json:"address"`

	// Value is the desired output state (true=ON, false=OFF).
	Value bool `json:"value"`

	// PulseDuration if non-zero, the output will be pulsed for this duration
	// then automatically returned to the previous state.
	PulseDuration time.Duration `json:"pulseDuration"`

	// CommandID is a unique identifier for this command (for tracing).
	CommandID string `json:"commandId"`

	// CorrelationID links this command to a higher-level request (e.g., safety decision).
	CorrelationID string `json:"correlationId"`
}

// IOHealth represents the health status of an I/O adapter.
type IOHealth struct {
	// Online indicates whether the I/O module is currently connected and responsive.
	Online bool `json:"online"`

	// LastPollAt is the timestamp of the last successful poll cycle.
	LastPollAt time.Time `json:"lastPollAt"`

	// ErrorCount is the number of consecutive errors since last success.
	ErrorCount int `json:"errorCount"`

	// LatencyMs is the last measured round-trip latency in milliseconds.
	LatencyMs int64 `json:"latencyMs"`
}

// ModbusException represents a Modbus protocol exception response.
type ModbusException struct {
	// FunctionCode is the Modbus function that caused the exception.
	FunctionCode byte `json:"functionCode"`

	// ExceptionCode is the Modbus exception code returned by the device.
	ExceptionCode byte `json:"exceptionCode"`
}

// String returns a human-readable description of the exception.
func (e ModbusException) String() string {
	names := map[byte]string{
		0x01: "Illegal Function",
		0x02: "Illegal Data Address",
		0x03: "Illegal Data Value",
		0x04: "Server Device Failure",
		0x05: "Acknowledge",
		0x06: "Server Device Busy",
	}
	name, ok := names[e.ExceptionCode]
	if !ok {
		name = "Unknown Exception"
	}
	return name
}

// NumDIPoints is the number of digital input points on the I/O module.
const NumDIPoints = 8

// NumDOPoints is the number of digital output points on the I/O module.
const NumDOPoints = 8
