package equipment

import (
	"sync"
	"time"

	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
	domainErrors "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
)

// Manager manages multiple equipment states and coordinates DI updates
// across all registered equipment.
//
// SAFETY CONSTRAINTS:
//   - No direct machine power control.
//   - All outputs reference PLC/Safety Relay only.
//   - Stale DI input produces UNKNOWN state.
//   - I/O failure is never treated as normal.
type Manager struct {
	mu        sync.RWMutex
	equipment map[string]*EquipmentState
}

// NewManager creates a new equipment state manager.
func NewManager() *Manager {
	return &Manager{
		equipment: make(map[string]*EquipmentState),
	}
}

// RegisterEquipment registers a new equipment with the given configuration.
// Returns an error if equipment with the same ID is already registered.
func (m *Manager) RegisterEquipment(id string, config EquipmentConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "" {
		return domainErrors.NewValidationError("id", "equipment ID must not be empty")
	}

	if _, exists := m.equipment[id]; exists {
		return domainErrors.NewConflictError("equipment", "already registered: "+id)
	}

	config.ID = id
	m.equipment[id] = NewEquipmentState(config)
	return nil
}

// UpdateFromDI updates all registered equipment from the provided DI states.
// Each equipment reads the relevant DI addresses from the provided states.
func (m *Manager) UpdateFromDI(diStates []ioAdapter.DIState) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, eq := range m.equipment {
		eq.Update(diStates)
	}
}

// GetState returns the snapshot of a specific equipment.
// Returns an error if the equipment is not registered.
func (m *Manager) GetState(id string, now time.Time) (Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	eq, exists := m.equipment[id]
	if !exists {
		return Snapshot{}, domainErrors.NewNotFoundError("equipment", id)
	}

	return eq.GetSnapshot(now), nil
}

// GetAllStates returns snapshots of all registered equipment at the given time.
func (m *Manager) GetAllStates(now time.Time) []Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := make([]Snapshot, 0, len(m.equipment))
	for _, eq := range m.equipment {
		snapshots = append(snapshots, eq.GetSnapshot(now))
	}
	return snapshots
}

// CheckStaleness checks all equipment for staleness and marks them UNKNOWN.
// This enforces the requirement: stale DI input -> UNKNOWN equipment state.
// Returns the list of equipment IDs that transitioned to stale/UNKNOWN.
func (m *Manager) CheckStaleness(now time.Time) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var staleIDs []string
	for id, eq := range m.equipment {
		if eq.IsStale(now) {
			eq.MarkStale()
			staleIDs = append(staleIDs, id)
		}
	}
	return staleIDs
}

// Count returns the number of registered equipment.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.equipment)
}

// Unregister removes an equipment from the manager.
// Returns an error if the equipment is not registered.
func (m *Manager) Unregister(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.equipment[id]; !exists {
		return domainErrors.NewNotFoundError("equipment", id)
	}

	delete(m.equipment, id)
	return nil
}
