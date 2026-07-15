package occupancy

import (
	"fmt"
	"sync"
	"time"
)

// CameraEventType identifies the type of event received from the camera system.
type CameraEventType string

const (
	// EventPersonDetected indicates one or more persons were detected in the zone.
	EventPersonDetected CameraEventType = "PERSON_DETECTED"

	// EventNoPerson indicates no person was detected in the zone.
	EventNoPerson CameraEventType = "NO_PERSON"

	// EventCameraOffline indicates the camera has gone offline.
	// R3 SAFETY CRITICAL: Camera offline MUST produce UNKNOWN state, NEVER VACANT.
	EventCameraOffline CameraEventType = "CAMERA_OFFLINE"

	// EventCameraOnline indicates the camera has come back online.
	EventCameraOnline CameraEventType = "CAMERA_ONLINE"

	// EventError indicates a communication or parse error.
	// R3 SAFETY CRITICAL: Errors MUST produce UNKNOWN state, NEVER VACANT.
	EventError CameraEventType = "ERROR"
)

// CameraEvent represents a single event from the camera/detection system.
type CameraEvent struct {
	// ZoneID identifies the zone this event pertains to.
	ZoneID string

	// Type is the category of the event.
	Type CameraEventType

	// PersonCount is the number of persons detected (relevant for PERSON_DETECTED).
	PersonCount int

	// Timestamp is when the event was generated.
	Timestamp time.Time

	// CameraID identifies the source camera.
	CameraID string

	// ErrorMessage provides detail for ERROR events.
	ErrorMessage string
}

// StateTransition records a state change in the occupancy state machine.
type StateTransition struct {
	// ZoneID identifies the zone that transitioned.
	ZoneID string

	// From is the state before the transition.
	From OccupancyState

	// To is the state after the transition.
	To OccupancyState

	// Reason describes why the transition occurred.
	Reason string

	// Timestamp is when the transition occurred.
	Timestamp time.Time
}

// StateMachine implements the zone occupancy state machine.
//
// R3 SAFETY CRITICAL: This is the core safety component.
// All state transitions are governed by strict rules:
//   - Only VACANT_CONFIRMED satisfies the vacancy condition.
//   - Camera offline always produces UNKNOWN.
//   - Errors always produce UNKNOWN.
//   - Staleness always produces STALE (never VACANT).
//   - Vacancy confirmation requires BOTH time duration AND consecutive samples.
//
// Thread safety: StateMachine is safe for concurrent use via internal mutex.
type StateMachine struct {
	mu     sync.Mutex
	state  ZoneState
	config Config
}

// NewStateMachine creates a new state machine for the given zone.
//
// R3 SAFETY CRITICAL: The initial state is ALWAYS UNKNOWN.
// A zone must receive positive detection data before transitioning
// to any other state.
func NewStateMachine(zoneID string, cfg Config) *StateMachine {
	now := time.Now()
	return &StateMachine{
		state: ZoneState{
			ZoneID:         zoneID,
			CurrentState:   StateUnknown,
			PreviousState:  StateUnknown,
			PersonCount:    0,
			LastEventAt:    now,
			StateChangedAt: now,
		},
		config: cfg,
	}
}

// GetState returns the current zone state.
// Thread-safe: acquires internal lock.
func (sm *StateMachine) GetState() ZoneState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.state
}

// ProcessEvent processes a camera event and returns any state transition.
//
// R3 SAFETY CRITICAL: This method enforces all safety invariants:
//   - CAMERA_OFFLINE -> UNKNOWN (never VACANT)
//   - ERROR -> UNKNOWN (never VACANT)
//   - NO_PERSON transitions to VACANT_PENDING, not directly to VACANT_CONFIRMED
//   - VACANT_CONFIRMED requires meeting BOTH time and sample thresholds
//
// Thread-safe: acquires internal lock.
func (sm *StateMachine) ProcessEvent(event CameraEvent) (StateTransition, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if event.ZoneID != sm.state.ZoneID {
		return StateTransition{}, fmt.Errorf("event zone %q does not match machine zone %q",
			event.ZoneID, sm.state.ZoneID)
	}

	sm.state.LastEventAt = event.Timestamp

	switch event.Type {
	case EventPersonDetected:
		return sm.handlePersonDetected(event), nil

	case EventNoPerson:
		return sm.handleNoPerson(event), nil

	case EventCameraOffline:
		return sm.handleCameraOffline(event), nil

	case EventCameraOnline:
		return sm.handleCameraOnline(event), nil

	case EventError:
		return sm.handleError(event), nil

	default:
		// Unknown event type: fail-safe to UNKNOWN.
		return sm.transitionTo(StateUnknown, event.Timestamp,
			fmt.Sprintf("unknown event type: %s", event.Type)), nil
	}
}

