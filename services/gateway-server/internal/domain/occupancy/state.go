// Package occupancy implements the zone occupancy state machine for the SafeGAI
// edge gateway. This is an R3 SAFETY CRITICAL component.
//
// Safety Invariants:
//   - Only VACANT_CONFIRMED satisfies the vacancy condition.
//   - Initial state for all zones MUST be UNKNOWN.
//   - Camera offline MUST produce UNKNOWN, NEVER VACANT.
//   - STALE timeout MUST NOT produce VACANT_CONFIRMED.
//   - UNKNOWN and STALE block equipment restart.
//
// These invariants are enforced by the state machine and verified by
// comprehensive truth-table tests and explicit forbidden-transition tests.
package occupancy

import "time"

// OccupancyState represents the occupancy state of a monitored zone.
// R3 SAFETY CRITICAL: Only VACANT_CONFIRMED satisfies the vacancy condition.
type OccupancyState string

const (
	// StateOccupied indicates a person has been detected in the zone.
	StateOccupied OccupancyState = "OCCUPIED"

	// StateVacantPending indicates no person detected but confirmation is pending.
	// This state does NOT satisfy the vacancy condition.
	StateVacantPending OccupancyState = "VACANT_PENDING"

	// StateVacantConfirmed indicates vacancy has been confirmed by meeting both
	// the time duration and consecutive sample count requirements.
	// R3 SAFETY CRITICAL: This is the ONLY state that satisfies the vacancy condition.
	StateVacantConfirmed OccupancyState = "VACANT_CONFIRMED"

	// StateUnknown indicates the zone state cannot be determined (e.g. camera offline,
	// system startup, or communication error).
	// R3 SAFETY CRITICAL: UNKNOWN does NOT satisfy vacancy. UNKNOWN blocks restart.
	StateUnknown OccupancyState = "UNKNOWN"

	// StateStale indicates no data has been received within the staleness timeout.
	// R3 SAFETY CRITICAL: STALE does NOT satisfy vacancy. STALE blocks restart.
	StateStale OccupancyState = "STALE"
)

// ZoneState holds the complete occupancy state for a single monitored zone.
// R3 SAFETY CRITICAL: The initial state of all zones MUST be UNKNOWN.
type ZoneState struct {
	// ZoneID uniquely identifies the monitored zone.
	ZoneID string

	// CurrentState is the current occupancy state of the zone.
	CurrentState OccupancyState

	// PreviousState is the state before the most recent transition.
	PreviousState OccupancyState

	// PersonCount is the most recent person count from the camera.
	PersonCount int

	// LastEventAt is the timestamp of the most recent event received.
	LastEventAt time.Time

	// StateChangedAt is the timestamp when the current state was entered.
	StateChangedAt time.Time

	// PendingSince is the timestamp when VACANT_PENDING was entered.
	// Zero value if not in VACANT_PENDING state.
	PendingSince time.Time

	// ConsecutiveVacantSamples counts consecutive no-person detections
	// since entering VACANT_PENDING.
	ConsecutiveVacantSamples int
}

// IsSafeVacant returns true ONLY when the zone is in VACANT_CONFIRMED state.
//
// R3 SAFETY CRITICAL: This method is the authoritative check for vacancy.
// It MUST return false for ALL other states including UNKNOWN and STALE.
// Camera offline, data timeout, and parse errors MUST NOT produce a true result.
//
// Callers MUST use this method (not direct state comparison) to determine
// whether vacancy is satisfied for safety decisions.
func (z *ZoneState) IsSafeVacant() bool {
	return z.CurrentState == StateVacantConfirmed
}
