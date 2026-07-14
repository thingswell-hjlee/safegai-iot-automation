// Package safety implements the Activity Engine safety rule evaluator.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
//
// This package evaluates fixed safety rules combining occupancy and equipment
// state to produce deterministic safety decisions. All rules are deterministic:
// the same input ALWAYS produces the same output.
//
// Key safety invariants:
//   - UNKNOWN and STALE ALWAYS block restart (SAFETY_CONFIRMATION_UNAVAILABLE)
//   - No automatic restart from AI vacancy alone
//   - Only VACANT_CONFIRMED satisfies vacancy condition for restart
//   - Output commands are PLC/Safety Relay stop requests only
//   - Rules are evaluated in strict priority order
//
// Safety rules implemented:
//   - R-01: OCCUPIED + RUNNING = WARNING + STOP_REQUEST_REQUIRED
//   - R-02: restart + zone != VACANT_CONFIRMED = RESTART_INTERLOCK
//   - R-03: UNKNOWN or STALE = SAFETY_CONFIRMATION_UNAVAILABLE
//   - R-04: approved work window + STOPPED = MAINTENANCE_MONITORING
//   - R-05: duplicate suppression (same zone+equip+decision in time window)
package safety

import "time"

// SafetyDecision represents the outcome of a safety rule evaluation.
// [R3] All decisions are deterministic given the same input.
type SafetyDecision string

const (
	// DecisionSafe indicates no safety concern detected.
	DecisionSafe SafetyDecision = "SAFE"

	// DecisionWarning indicates a safety concern that requires attention.
	DecisionWarning SafetyDecision = "WARNING"

	// DecisionStopRequestRequired indicates equipment must receive a stop request
	// via PLC or Safety Relay. The app does NOT directly cut power.
	DecisionStopRequestRequired SafetyDecision = "STOP_REQUEST_REQUIRED"

	// DecisionRestartInterlock blocks equipment restart because zone vacancy
	// is not confirmed. Only VACANT_CONFIRMED allows restart.
	// [R3] No automatic restart from AI vacancy alone.
	DecisionRestartInterlock SafetyDecision = "RESTART_INTERLOCK"

	// DecisionSafetyConfirmationUnavailable indicates that safety state cannot
	// be determined (UNKNOWN or STALE zone state). This ALWAYS blocks restart.
	// [R3] UNKNOWN and STALE always produce this decision when restart is requested.
	DecisionSafetyConfirmationUnavailable SafetyDecision = "SAFETY_CONFIRMATION_UNAVAILABLE"

	// DecisionMaintenanceMonitoring indicates an approved maintenance window
	// with equipment confirmed stopped. Monitoring continues during maintenance.
	DecisionMaintenanceMonitoring SafetyDecision = "MAINTENANCE_MONITORING"
)

// DecisionResult represents the output of a safety rule evaluation.
// [R3] This struct is the primary output of the Activity Engine.
type DecisionResult struct {
	// Decision is the safety decision outcome.
	Decision SafetyDecision `json:"decision"`

	// Rule identifies which rule produced this decision (e.g. "R-01", "R-02").
	Rule string `json:"rule"`

	// ZoneID identifies the zone this decision applies to.
	ZoneID string `json:"zoneId"`

	// EquipmentID identifies the equipment this decision applies to.
	EquipmentID string `json:"equipmentId"`

	// Reason is a human-readable explanation of the decision.
	Reason string `json:"reason"`

	// Timestamp is when the decision was made.
	Timestamp time.Time `json:"timestamp"`

	// CorrelationID links this decision to the triggering event chain.
	CorrelationID string `json:"correlationId"`
}

// DecisionSeverity returns a numeric severity for ordering decisions.
// Higher severity = more critical safety concern.
// [R3] Severity ordering is fixed and deterministic.
func DecisionSeverity(d SafetyDecision) int {
	switch d {
	case DecisionSafetyConfirmationUnavailable:
		return 100
	case DecisionStopRequestRequired:
		return 90
	case DecisionRestartInterlock:
		return 80
	case DecisionWarning:
		return 70
	case DecisionMaintenanceMonitoring:
		return 60
	case DecisionSafe:
		return 0
	default:
		return 0
	}
}
