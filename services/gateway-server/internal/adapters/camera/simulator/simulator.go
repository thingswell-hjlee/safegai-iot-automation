// Package simulator provides a simulated camera adapter for testing.
// It generates deterministic camera events based on configurable scenarios,
// enabling end-to-end testing without real camera hardware.
package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
)

// Config holds simulator configuration.
type Config struct {
	// CameraID is the identifier for the simulated camera.
	CameraID string

	// Zones lists the zones this simulated camera monitors.
	Zones []string

	// MaxPersons is the maximum number of persons the simulator can report.
	MaxPersons int

	// EventInterval is the time between generated events.
	EventInterval time.Duration

	// Scenarios holds scenario functions to execute during subscription.
	Scenarios []ScenarioFunc
}

// DefaultConfig returns a default simulator configuration.
func DefaultConfig() Config {
	return Config{
		CameraID:      "cam-sim-001",
		Zones:         []string{"zone-A"},
		MaxPersons:    10,
		EventInterval: 500 * time.Millisecond,
		Scenarios:     nil,
	}
}

// Simulator implements the CameraAdapter interface for testing purposes.
// It generates events from configured scenarios without requiring real hardware.
type Simulator struct {
	mu     sync.Mutex
	config Config
	health camera.CameraHealth
	closed bool
	cancel context.CancelFunc
}

// New creates a new camera simulator with the given configuration.
func New(cfg Config) *Simulator {
	return &Simulator{
		config: cfg,
		health: camera.CameraHealth{
			Online:      false,
			LastEventAt: time.Time{},
			ErrorCount:  0,
			LatencyMs:   0,
		},
	}
}

// Connect establishes a simulated connection to the camera.
func (s *Simulator) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("simulator: already closed")
	}

	s.health.Online = true
	s.health.LatencyMs = 1 // Simulated 1ms latency
	return nil
}

// Health returns the current health status of the simulated camera.
func (s *Simulator) Health() camera.CameraHealth {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.health
}

// SubscribeEvents starts streaming simulated camera events to the provided channel.
// Events are generated from the configured scenarios. If no scenarios are configured,
// a default occupied scenario is used.
func (s *Simulator) SubscribeEvents(ctx context.Context, ch chan<- camera.RawCameraEvent) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("simulator: already closed")
	}
	if !s.health.Online {
		s.mu.Unlock()
		return fmt.Errorf("simulator: not connected")
	}

	scenarios := s.config.Scenarios
	if len(scenarios) == 0 {
		scenarios = []ScenarioFunc{Occupied(s.config.Zones[0])}
	}

	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	go s.runScenarios(subCtx, ch, scenarios)
	return nil
}

// runScenarios executes scenario functions and sends events to the channel.
func (s *Simulator) runScenarios(ctx context.Context, ch chan<- camera.RawCameraEvent, scenarios []ScenarioFunc) {
	baseTime := time.Now().UTC()

	for _, scenario := range scenarios {
		events := scenario(baseTime)
		for _, evt := range events {
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case ch <- evt:
				s.mu.Lock()
				s.health.LastEventAt = time.Now().UTC()
				s.mu.Unlock()
			case <-ctx.Done():
				return
			}

			// Small delay between events for realism
			select {
			case <-time.After(s.config.EventInterval):
			case <-ctx.Done():
				return
			}
		}
		// Advance base time between scenarios
		baseTime = baseTime.Add(10 * time.Second)
	}
}

// GetSnapshot returns a simulated snapshot (a small JPEG-like placeholder).
func (s *Simulator) GetSnapshot(ctx context.Context, zoneID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("simulator: already closed")
	}
	if !s.health.Online {
		return nil, fmt.Errorf("simulator: not connected")
	}

	// Verify zone is valid
	found := false
	for _, z := range s.config.Zones {
		if z == zoneID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("simulator: unknown zone %q", zoneID)
	}

	// Return a minimal placeholder (not a real JPEG, but satisfies the interface)
	placeholder := []byte(fmt.Sprintf("SIMULATED_SNAPSHOT:zone=%s:time=%d", zoneID, time.Now().UnixMilli()))
	return placeholder, nil
}

// GetCapabilities returns the capabilities of the simulated camera.
func (s *Simulator) GetCapabilities() camera.Capabilities {
	s.mu.Lock()
	defer s.mu.Unlock()

	return camera.Capabilities{
		Zones:            s.config.Zones,
		MaxPersons:       s.config.MaxPersons,
		SupportsSnapshot: true,
		StreamURL:        "",
	}
}

// Close terminates the simulated connection and releases resources.
func (s *Simulator) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.health.Online = false
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// Compile-time interface compliance check.
var _ camera.CameraAdapter = (*Simulator)(nil)
