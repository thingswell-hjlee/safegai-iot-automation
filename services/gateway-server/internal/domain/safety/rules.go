package safety

// This file implements individual safety rules R-01 through R-04.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
//
// Each rule is a pure function: deterministic, no side effects.
// Same input ALWAYS produces same output.
//
// Rule R-05 (duplicate suppression) is implemented in dedup.go
// because it operates on the output stream, not on individual evaluations.

import (
	"fmt"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// EvaluateOccupiedRunning implements Rule R-01.
//
// [R3] SAFETY CRITICAL FUNCTION
//
// Rule: zone=OCCUPIED AND machine=RUNNING => WARNING + STOP_REQUEST_REQUIRED
//
// When a zone is occupied and equipment is running, a stop request must
// be sent to the PLC or Safety Relay. The app does NOT directly cut power.
//
// Parameters:
//   - zoneState: current occupancy state of the zone
//   - equipState: current running state of the equipment
//   - zoneID: identifier of the zone
//   - equipmentID: identifier of the equipment
//   - correlationID: event correlation ID
//
// Returns: DecisionResult with STOP_REQUEST_REQUIRED if rule triggers, nil otherwise.
func EvaluateOccupiedRunning(
	zoneState events.OccupancyState,
	equipState events.EquipmentState,
	zoneID, equipmentID, correlationID string,
) *DecisionResult {
	if zoneState == events.OccupancyOccupied && equipState == events.EquipmentRunning {
		return &DecisionResult{
			Decision:      DecisionStopRequestRequired,
			Rule:          "R-01",
			ZoneID:        zoneID,
			EquipmentID:   equipmentID,
			Reason:        fmt.Sprintf("zone %s is OCCUPIED and equipment %s is RUNNING: stop request required", zoneID, equipmentID),
			Timestamp:     time.Now(),
			CorrelationID: correlationID,
		}
	}
	return nil
}

// EvaluateRestartInterlock implements Rule R-02.
//
// [R3] SAFETY CRITICAL FUNCTION
//
// Rule: restart_request=ON AND zone != VACANT_CONFIRMED => RESTART_INTERLOCK
//
// A restart is only allowed when the zone is VACANT_CONFIRMED.
// All other states (OCCUPIED, VACANT_PENDING, UNKNOWN, STALE) block restart.
// [R3] No automatic restart from AI vacancy alone.
// [R3] UNKNOWN and STALE ALWAYS block restart (also covered by R-03).
//
// Parameters:
//   - zoneState: current occupancy state of the zone
//   - restartRequested: whether a restart has been requested for equipment in this zone
//   - zoneID: identifier of the zone
//   - equipmentID: identifier of the equipment requesting restart
//   - correlationID: event correlation ID
//
// Returns: DecisionResult with RESTART_INTERLOCK if rule triggers, nil otherwise.
func EvaluateRestartInterlock(
	zoneState events.OccupancyState,
	restartRequested bool,
	zoneID, equipmentID, correlationID string,
) *DecisionResult {
	if !restartRequested {
		return nil
	}

	// Only VACANT_CONFIRMED allows restart
	if zoneState != events.OccupancyVacantConfirmed {
		return &DecisionResult{
			Decision:      DecisionRestartInterlock,
			Rule:          "R-02",
			ZoneID:        zoneID,
			EquipmentID:   equipmentID,
			Reason:        fmt.Sprintf("restart blocked for equipment %s: zone %s is %s (only VACANT_CONFIRMED allows restart)", equipmentID, zoneID, string(zoneState)),
			Timestamp:     time.Now(),
			CorrelationID: correlationID,
		}
	}
	return nil
}

// EvaluateSafetyUnavailable implements Rule R-03.
//
// [R3] SAFETY CRITICAL FUNCTION
//
// Rule: UNKNOWN OR STALE => SAFETY_CONFIRMATION_UNAVAILABLE
//
// When the zone occupancy state is UNKNOWN or STALE, safety confirmation
// cannot be provided. This ALWAYS blocks restart regardless of other conditions.
// Camera offline = UNKNOWN (never VACANT).
// Data timeout = STALE (never VACANT_CONFIRMED).
//
// Parameters:
//   - zoneState: current occupancy state of the zone
//   - zoneID: identifier of the zone
//   - equipmentID: identifier of related equipment
//   - correlationID: event correlation ID
//
// Returns: DecisionResult with SAFETY_CONFIRMATION_UNAVAILABLE if rule triggers, nil otherwise.
func EvaluateSafetyUnavailable(
	zoneState events.OccupancyState,
	zoneID, equipmentID, correlationID string,
) *DecisionResult {
	if zoneState == events.OccupancyUnknown || zoneState == events.OccupancyStale {
		return &DecisionResult{
			Decision:      DecisionSafetyConfirmationUnavailable,
			Rule:          "R-03",
			ZoneID:        zoneID,
			EquipmentID:   equipmentID,
			Reason:        fmt.Sprintf("safety confirmation unavailable: zone %s is %s", zoneID, string(zoneState)),
			Timestamp:     time.Now(),
			CorrelationID: correlationID,
		}
	}
	return nil
}

// EvaluateMaintenanceWindow implements Rule R-04.
//
// [R3] SAFETY CRITICAL FUNCTION
//
// Rule: approved work window AND equipment=STOPPED => MAINTENANCE_MONITORING
//
// When a work window is active (approved by authorized personnel) and
// equipment is confirmed stopped, the system enters maintenance monitoring mode.
// If equipment is RUNNING during a maintenance window, that is a WARNING
// (equipment should have been stopped before maintenance).
//
// Parameters:
//   - workWindowActive: whether an approved maintenance window is currently active
//   - equipState: current running state of the equipment
//   - zoneID: identifier of the zone
//   - equipmentID: identifier of the equipment
//   - correlationID: event correlation ID
//
// Returns: DecisionResult if rule triggers, nil otherwise.
func EvaluateMaintenanceWindow(
	workWindowActive bool,
	equipState events.EquipmentState,
	zoneID, equipmentID, correlationID string,
) *DecisionResult {
	if !workWindowActive {
		return nil
	}

	if equipState == events.EquipmentStopped {
		return &DecisionResult{
			Decision:      DecisionMaintenanceMonitoring,
			Rule:          "R-04",
			ZoneID:        zoneID,
			EquipmentID:   equipmentID,
			Reason:        fmt.Sprintf("maintenance window active and equipment %s is STOPPED: monitoring mode", equipmentID),
			Timestamp:     time.Now(),
			CorrelationID: correlationID,
		}
	}

	if equipState == events.EquipmentRunning {
		return &DecisionResult{
			Decision:      DecisionWarning,
			Rule:          "R-04",
			ZoneID:        zoneID,
			EquipmentID:   equipmentID,
			Reason:        fmt.Sprintf("maintenance window active but equipment %s is RUNNING: equipment should be stopped for maintenance", equipmentID),
			Timestamp:     time.Now(),
			CorrelationID: correlationID,
		}
	}

	return nil
}
