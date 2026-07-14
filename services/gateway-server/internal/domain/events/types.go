// Package events defines domain event types and state constants
// for the SafeGAI edge gateway.
package events

// OccupancyState represents the occupancy state of a zone.
// UNKNOWN and STALE are never treated as vacancy.
// Only VACANT_CONFIRMED satisfies vacancy.
type OccupancyState string

const (
	OccupancyOccupied        OccupancyState = "OCCUPIED"
	OccupancyVacantPending   OccupancyState = "VACANT_PENDING"
	OccupancyVacantConfirmed OccupancyState = "VACANT_CONFIRMED"
	OccupancyUnknown         OccupancyState = "UNKNOWN"
	OccupancyStale           OccupancyState = "STALE"
)

// ValidOccupancyStates contains all valid OccupancyState values.
var ValidOccupancyStates = []OccupancyState{
	OccupancyOccupied,
	OccupancyVacantPending,
	OccupancyVacantConfirmed,
	OccupancyUnknown,
	OccupancyStale,
}

// IsValid returns true if the OccupancyState is a recognized constant.
func (s OccupancyState) IsValid() bool {
	for _, v := range ValidOccupancyStates {
		if s == v {
			return true
		}
	}
	return false
}

// IsVacant returns true only for VACANT_CONFIRMED.
// UNKNOWN and STALE are explicitly not vacancy.
func (s OccupancyState) IsVacant() bool {
	return s == OccupancyVacantConfirmed
}

// EquipmentState represents the running state of equipment.
// Equipment states are determined from DI (Digital Input) signals.
// Stale DI input must result in UNKNOWN state.
type EquipmentState string

const (
	EquipmentRunning          EquipmentState = "RUNNING"
	EquipmentStopped          EquipmentState = "STOPPED"
	EquipmentRestartRequested EquipmentState = "RESTART_REQUESTED"
	EquipmentUnknown          EquipmentState = "UNKNOWN"
)

// ValidEquipmentStates contains all valid EquipmentState values.
var ValidEquipmentStates = []EquipmentState{
	EquipmentRunning,
	EquipmentStopped,
	EquipmentRestartRequested,
	EquipmentUnknown,
}

// IsValid returns true if the EquipmentState is a recognized constant.
func (s EquipmentState) IsValid() bool {
	for _, v := range ValidEquipmentStates {
		if s == v {
			return true
		}
	}
	return false
}

// SafetyDecision represents the outcome of a safety rule evaluation.
type SafetyDecision string

const (
	SafetyDecisionSafe                          SafetyDecision = "SAFE"
	SafetyDecisionWarning                       SafetyDecision = "WARNING"
	SafetyDecisionStopRequestRequired           SafetyDecision = "STOP_REQUEST_REQUIRED"
	SafetyDecisionRestartInterlock              SafetyDecision = "RESTART_INTERLOCK"
	SafetyDecisionSafetyConfirmationUnavailable SafetyDecision = "SAFETY_CONFIRMATION_UNAVAILABLE"
	SafetyDecisionMaintenanceMonitoring         SafetyDecision = "MAINTENANCE_MONITORING"
)

// ValidSafetyDecisions contains all valid SafetyDecision values.
var ValidSafetyDecisions = []SafetyDecision{
	SafetyDecisionSafe,
	SafetyDecisionWarning,
	SafetyDecisionStopRequestRequired,
	SafetyDecisionRestartInterlock,
	SafetyDecisionSafetyConfirmationUnavailable,
	SafetyDecisionMaintenanceMonitoring,
}

// IsValid returns true if the SafetyDecision is a recognized constant.
func (d SafetyDecision) IsValid() bool {
	for _, v := range ValidSafetyDecisions {
		if d == v {
			return true
		}
	}
	return false
}
