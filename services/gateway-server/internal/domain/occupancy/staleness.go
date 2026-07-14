package occupancy

import (
	"sync"
	"time"
)

// StaleChecker monitors multiple zone state machines for staleness.
//
// R3 SAFETY CRITICAL: Zones that have not received data within the
// configured StaleTimeout MUST transition to STALE state.
// STALE state does NOT satisfy vacancy and blocks equipment restart.
//
// The StaleChecker is designed to be called periodically (e.g. every second)
// to detect and apply staleness transitions across all managed zones.
type StaleChecker struct {
	mu       sync.Mutex
	machines map[string]*StateMachine
	config   Config
}

// NewStaleChecker creates a new StaleChecker with the given configuration.
func NewStaleChecker(cfg Config) *StaleChecker {
	return &StaleChecker{
		machines: make(map[string]*StateMachine),
		config:   cfg,
	}
}

// Register adds a state machine to the staleness checker.
// Thread-safe: acquires internal lock.
func (sc *StaleChecker) Register(sm *StateMachine) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	state := sm.GetState()
	sc.machines[state.ZoneID] = sm
}

// Unregister removes a state machine from the staleness checker.
// Thread-safe: acquires internal lock.
func (sc *StaleChecker) Unregister(zoneID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.machines, zoneID)
}

// Check evaluates all registered zones for staleness at the given time.
// Returns a slice of transitions for zones that became stale.
//
// R3 SAFETY CRITICAL: This method enforces the staleness timeout.
// Any zone without data within StaleTimeout MUST transition to STALE.
// STALE does NOT satisfy vacancy.
//
// Thread-safe: acquires internal lock.
func (sc *StaleChecker) Check(now time.Time) []StateTransition {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var transitions []StateTransition
	for _, sm := range sc.machines {
		if t, stale := sm.CheckStaleness(now); stale {
			transitions = append(transitions, t)
		}
	}
	return transitions
}

// CheckZones evaluates the given zone states for staleness against the configured timeout.
// This is a stateless utility method that does not require registered machines.
//
// R3 SAFETY CRITICAL: Zones exceeding the stale timeout are identified for transition.
// STALE state does NOT satisfy vacancy and blocks restart.
func (sc *StaleChecker) CheckZones(zones []ZoneState, now time.Time) []StateTransition {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var transitions []StateTransition
	for i := range zones {
		zone := &zones[i]
		// Skip already stale zones.
		if zone.CurrentState == StateStale {
			continue
		}

		elapsed := now.Sub(zone.LastEventAt)
		if elapsed >= sc.config.StaleTimeout {
			transitions = append(transitions, StateTransition{
				ZoneID:    zone.ZoneID,
				From:      zone.CurrentState,
				To:        StateStale,
				Reason:    "staleness timeout exceeded",
				Timestamp: now,
			})
		}
	}
	return transitions
}

// MachineCount returns the number of registered state machines.
func (sc *StaleChecker) MachineCount() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return len(sc.machines)
}
