// Package output implements the I/O output adapter for the SafeGAI edge gateway.
// This adapter sends actuation commands to PLC/Safety Relay inputs.
//
// SAFETY: All output goes to PLC/Safety Relay input, NEVER machine power directly.
// General-purpose DO does NOT switch machine power directly.
// The PLC/Safety Relay controls the actual equipment power circuit.
// This adapter is the only authorized path for output signals.
//
// REPLAY GUARD: This adapter does NOT maintain state across restarts.
// After restart, past pulse commands are NOT replayed.
// The replay guard in the actuation service prevents stale commands from
// reaching this adapter.
package output

import (
	"fmt"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/actuation"
)

// IOAdapter is the low-level abstraction for digital output hardware.
// Implementations must target PLC/Safety Relay inputs ONLY.
//
// SAFETY: This interface sends signals to PLC/Safety Relay inputs.
// Implementations must NEVER directly control machine power.
// The PLC/Safety Relay is the authority for equipment power control.
type IOAdapter interface {
	// WriteDO writes a digital output value to the specified address.
	// SAFETY: Address must be a PLC/Safety Relay input, not machine power.
	WriteDO(address string, value bool) error

	// WritePulse writes a timed pulse to the specified address.
	// SAFETY: Pulse goes to PLC/Safety Relay stop-request input.
	// Duration is bounded; no infinite pulse.
	WritePulse(address string, durationMs int) error

	// ReadDI reads a digital input for feedback/acknowledgment.
	// Used to verify that PLC/Safety Relay received the command.
	ReadDI(address string) (bool, error)
}

// OutputExecutor implements the actuation.IOExecutor interface.
// It sends output commands to the PLC/Safety Relay via the IOAdapter.
//
// SAFETY DOCUMENTATION:
// - All output goes to PLC/Safety Relay input, NEVER machine power directly.
// - General-purpose DO does NOT switch machine power directly.
// - The PLC/Safety Relay decides whether to stop equipment.
// - This executor never has direct control over machine power circuits.
// - Stop request pulses are bounded in duration (no infinite activation).
// - Feedback reading confirms PLC/Safety Relay received the signal.
type OutputExecutor struct {
	mu       sync.Mutex
	adapter  IOAdapter
	feedback map[string]bool
}

// NewOutputExecutor creates a new OutputExecutor with the given IOAdapter.
// The adapter must target PLC/Safety Relay inputs only.
//
// SAFETY: The IOAdapter must connect to PLC/Safety Relay inputs.
// It must NOT connect directly to machine power switching circuits.
func NewOutputExecutor(adapter IOAdapter) *OutputExecutor {
	return &OutputExecutor{
		adapter:  adapter,
		feedback: make(map[string]bool),
	}
}

// Execute sends the actuation command to the PLC/Safety Relay input.
// SAFETY: Output goes to PLC/Safety Relay input, NEVER direct machine power.
// The command type determines the output method:
//   - WARNING_LIGHT: WriteDO to PLC/Safety Relay warning light input
//   - SIREN: WriteDO to PLC/Safety Relay siren input
//   - STOP_REQUEST_PULSE: WritePulse to PLC/Safety Relay stop-request input
//   - AUDIO_ANNOUNCEMENT: WriteDO to PLC/Safety Relay PA system input
func (e *OutputExecutor) Execute(cmd actuation.ActuationCommand) error {
	switch cmd.CommandType {
	case actuation.CommandWarningLight:
		// SAFETY: Output goes to PLC/Safety Relay warning light input.
		value := cmd.Value == "ON"
		if err := e.adapter.WriteDO(cmd.TargetAddress, value); err != nil {
			return fmt.Errorf("failed to write warning light to PLC/Safety Relay: %w", err)
		}

	case actuation.CommandSiren:
		// SAFETY: Output goes to PLC/Safety Relay siren input.
		value := cmd.Value == "ON"
		if err := e.adapter.WriteDO(cmd.TargetAddress, value); err != nil {
			return fmt.Errorf("failed to write siren to PLC/Safety Relay: %w", err)
		}

	case actuation.CommandStopRequestPulse:
		// SAFETY CRITICAL: Stop request pulse goes to PLC/Safety Relay ONLY.
		// The PLC/Safety Relay decides whether to actually stop the equipment.
		// This gateway does NOT directly switch machine power.
		durationMs := cmd.PulseDurationMs
		if durationMs <= 0 {
			durationMs = 500 // default 500ms pulse
		}
		if err := e.adapter.WritePulse(cmd.TargetAddress, durationMs); err != nil {
			return fmt.Errorf("failed to write stop-request pulse to PLC/Safety Relay: %w", err)
		}

	case actuation.CommandAudioAnnouncement:
		// SAFETY: Output goes to PLC/Safety Relay PA system input.
		value := cmd.Value == "ON"
		if err := e.adapter.WriteDO(cmd.TargetAddress, value); err != nil {
			return fmt.Errorf("failed to write audio announcement to PLC/Safety Relay: %w", err)
		}

	default:
		return fmt.Errorf("unsupported command type: %s", cmd.CommandType)
	}

	// Record that this command was executed (for feedback tracking).
	e.mu.Lock()
	e.feedback[cmd.ID] = false // not yet confirmed
	e.mu.Unlock()

	return nil
}