// CheckStaleness checks whether the zone has gone stale based on the given time.
// Returns the transition and true if the zone transitioned to STALE.
//
// R3 SAFETY CRITICAL: STALE timeout MUST NOT produce VACANT_CONFIRMED.
// STALE blocks equipment restart.
//
// Thread-safe: acquires internal lock.
func (sm *StateMachine) CheckStaleness(now time.Time) (StateTransition, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Already stale - no transition needed.
	if sm.state.CurrentState == StateStale {
		return StateTransition{}, false
	}

	elapsed := now.Sub(sm.state.LastEventAt)
	if elapsed >= sm.config.StaleTimeout {
		transition := sm.transitionTo(StateStale, now,
			fmt.Sprintf("no event received for %v (timeout=%v)",
				elapsed.Truncate(time.Millisecond), sm.config.StaleTimeout))
		return transition, true
	}

	return StateTransition{}, false
}

// handlePersonDetected processes a PERSON_DETECTED event.
// Valid from ANY state: always transitions to OCCUPIED.
func (sm *StateMachine) handlePersonDetected(event CameraEvent) StateTransition {
	sm.state.PersonCount = event.PersonCount
	if sm.state.PersonCount < 1 {
		sm.state.PersonCount = 1
	}

	if sm.state.CurrentState == StateOccupied {
		// Already occupied - update count, no state change.
		return StateTransition{
			ZoneID:    sm.state.ZoneID,
			From:      StateOccupied,
			To:        StateOccupied,
			Reason:    "person count updated",
			Timestamp: event.Timestamp,
		}
	}

	// Reset pending vacancy tracking.
	sm.state.PendingSince = time.Time{}
	sm.state.ConsecutiveVacantSamples = 0

	return sm.transitionTo(StateOccupied, event.Timestamp,
		fmt.Sprintf("person detected (count=%d)", sm.state.PersonCount))
}

// handleNoPerson processes a NO_PERSON event.
//
// R3 SAFETY CRITICAL: NO_PERSON never directly produces VACANT_CONFIRMED.
// It transitions to VACANT_PENDING, and only if BOTH:
//   - VacancyConfirmDuration has elapsed since PendingSince, AND
//   - VacancyConfirmSamples consecutive no-person samples received
//
// ...does it transition to VACANT_CONFIRMED.
func (sm *StateMachine) handleNoPerson(event CameraEvent) StateTransition {
	sm.state.PersonCount = 0

	switch sm.state.CurrentState {
	case StateUnknown, StateOccupied, StateStale:
		// Start vacancy pending period.
		sm.state.PendingSince = event.Timestamp
		sm.state.ConsecutiveVacantSamples = 1
		return sm.transitionTo(StateVacantPending, event.Timestamp,
			"no person detected, starting vacancy confirmation")

	case StateVacantPending:
		// Accumulate consecutive vacant sample.
		sm.state.ConsecutiveVacantSamples++
		return sm.checkVacancyConfirmation(event.Timestamp)

	case StateVacantConfirmed:
		// Already confirmed - no transition needed.
		return StateTransition{
			ZoneID:    sm.state.ZoneID,
			From:      StateVacantConfirmed,
			To:        StateVacantConfirmed,
			Reason:    "vacancy maintained",
			Timestamp: event.Timestamp,
		}

	default:
		// Defensive: unknown state, go to VACANT_PENDING.
		sm.state.PendingSince = event.Timestamp
		sm.state.ConsecutiveVacantSamples = 1
		return sm.transitionTo(StateVacantPending, event.Timestamp,
			"no person detected from unknown internal state")
	}
}

