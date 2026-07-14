// Package io defines the adapter interface for industrial I/O devices.
// This covers isolated Modbus TCP/RTU digital I/O modules
// (8 DI / 8 DO configuration).
//
// SAFETY CONSTRAINTS:
//   - No direct machine power control. All DO outputs go to PLC or Safety Relay only.
//   - I/O failure must never be treated as a normal or safe state.
//   - USB Relay is not acceptable for production output.
//   - Only external isolated I/O modules are supported (no PC GPIO).
package io

import "context"

// IOAdapter is the primary interface for industrial I/O device integration.
// All implementations (real Modbus hardware, simulator) must satisfy this interface.
//
// The adapter communicates with an external isolated I/O module that provides
// 8 digital inputs and 8 digital outputs via Modbus TCP or RTU protocol.
// Digital outputs are routed exclusively to PLC or Safety Relay for equipment control.
type IOAdapter interface {
	// Connect establishes a connection to the I/O module.
	// Returns an error if the connection cannot be established.
	Connect(ctx context.Context, config IOConfig) error

	// ReadDI reads a single digital input at the specified address (0-7).
	// Returns the current state (true=ON, false=OFF) and any error.
	// An error indicates I/O failure and must not be treated as a normal state.
	ReadDI(ctx context.Context, address int) (bool, error)

	// ReadAllDI reads all 8 digital input points.
	// Returns the state of each DI with quality information.
	// Errors indicate communication failure; stale or failed reads
	// must result in reduced quality (not normal operation).
	ReadAllDI(ctx context.Context) ([]DIState, error)

	// WriteDO writes a command to a single digital output at the specified address (0-7).
	// DO commands are routed to PLC or Safety Relay only. This interface does NOT
	// provide direct machine power control.
	WriteDO(ctx context.Context, address int, cmd DOCommand) error

	// Health returns the current health status of the I/O adapter.
	Health() IOHealth

	// Close terminates the connection and releases resources.
	Close() error
}