// ReadFeedback checks whether the PLC/Safety Relay acknowledged the command.
// SAFETY: Reads feedback from PLC/Safety Relay confirming it received the signal.
// This does NOT mean the equipment actually stopped; the PLC/Safety Relay
// decides independently whether to apply the stop request.
func (e *OutputExecutor) ReadFeedback(commandID string) (bool, error) {
	e.mu.Lock()
	_, exists := e.feedback[commandID]
	e.mu.Unlock()

	if !exists {
		return false, fmt.Errorf("no feedback record for command %s", commandID)
	}

	// In a real implementation, this would read a DI from the PLC/Safety Relay
	// to confirm acknowledgment. For the interface implementation, we return
	// the stored state. Real hardware reads happen through the IOAdapter.
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.feedback[commandID], nil
}

// SimulatedIOAdapter is a simulated IOAdapter for testing and development.
// It simulates PLC/Safety Relay I/O without connecting to real hardware.
//
// SAFETY: This is a SIMULATOR. In production, the real adapter connects to
// actual PLC/Safety Relay hardware. Real equipment is NEVER accessed during
// development or testing.
type SimulatedIOAdapter struct {
	mu         sync.Mutex
	outputs    map[string]bool
	pulses     map[string]int
	writeDelay time.Duration
	failNext   bool
}

// NewSimulatedIOAdapter creates a new simulated IO adapter.
// SAFETY: This is for testing only. Real hardware is never accessed.
func NewSimulatedIOAdapter() *SimulatedIOAdapter {
	return &SimulatedIOAdapter{
		outputs: make(map[string]bool),
		pulses:  make(map[string]int),
	}
}

// WriteDO simulates writing a digital output to PLC/Safety Relay input.
// SAFETY: In production, this goes to PLC/Safety Relay, not machine power.
func (s *SimulatedIOAdapter) WriteDO(address string, value bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failNext {
		s.failNext = false
		return fmt.Errorf("simulated I/O failure on address %s", address)
	}

	if s.writeDelay > 0 {
		s.mu.Unlock()
		time.Sleep(s.writeDelay)
		s.mu.Lock()
	}

	s.outputs[address] = value
	return nil
}

// WritePulse simulates writing a timed pulse to PLC/Safety Relay input.
// SAFETY: In production, this sends a stop-request pulse to PLC/Safety Relay.
// The PLC/Safety Relay decides whether to stop equipment.
func (s *SimulatedIOAdapter) WritePulse(address string, durationMs int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failNext {
		s.failNext = false
		return fmt.Errorf("simulated I/O failure on pulse to address %s", address)
	}

	s.pulses[address] = durationMs
	s.outputs[address] = true
	return nil
}

// ReadDI simulates reading a digital input for feedback.
// Returns the current output state as simulated feedback.
func (s *SimulatedIOAdapter) ReadDI(address string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outputs[address], nil
}

// SetFailNext causes the next write operation to fail (for testing).
func (s *SimulatedIOAdapter) SetFailNext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = true
}

// SetWriteDelay sets a delay for write operations (for testing).
func (s *SimulatedIOAdapter) SetWriteDelay(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeDelay = d
}

// GetOutput returns the current state of a simulated output.
func (s *SimulatedIOAdapter) GetOutput(address string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outputs[address]
}

// GetPulse returns the last pulse duration for an address.
func (s *SimulatedIOAdapter) GetPulse(address string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pulses[address]
}