// checkVacancyConfirmation evaluates whether vacancy confirmation criteria are met.
//
// R3 SAFETY CRITICAL: BOTH conditions must be satisfied simultaneously:
//  1. Time elapsed since PendingSince >= VacancyConfirmDuration
//  2. ConsecutiveVacantSamples >= VacancyConfirmSamples
//
// If either condition is not met, the zone remains in VACANT_PENDING.
func (sm *StateMachine) checkVacancyConfirmation(now time.Time) StateTransition {
	elapsed := now.Sub(sm.state.PendingSince)
	timeMet := elapsed >= sm.config.VacancyConfirmDuration
	samplesMet := sm.state.ConsecutiveVacantSamples >= sm.config.VacancyConfirmSamples

	if timeMet && samplesMet {
		return sm.transitionTo(StateVacantConfirmed, now,
			fmt.Sprintf("vacancy confirmed: %d samples over %v",
				sm.state.ConsecutiveVacantSamples, elapsed.Truncate(time.Millisecond)))
	}

	// Remain in VACANT_PENDING.
	return StateTransition{
		ZoneID:    sm.state.ZoneID,
		From:      StateVacantPending,
		To:        StateVacantPending,
		Reason:    fmt.Sprintf("pending: samples=%d/%d, time=%v/%v", sm.state.ConsecutiveVacantSamples, sm.config.VacancyConfirmSamples, elapsed.Truncate(time.Millisecond), sm.config.VacancyConfirmDuration),
		Timestamp: now,
	}
}

// handleCameraOffline processes a CAMERA_OFFLINE event.
//
// R3 SAFETY CRITICAL: Camera offline MUST always produce UNKNOWN state.
// Camera offline MUST NEVER produce VACANT_CONFIRMED.
// This is a fundamental safety invariant of the system.
func (sm *StateMachine) handleCameraOffline(event CameraEvent) StateTransition {
	// Reset pending vacancy tracking - cannot confirm vacancy without camera.
	sm.state.PendingSince = time.Time{}
	sm.state.ConsecutiveVacantSamples = 0
	sm.state.PersonCount = 0

	if sm.state.CurrentState == StateUnknown {
		return StateTransition{
			ZoneID:    sm.state.ZoneID,
			From:      StateUnknown,
			To:        StateUnknown,
			Reason:    "camera offline, already unknown",
			Timestamp: event.Timestamp,
		}
	}

	return sm.transitionTo(StateUnknown, event.Timestamp,
		fmt.Sprintf("camera offline (id=%s)", event.CameraID))
}

// handleCameraOnline processes a CAMERA_ONLINE event.
// Camera coming online does not change state - actual detection data is needed.
func (sm *StateMachine) handleCameraOnline(event CameraEvent) StateTransition {
	// Camera online is informational - state change requires detection data.
	return StateTransition{
		ZoneID:    sm.state.ZoneID,
		From:      sm.state.CurrentState,
		To:        sm.state.CurrentState,
		Reason:    fmt.Sprintf("camera online (id=%s), awaiting detection data", event.CameraID),
		Timestamp: event.Timestamp,
	}
}

// handleError processes an ERROR event.
//
// R3 SAFETY CRITICAL: Errors MUST always produce UNKNOWN state.
// Errors MUST NEVER produce VACANT_CONFIRMED.
// Parse failures, communication errors, and any other anomalies
// are fail-safe to UNKNOWN.
func (sm *StateMachine) handleError(event CameraEvent) StateTransition {
	// Reset pending vacancy tracking - cannot confirm vacancy during error.
	sm.state.PendingSince = time.Time{}
	sm.state.ConsecutiveVacantSamples = 0

	if sm.state.CurrentState == StateUnknown {
		return StateTransition{
			ZoneID:    sm.state.ZoneID,
			From:      StateUnknown,
			To:        StateUnknown,
			Reason:    fmt.Sprintf("error: %s, already unknown", event.ErrorMessage),
			Timestamp: event.Timestamp,
		}
	}

	return sm.transitionTo(StateUnknown, event.Timestamp,
		fmt.Sprintf("error: %s", event.ErrorMessage))
}

// transitionTo performs the actual state transition and updates ZoneState.
func (sm *StateMachine) transitionTo(newState OccupancyState, ts time.Time, reason string) StateTransition {
	from := sm.state.CurrentState
	sm.state.PreviousState = from
	sm.state.CurrentState = newState
	sm.state.StateChangedAt = ts

	return StateTransition{
		ZoneID:    sm.state.ZoneID,
		From:      from,
		To:        newState,
		Reason:    reason,
		Timestamp: ts,
	}
}
