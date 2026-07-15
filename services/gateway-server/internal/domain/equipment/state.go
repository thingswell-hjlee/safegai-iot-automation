// Package equipment manages equipment running state derived from
// digital input signals. Stale or failed I/O input always produces
// UNKNOWN equipment state.
//
// SAFETY CONSTRAINTS:
//   - Stale DI input MUST produce UNKNOWN equipment state.
//   - I/O failure MUST NOT be treated as a normal state.
//   - No direct machine power control logic exists in this package.
//   - All output commands reference PLC/Safety Relay only.
package equipment

import (
	"sync"
	"time"

	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// EquipmentConfig holds configuration for a single equipment.
type EquipmentConfig struct {
	// ID is the unique identifier for the equipment.
	ID string

	// RunningDIAddress is the DI address that indicates running state (default: 0).
	RunningDIAddress int

	// RestartDIAddress is the DI address for restart request (default: 1).
	RestartDIAddress int

	// StaleDuration is the maximum time since last DI update before
	// the state is considered stale and transitions to UNKNOWN.
	StaleDuration time.Duration
}

// DefaultEquipmentConfig returns a default equipment configuration.
func DefaultEquipmentConfig(id string) EquipmentConfig {
	return EquipmentConfig{
		ID:               id,
		RunningDIAddress: 0,
		RestartDIAddress: 1,
		StaleDuration:    5 * time.Second,
	}
}

// EquipmentState represents the current state of a piece of equipment.
type EquipmentState struct {
	mu sync.RWMutex

	// ID is the unique equipment identifier.
	ID string

	// State is the current equipment state derived from DI inputs.
	State events.EquipmentState

	// RestartRequested indicates an operator has requested a restart.
	// This is NOT an equipment state per canonical contract; it is a
	// separate operator request tracked as an audit event.
	RestartRequested bool

	// Quality indicates the reliability of the current state.
	Quality ioAdapter.DIQuality

	// LastUpdate is the timestamp of the last successful DI update.
	LastUpdate time.Time

	// StaleDuration is the configured staleness threshold.
	StaleDuration time.Duration

	// config holds the equipment configuration.
	config EquipmentConfig
}

// NewEquipmentState creates a new equipment state tracker.
// Initial state is UNKNOWN until a valid DI update is received.
func NewEquipmentState(cfg EquipmentConfig) *EquipmentState {
	return &EquipmentState{
		ID:            cfg.ID,
		State:         events.EquipmentUnknown,
		Quality:       ioAdapter.DIQualityBad,
		LastUpdate:    time.Time{},
		StaleDuration: cfg.StaleDuration,
		config:        cfg,
	}
}

// Update evaluates the provided DI states and determines the equipment state.
// If the relevant DI values have good quality, the state is derived from them.
// If quality is not good, the state transitions to UNKNOWN.
func (es *EquipmentState) Update(diStates []ioAdapter.DIState) {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Validate we have enough DI points
	if len(diStates) == 0 {
		es.State = events.EquipmentUnknown
		es.Quality = ioAdapter.DIQualityBad
		return
	}

	// Check if the relevant DI addresses are available
	runAddr := es.config.RunningDIAddress
	restartAddr := es.config.RestartDIAddress

	if runAddr >= len(diStates) {
		es.State = events.EquipmentUnknown
		es.Quality = ioAdapter.DIQualityBad
		return
	}

	runDI := diStates[runAddr]

	// Check quality of the running DI
	if runDI.Quality != ioAdapter.DIQualityGood {
		// Bad or stale quality: I/O failure must not be treated as normal
		es.State = events.EquipmentUnknown
		es.Quality = runDI.Quality
		return
	}

	// Check restart DI if available
	var restartActive bool
	if restartAddr < len(diStates) {
		restartDI := diStates[restartAddr]
		if restartDI.Quality == ioAdapter.DIQualityGood {
			restartActive = restartDI.Value
		}
	}

	// Derive state from DI values
	now := runDI.LastUpdate
	es.LastUpdate = now
	es.Quality = ioAdapter.DIQualityGood

	// Track restart request separately (not an equipment state)
	es.RestartRequested = restartActive

	switch {
	case runDI.Value:
		es.State = events.EquipmentRunning
	default:
		es.State = events.EquipmentStopped
	}
}

// IsStale returns true if the last update is older than the configured StaleDuration.
// Stale DI input MUST produce UNKNOWN equipment state.
func (es *EquipmentState) IsStale(now time.Time) bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// If we never received an update, it is stale
	if es.LastUpdate.IsZero() {
		return true
	}

	return now.Sub(es.LastUpdate) > es.StaleDuration
}

// GetState returns the current equipment state.
// If the state is stale (determined by the caller), this should return UNKNOWN.
func (es *EquipmentState) GetState() events.EquipmentState {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.State
}

// GetQuality returns the current quality.
func (es *EquipmentState) GetQuality() ioAdapter.DIQuality {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.Quality
}

// GetLastUpdate returns the timestamp of the last update.
func (es *EquipmentState) GetLastUpdate() time.Time {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.LastUpdate
}

// MarkStale transitions the equipment state to UNKNOWN due to staleness.
// This enforces the requirement: stale DI input -> UNKNOWN equipment state.
func (es *EquipmentState) MarkStale() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.State = events.EquipmentUnknown
	es.Quality = ioAdapter.DIQualityStale
}

// Snapshot returns a read-only copy of the equipment state for reporting.
type Snapshot struct {
	ID            string
	State         events.EquipmentState
	Quality       ioAdapter.DIQuality
	LastUpdate    time.Time
	StaleDuration time.Duration
	IsStale       bool
}

// GetSnapshot returns a point-in-time snapshot of the equipment state.
func (es *EquipmentState) GetSnapshot(now time.Time) Snapshot {
	es.mu.RLock()
	defer es.mu.RUnlock()

	stale := es.LastUpdate.IsZero() || now.Sub(es.LastUpdate) > es.StaleDuration
	state := es.State
	quality := es.Quality

	if stale && state != events.EquipmentUnknown {
		state = events.EquipmentUnknown
		quality = ioAdapter.DIQualityStale
	}

	return Snapshot{
		ID:            es.ID,
		State:         state,
		Quality:       quality,
		LastUpdate:    es.LastUpdate,
		StaleDuration: es.StaleDuration,
		IsStale:       stale,
	}
}
