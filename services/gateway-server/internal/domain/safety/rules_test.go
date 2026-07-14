package safety

// Tests for individual safety rules R-01 through R-04.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
// These tests verify the correctness of individual safety rule functions.

import (
	"testing"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// TestR01_OccupiedRunning verifies R-01 triggers for OCCUPIED + RUNNING.
func TestR01_OccupiedRunning(t *testing.T) {
	result := EvaluateOccupiedRunning(
		events.OccupancyOccupied,
		events.EquipmentRunning,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected STOP_REQUEST_REQUIRED for OCCUPIED + RUNNING, got nil")
	}
	if result.Decision != DecisionStopRequestRequired {
		t.Fatalf("expected STOP_REQUEST_REQUIRED, got %s", result.Decision)
	}
	if result.Rule != "R-01" {
		t.Fatalf("expected rule R-01, got %s", result.Rule)
	}
}

// TestR01_OccupiedStopped verifies R-01 does NOT trigger for OCCUPIED + STOPPED.
func TestR01_OccupiedStopped(t *testing.T) {
	result := EvaluateOccupiedRunning(
		events.OccupancyOccupied,
		events.EquipmentStopped,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for OCCUPIED + STOPPED, got %s", result.Decision)
	}
}

// TestR01_VacantConfirmedRunning verifies R-01 does NOT trigger for VACANT_CONFIRMED + RUNNING.
func TestR01_VacantConfirmedRunning(t *testing.T) {
	result := EvaluateOccupiedRunning(
		events.OccupancyVacantConfirmed,
		events.EquipmentRunning,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for VACANT_CONFIRMED + RUNNING, got %s", result.Decision)
	}
}

// TestR01_UnknownRunning verifies R-01 does NOT trigger for UNKNOWN + RUNNING.
// (UNKNOWN is handled by R-03 instead)
func TestR01_UnknownRunning(t *testing.T) {
	result := EvaluateOccupiedRunning(
		events.OccupancyUnknown,
		events.EquipmentRunning,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for UNKNOWN + RUNNING, got %s", result.Decision)
	}
}

// TestR01_StaleRunning verifies R-01 does NOT trigger for STALE + RUNNING.
func TestR01_StaleRunning(t *testing.T) {
	result := EvaluateOccupiedRunning(
		events.OccupancyStale,
		events.EquipmentRunning,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for STALE + RUNNING, got %s", result.Decision)
	}
}

// TestR01_VacantPendingRunning verifies R-01 does NOT trigger for VACANT_PENDING + RUNNING.
func TestR01_VacantPendingRunning(t *testing.T) {
	result := EvaluateOccupiedRunning(
		events.OccupancyVacantPending,
		events.EquipmentRunning,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for VACANT_PENDING + RUNNING, got %s", result.Decision)
	}
}

// TestR02_RestartBlockedByOccupied verifies R-02 blocks restart when zone is OCCUPIED.
func TestR02_RestartBlockedByOccupied(t *testing.T) {
	result := EvaluateRestartInterlock(
		events.OccupancyOccupied,
		true,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected RESTART_INTERLOCK for restart + OCCUPIED, got nil")
	}
	if result.Decision != DecisionRestartInterlock {
		t.Fatalf("expected RESTART_INTERLOCK, got %s", result.Decision)
	}
	if result.Rule != "R-02" {
		t.Fatalf("expected rule R-02, got %s", result.Rule)
	}
}

// TestR02_RestartBlockedByUnknown verifies R-02 blocks restart when zone is UNKNOWN.
// [R3] UNKNOWN ALWAYS blocks restart.
func TestR02_RestartBlockedByUnknown(t *testing.T) {
	result := EvaluateRestartInterlock(
		events.OccupancyUnknown,
		true,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected RESTART_INTERLOCK for restart + UNKNOWN, got nil")
	}
	if result.Decision != DecisionRestartInterlock {
		t.Fatalf("expected RESTART_INTERLOCK, got %s", result.Decision)
	}
}

// TestR02_RestartBlockedByStale verifies R-02 blocks restart when zone is STALE.
// [R3] STALE ALWAYS blocks restart.
func TestR02_RestartBlockedByStale(t *testing.T) {
	result := EvaluateRestartInterlock(
		events.OccupancyStale,
		true,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected RESTART_INTERLOCK for restart + STALE, got nil")
	}
	if result.Decision != DecisionRestartInterlock {
		t.Fatalf("expected RESTART_INTERLOCK, got %s", result.Decision)
	}
}

// TestR02_RestartBlockedByPending verifies R-02 blocks restart when zone is VACANT_PENDING.
// [R3] VACANT_PENDING is NOT sufficient for restart. Only VACANT_CONFIRMED.
func TestR02_RestartBlockedByPending(t *testing.T) {
	result := EvaluateRestartInterlock(
		events.OccupancyVacantPending,
		true,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected RESTART_INTERLOCK for restart + VACANT_PENDING, got nil")
	}
	if result.Decision != DecisionRestartInterlock {
		t.Fatalf("expected RESTART_INTERLOCK, got %s", result.Decision)
	}
}

// TestR02_RestartAllowedByConfirmed verifies R-02 does NOT trigger for VACANT_CONFIRMED.
// [R3] Only VACANT_CONFIRMED allows restart.
func TestR02_RestartAllowedByConfirmed(t *testing.T) {
	result := EvaluateRestartInterlock(
		events.OccupancyVacantConfirmed,
		true,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for restart + VACANT_CONFIRMED, got %s", result.Decision)
	}
}

// TestR02_NoRestartRequest verifies R-02 does NOT trigger when no restart is requested.
func TestR02_NoRestartRequest(t *testing.T) {
	for _, state := range events.ValidOccupancyStates {
		result := EvaluateRestartInterlock(
			state,
			false,
			"zone-1", "equip-1", "corr-1",
		)
		if result != nil {
			t.Fatalf("expected nil for no restart request + %s, got %s", state, result.Decision)
		}
	}
}

// TestR03_UnknownUnavailable verifies R-03 triggers for UNKNOWN.
func TestR03_UnknownUnavailable(t *testing.T) {
	result := EvaluateSafetyUnavailable(
		events.OccupancyUnknown,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected SAFETY_CONFIRMATION_UNAVAILABLE for UNKNOWN, got nil")
	}
	if result.Decision != DecisionSafetyConfirmationUnavailable {
		t.Fatalf("expected SAFETY_CONFIRMATION_UNAVAILABLE, got %s", result.Decision)
	}
	if result.Rule != "R-03" {
		t.Fatalf("expected rule R-03, got %s", result.Rule)
	}
}

// TestR03_StaleUnavailable verifies R-03 triggers for STALE.
func TestR03_StaleUnavailable(t *testing.T) {
	result := EvaluateSafetyUnavailable(
		events.OccupancyStale,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected SAFETY_CONFIRMATION_UNAVAILABLE for STALE, got nil")
	}
	if result.Decision != DecisionSafetyConfirmationUnavailable {
		t.Fatalf("expected SAFETY_CONFIRMATION_UNAVAILABLE, got %s", result.Decision)
	}
}

// TestR03_NotTriggeredByOccupied verifies R-03 does NOT trigger for OCCUPIED.
func TestR03_NotTriggeredByOccupied(t *testing.T) {
	result := EvaluateSafetyUnavailable(
		events.OccupancyOccupied,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for OCCUPIED, got %s", result.Decision)
	}
}

// TestR03_NotTriggeredByVacantPending verifies R-03 does NOT trigger for VACANT_PENDING.
func TestR03_NotTriggeredByVacantPending(t *testing.T) {
	result := EvaluateSafetyUnavailable(
		events.OccupancyVacantPending,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for VACANT_PENDING, got %s", result.Decision)
	}
}

// TestR03_NotTriggeredByVacantConfirmed verifies R-03 does NOT trigger for VACANT_CONFIRMED.
func TestR03_NotTriggeredByVacantConfirmed(t *testing.T) {
	result := EvaluateSafetyUnavailable(
		events.OccupancyVacantConfirmed,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for VACANT_CONFIRMED, got %s", result.Decision)
	}
}

// TestR04_MaintenanceWindowStopped verifies R-04 triggers for active window + STOPPED.
func TestR04_MaintenanceWindowStopped(t *testing.T) {
	result := EvaluateMaintenanceWindow(
		true,
		events.EquipmentStopped,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected MAINTENANCE_MONITORING for active window + STOPPED, got nil")
	}
	if result.Decision != DecisionMaintenanceMonitoring {
		t.Fatalf("expected MAINTENANCE_MONITORING, got %s", result.Decision)
	}
	if result.Rule != "R-04" {
		t.Fatalf("expected rule R-04, got %s", result.Rule)
	}
}

// TestR04_MaintenanceWindowRunning verifies R-04 produces WARNING for active window + RUNNING.
func TestR04_MaintenanceWindowRunning(t *testing.T) {
	result := EvaluateMaintenanceWindow(
		true,
		events.EquipmentRunning,
		"zone-1", "equip-1", "corr-1",
	)
	if result == nil {
		t.Fatal("expected WARNING for active window + RUNNING, got nil")
	}
	if result.Decision != DecisionWarning {
		t.Fatalf("expected WARNING, got %s", result.Decision)
	}
	if result.Rule != "R-04" {
		t.Fatalf("expected rule R-04, got %s", result.Rule)
	}
}

// TestR04_NoWindow verifies R-04 does NOT trigger when no work window is active.
func TestR04_NoWindow(t *testing.T) {
	for _, eqState := range events.ValidEquipmentStates {
		result := EvaluateMaintenanceWindow(
			false,
			eqState,
			"zone-1", "equip-1", "corr-1",
		)
		if result != nil {
			t.Fatalf("expected nil for no window + %s, got %s", eqState, result.Decision)
		}
	}
}

// TestR04_WindowWithUnknownEquipment verifies R-04 does NOT trigger for UNKNOWN equipment.
func TestR04_WindowWithUnknownEquipment(t *testing.T) {
	result := EvaluateMaintenanceWindow(
		true,
		events.EquipmentUnknown,
		"zone-1", "equip-1", "corr-1",
	)
	if result != nil {
		t.Fatalf("expected nil for active window + UNKNOWN equipment, got %s", result.Decision)
	}
}

// TestRulesPopulateFields verifies all rule functions populate result fields correctly.
func TestRulesPopulateFields(t *testing.T) {
	// Test R-01
	r01 := EvaluateOccupiedRunning(
		events.OccupancyOccupied, events.EquipmentRunning,
		"z1", "e1", "c1",
	)
	if r01.ZoneID != "z1" || r01.EquipmentID != "e1" || r01.CorrelationID != "c1" {
		t.Fatal("R-01 result fields not populated correctly")
	}
	if r01.Reason == "" {
		t.Fatal("R-01 result reason is empty")
	}
	if r01.Timestamp.IsZero() {
		t.Fatal("R-01 result timestamp is zero")
	}

	// Test R-02
	r02 := EvaluateRestartInterlock(
		events.OccupancyOccupied, true,
		"z2", "e2", "c2",
	)
	if r02.ZoneID != "z2" || r02.EquipmentID != "e2" || r02.CorrelationID != "c2" {
		t.Fatal("R-02 result fields not populated correctly")
	}

	// Test R-03
	r03 := EvaluateSafetyUnavailable(
		events.OccupancyUnknown,
		"z3", "e3", "c3",
	)
	if r03.ZoneID != "z3" || r03.EquipmentID != "e3" || r03.CorrelationID != "c3" {
		t.Fatal("R-03 result fields not populated correctly")
	}

	// Test R-04
	r04 := EvaluateMaintenanceWindow(
		true, events.EquipmentStopped,
		"z4", "e4", "c4",
	)
	if r04.ZoneID != "z4" || r04.EquipmentID != "e4" || r04.CorrelationID != "c4" {
		t.Fatal("R-04 result fields not populated correctly")
	}
}
