package safety

// This file implements the WorkWindow management for Rule R-04.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
//
// A WorkWindow represents an approved maintenance time window.
// Only authorized personnel can create a work window.
// The system enters MAINTENANCE_MONITORING mode only when:
//   - A valid, non-expired work window exists for the zone
//   - Equipment in the zone is confirmed STOPPED
//
// [R3] Work windows do not override other safety rules.
// [R3] If zone is UNKNOWN/STALE, R-03 takes priority over R-04.

import (
	"sync"
	"time"
)

// WorkWindowStatus represents the lifecycle state of a work window.
type WorkWindowStatus string

const (
	WorkWindowStatusActive  WorkWindowStatus = "ACTIVE"
	WorkWindowStatusClosed  WorkWindowStatus = "CLOSED"
	WorkWindowStatusExpired WorkWindowStatus = "EXPIRED"
)

// WorkWindow represents an approved maintenance time window.
// [R3] Only authorized personnel can create work windows.
type WorkWindow struct {
	// ID uniquely identifies this work window.
	ID string `json:"id"`

	// ZoneID identifies the zone this work window applies to.
	ZoneID string `json:"zoneId"`

	// RequestedBy identifies who requested/approved the window.
	RequestedBy string `json:"requestedBy"`

	// StartedAt is when the window was activated.
	StartedAt time.Time `json:"startedAt"`

	// ExpiresAt is when the window expires (must be explicitly closed or expires).
	ExpiresAt time.Time `json:"expiresAt"`

	// Status is the current lifecycle state.
	Status WorkWindowStatus `json:"status"`
}

// IsActive returns true if the work window is currently active and not expired.
func (w *WorkWindow) IsActive(now time.Time) bool {
	if w.Status != WorkWindowStatusActive {
		return false
	}
	return now.Before(w.ExpiresAt)
}

// WorkWindowManager manages active maintenance windows.
// [R3] Thread-safe via internal mutex.
type WorkWindowManager struct {
	mu      sync.Mutex
	windows map[string]*WorkWindow // keyed by window ID
}

// NewWorkWindowManager creates a new WorkWindowManager.
func NewWorkWindowManager() *WorkWindowManager {
	return &WorkWindowManager{
		windows: make(map[string]*WorkWindow),
	}
}

// Start creates and activates a new work window.
// [R3] The requestedBy field must identify the authorized person.
func (m *WorkWindowManager) Start(id, zoneID, requestedBy string, duration time.Duration) *WorkWindow {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	w := &WorkWindow{
		ID:          id,
		ZoneID:      zoneID,
		RequestedBy: requestedBy,
		StartedAt:   now,
		ExpiresAt:   now.Add(duration),
		Status:      WorkWindowStatusActive,
	}
	m.windows[id] = w
	return w
}

// Close explicitly closes a work window before expiration.
func (m *WorkWindowManager) Close(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	w, ok := m.windows[id]
	if !ok {
		return false
	}
	w.Status = WorkWindowStatusClosed
	return true
}

// IsActive returns true if any active work window exists for the given zone.
// Expired windows are automatically marked as expired.
func (m *WorkWindowManager) IsActive(zoneID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, w := range m.windows {
		if w.ZoneID != zoneID {
			continue
		}
		if w.Status == WorkWindowStatusActive {
			if now.Before(w.ExpiresAt) {
				return true
			}
			// Auto-expire
			w.Status = WorkWindowStatusExpired
		}
	}
	return false
}

// GetActive returns all active work windows for the given zone.
// Expired windows are automatically marked as expired.
func (m *WorkWindowManager) GetActive(zoneID string) []*WorkWindow {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var active []*WorkWindow
	for _, w := range m.windows {
		if w.ZoneID != zoneID {
			continue
		}
		if w.Status == WorkWindowStatusActive {
			if now.Before(w.ExpiresAt) {
				active = append(active, w)
			} else {
				w.Status = WorkWindowStatusExpired
			}
		}
	}
	return active
}

// GetActiveZones returns the set of zone IDs that have active work windows.
// This is used to populate EvaluationContext.ActiveWorkWindows.
func (m *WorkWindowManager) GetActiveZones() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	zones := make(map[string]bool)
	for _, w := range m.windows {
		if w.Status == WorkWindowStatusActive {
			if now.Before(w.ExpiresAt) {
				zones[w.ZoneID] = true
			} else {
				w.Status = WorkWindowStatusExpired
			}
		}
	}
	return zones
}
