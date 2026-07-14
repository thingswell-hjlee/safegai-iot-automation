// Package simulator provides a simulated I/O adapter for testing.
// It emulates an isolated 8DI/8DO Modbus I/O module with configurable
// scenarios, enabling end-to-end testing without real hardware.
//
// This simulator replaces real Modbus hardware in development/test environments.
// No actual PLC, Safety Relay, or equipment connection is made.
package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
	domainErrors "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
)

// Config holds the simulator configuration.
type Config struct {
	// SimulatedLatencyMs is the simulated response latency in milliseconds.
	SimulatedLatencyMs int64

	// FailOnConnect if true, Connect() will return a connection error.
	FailOnConnect bool

	// FailOnRead if true, ReadDI/ReadAllDI will return I/O failure errors.
	FailOnRead bool

	// FailOnWrite if true, WriteDO will return I/O failure errors.
	FailOnWrite bool

	// TimeoutOnRead if true, ReadDI/ReadAllDI will return timeout errors.
	TimeoutOnRead bool
}

// DefaultConfig returns a default simulator configuration (all healthy).
func DefaultConfig() Config {
	return Config{
		SimulatedLatencyMs: 1,
		FailOnConnect:      false,
		FailOnRead:         false,
		FailOnWrite:        false,
		TimeoutOnRead:      false,
	}
}

// Simulator implements the IOAdapter interface for testing purposes.
// It maintains simulated DI/DO state and supports scenario-based testing.
type Simulator struct {
	mu     sync.Mutex
	config Config
	ioConf ioAdapter.IOConfig
	health ioAdapter.IOHealth

	// diState holds the simulated digital input states (8 points).
	diState [ioAdapter.NumDIPoints]bool

	// doState holds the simulated digital output states (8 points).
	doState [ioAdapter.NumDOPoints]bool

	// doLog records all DO commands for verification.
	doLog []ioAdapter.DOCommand

	connected bool
	closed    bool
}

// New creates a new I/O simulator with the given configuration.
func New(cfg Config) *Simulator {
	return &Simulator{
		config: cfg,
		health: ioAdapter.IOHealth{
			Online:     false,
			LastPollAt: time.Time{},
			ErrorCount: 0,
			LatencyMs:  cfg.SimulatedLatencyMs,
		},
	}
}

// Connect establishes a simulated connection to the I/O module.
func (s *Simulator) Connect(ctx context.Context, config ioAdapter.IOConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return domainErrors.NewIOFailureError("io-simulator", "connect", fmt.Errorf("simulator already closed"))
	}

	if s.config.FailOnConnect {
		s.health.ErrorCount++
		return domainErrors.NewConnectionError(config.Host, config.Port, "simulated connection failure", nil)
	}

	s.ioConf = config
	s.connected = true
	s.health.Online = true
	s.health.LastPollAt = time.Now().UTC()
	s.health.ErrorCount = 0
	return nil
}

// ReadDI reads a single digital input at the specified address (0-7).
func (s *Simulator) ReadDI(ctx context.Context, address int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkReady(); err != nil {
		return false, err
	}

	if address < 0 || address >= ioAdapter.NumDIPoints {
		return false, domainErrors.NewValidationError("address", fmt.Sprintf("DI address %d out of range [0,%d)", address, ioAdapter.NumDIPoints))
	}

	if s.config.TimeoutOnRead {
		s.health.ErrorCount++
		return false, domainErrors.NewTimeoutError(fmt.Sprintf("ReadDI address=%d", address))
	}

	if s.config.FailOnRead {
		s.health.ErrorCount++
		return false, domainErrors.NewIOFailureError("io-simulator", fmt.Sprintf("ReadDI address=%d", address), fmt.Errorf("simulated read failure"))
	}

	s.health.LastPollAt = time.Now().UTC()
	s.health.LatencyMs = s.config.SimulatedLatencyMs
	return s.diState[address], nil
}

// ReadAllDI reads all 8 digital input points.
func (s *Simulator) ReadAllDI(ctx context.Context) ([]ioAdapter.DIState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkReady(); err != nil {
		return nil, err
	}

	if s.config.TimeoutOnRead {
		s.health.ErrorCount++
		return nil, domainErrors.NewTimeoutError("ReadAllDI")
	}

	if s.config.FailOnRead {
		s.health.ErrorCount++
		return nil, domainErrors.NewIOFailureError("io-simulator", "ReadAllDI", fmt.Errorf("simulated read failure"))
	}

	now := time.Now().UTC()
	states := make([]ioAdapter.DIState, ioAdapter.NumDIPoints)
	for i := 0; i < ioAdapter.NumDIPoints; i++ {
		states[i] = ioAdapter.DIState{
			Address:    i,
			Value:      s.diState[i],
			Quality:    ioAdapter.DIQualityGood,
			LastUpdate: now,
		}
	}

	s.health.LastPollAt = now
	s.health.LatencyMs = s.config.SimulatedLatencyMs
	return states, nil
}

// WriteDO writes a command to a single digital output.
// In a real system, DO is routed to PLC or Safety Relay only.
// This simulator records the command for test verification.
func (s *Simulator) WriteDO(ctx context.Context, address int, cmd ioAdapter.DOCommand) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkReady(); err != nil {
		return err
	}

	if address < 0 || address >= ioAdapter.NumDOPoints {
		return domainErrors.NewValidationError("address", fmt.Sprintf("DO address %d out of range [0,%d)", address, ioAdapter.NumDOPoints))
	}

	if s.config.FailOnWrite {
		s.health.ErrorCount++
		return domainErrors.NewIOFailureError("io-simulator", fmt.Sprintf("WriteDO address=%d", address), fmt.Errorf("simulated write failure"))
	}

	s.doState[address] = cmd.Value
	s.doLog = append(s.doLog, cmd)
	return nil
}

// Health returns the current health status of the simulated I/O module.
func (s *Simulator) Health() ioAdapter.IOHealth {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.health
}

// Close terminates the simulated connection.
func (s *Simulator) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.connected = false
	s.health.Online = false
	return nil
}

// --- Test helpers (not part of the IOAdapter interface) ---

// SetDI sets a specific digital input value for testing.
func (s *Simulator) SetDI(address int, value bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if address >= 0 && address < ioAdapter.NumDIPoints {
		s.diState[address] = value
	}
}

// SetAllDI sets all digital input values at once for testing.
func (s *Simulator) SetAllDI(values [ioAdapter.NumDIPoints]bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diState = values
}

// GetDO returns the current state of a digital output for test verification.
func (s *Simulator) GetDO(address int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if address >= 0 && address < ioAdapter.NumDOPoints {
		return s.doState[address]
	}
	return false
}

// GetDOLog returns all recorded DO commands for test verification.
func (s *Simulator) GetDOLog() []ioAdapter.DOCommand {
	s.mu.Lock()
	defer s.mu.Unlock()
	log := make([]ioAdapter.DOCommand, len(s.doLog))
	copy(log, s.doLog)
	return log
}

// SetConfig updates the simulator configuration (for injecting faults mid-test).
func (s *Simulator) SetConfig(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
}

// checkReady verifies the simulator is connected and not closed.
// Must be called with the mutex held.
func (s *Simulator) checkReady() error {
	if s.closed {
		return domainErrors.NewIOFailureError("io-simulator", "check", fmt.Errorf("simulator closed"))
	}
	if !s.connected {
		return domainErrors.NewIOFailureError("io-simulator", "check", fmt.Errorf("not connected"))
	}
	return nil
}

// Compile-time interface compliance check.
var _ ioAdapter.IOAdapter = (*Simulator)(nil)
