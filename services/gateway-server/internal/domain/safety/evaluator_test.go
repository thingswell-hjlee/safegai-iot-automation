package safety

// Comprehensive truth-table tests for the Activity Engine evaluator.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
// These tests verify the complete behavior of the safety rule evaluator,
// including priority ordering, determinism, and all state combinations.

import (
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// helper to create a simple evaluation context for one zone+equipment pair.
func makeSimpleCtx(
	zoneID string, zoneState events.OccupancyState,
	equipID string, equipState events.EquipmentState,
	restartReq bool, workWindow bool,
) EvaluationContext {
	return EvaluationContext{
		ZoneStates:        map[string]events.OccupancyState{zoneID: zoneState},
		EquipmentStates:   map[string]events.EquipmentState{equipID: equipState},
		RestartRequested:  map[string]bool{equipID: restartReq},
		ActiveWorkWindows: map[string]bool{zoneID: workWindow},
		ZoneEquipmentMap:  map[string][]string{zoneID: {equipID}},
		CorrelationID:     "test-corr",
	}
}

// newTestEvaluator creates an evaluator with a long suppression window (won't trigger in tests).
func newTestEvaluator() *Evaluator {
	return NewEvaluator(1 * time.Millisecond) // very short for non-dedup tests
}

// newNoDedupEvaluator creates a fresh evaluator per call to avoid dedup interference.
func newNoDedupEvaluator() *Evaluator {
	return NewEvaluator(0) // zero window = no dedup
}

// containsDecision checks if any result has the given decision.
func containsDecision(results []DecisionResult, d SafetyDecision) bool {
	for _, r := range results {
		if r.Decision == d {
			return true
		}
	}
	return false
}

// containsRule checks if any result has the given rule.
func containsRule(results []DecisionResult, rule string) bool {
	for _, r := range results {
		if r.Rule == rule {
			return true
		}
	}
	return false
}

// TestR01_OccupiedRunning_Evaluator verifies R-01 in full evaluator context.
// OCCUPIED + RUNNING = STOP_REQUEST_REQUIRED
func TestR01_OccupiedRunning_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyOccupied, "eq-1", events.EquipmentRunning, false, false)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionStopRequestRequired) {
		t.Fatalf("expected STOP_REQUEST_REQUIRED in results: %+v", results)
	}
	if !containsRule(results, "R-01") {
		t.Fatalf("expected rule R-01 in results: %+v", results)
	}
}

// TestR01_OccupiedStopped_Evaluator verifies OCCUPIED + STOPPED = SAFE.
func TestR01_OccupiedStopped_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyOccupied, "eq-1", events.EquipmentStopped, false, false)

	results := e.Evaluate(ctx)
	if len(results) != 1 || results[0].Decision != DecisionSafe {
		t.Fatalf("expected only SAFE for OCCUPIED + STOPPED, got: %+v", results)
	}
}

// TestR01_VacantConfirmedRunning_Evaluator verifies VACANT_CONFIRMED + RUNNING = SAFE.
func TestR01_VacantConfirmedRunning_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyVacantConfirmed, "eq-1", events.EquipmentRunning, false, false)

	results := e.Evaluate(ctx)
	if len(results) != 1 || results[0].Decision != DecisionSafe {
		t.Fatalf("expected only SAFE for VACANT_CONFIRMED + RUNNING, got: %+v", results)
	}
}

// TestR02_RestartBlockedByOccupied_Evaluator verifies restart + OCCUPIED = RESTART_INTERLOCK.
func TestR02_RestartBlockedByOccupied_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyOccupied, "eq-1", events.EquipmentStopped, true, false)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionRestartInterlock) {
		t.Fatalf("expected RESTART_INTERLOCK for restart + OCCUPIED, got: %+v", results)
	}
}

// TestR02_RestartBlockedByUnknown_Evaluator verifies restart + UNKNOWN = SAFETY_CONFIRMATION_UNAVAILABLE.
// [R3] UNKNOWN always blocks restart. R-03 takes priority over R-02.
func TestR02_RestartBlockedByUnknown_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyUnknown, "eq-1", events.EquipmentStopped, true, false)

	results := e.Evaluate(ctx)
	// R-03 fires with highest priority for UNKNOWN
	if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
		t.Fatalf("expected SAFETY_CONFIRMATION_UNAVAILABLE for restart + UNKNOWN, got: %+v", results)
	}
}

// TestR02_RestartBlockedByStale_Evaluator verifies restart + STALE = SAFETY_CONFIRMATION_UNAVAILABLE.
// [R3] STALE always blocks restart. R-03 takes priority.
func TestR02_RestartBlockedByStale_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyStale, "eq-1", events.EquipmentStopped, true, false)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
		t.Fatalf("expected SAFETY_CONFIRMATION_UNAVAILABLE for restart + STALE, got: %+v", results)
	}
}

// TestR02_RestartBlockedByPending_Evaluator verifies restart + VACANT_PENDING = RESTART_INTERLOCK.
func TestR02_RestartBlockedByPending_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyVacantPending, "eq-1", events.EquipmentStopped, true, false)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionRestartInterlock) {
		t.Fatalf("expected RESTART_INTERLOCK for restart + VACANT_PENDING, got: %+v", results)
	}
}

// TestR02_RestartAllowedByConfirmed_Evaluator verifies restart + VACANT_CONFIRMED = SAFE.
// [R3] Only VACANT_CONFIRMED allows restart.
func TestR02_RestartAllowedByConfirmed_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyVacantConfirmed, "eq-1", events.EquipmentStopped, true, false)

	results := e.Evaluate(ctx)
	if len(results) != 1 || results[0].Decision != DecisionSafe {
		t.Fatalf("expected only SAFE for restart + VACANT_CONFIRMED, got: %+v", results)
	}
}

// TestR03_UnknownUnavailable_Evaluator verifies UNKNOWN = SAFETY_CONFIRMATION_UNAVAILABLE.
func TestR03_UnknownUnavailable_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyUnknown, "eq-1", events.EquipmentRunning, false, false)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
		t.Fatalf("expected SAFETY_CONFIRMATION_UNAVAILABLE for UNKNOWN, got: %+v", results)
	}
	// R-03 supersedes R-01 (even though RUNNING)
	if containsDecision(results, DecisionStopRequestRequired) {
		t.Fatalf("R-03 should supersede R-01 for UNKNOWN zone, got: %+v", results)
	}
}

// TestR03_StaleUnavailable_Evaluator verifies STALE = SAFETY_CONFIRMATION_UNAVAILABLE.
func TestR03_StaleUnavailable_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyStale, "eq-1", events.EquipmentRunning, false, false)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
		t.Fatalf("expected SAFETY_CONFIRMATION_UNAVAILABLE for STALE, got: %+v", results)
	}
}

// TestR04_MaintenanceWindow_Evaluator verifies work window + STOPPED = MAINTENANCE_MONITORING.
func TestR04_MaintenanceWindow_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyVacantConfirmed, "eq-1", events.EquipmentStopped, false, true)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionMaintenanceMonitoring) {
		t.Fatalf("expected MAINTENANCE_MONITORING for work window + STOPPED, got: %+v", results)
	}
}

// TestR04_MaintenanceWindowRunning_Evaluator verifies work window + RUNNING = WARNING.
func TestR04_MaintenanceWindowRunning_Evaluator(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyVacantConfirmed, "eq-1", events.EquipmentRunning, false, true)

	results := e.Evaluate(ctx)
	if !containsDecision(results, DecisionWarning) {
		t.Fatalf("expected WARNING for work window + RUNNING, got: %+v", results)
	}
}

// TestDeterministic verifies same input always produces same output (run 100 times).
// [R3] Rules MUST be deterministic.
func TestDeterministic(t *testing.T) {
	type testCase struct {
		name       string
		zoneState  events.OccupancyState
		equipState events.EquipmentState
		restart    bool
		window     bool
	}

	cases := []testCase{
		{"occupied-running", events.OccupancyOccupied, events.EquipmentRunning, false, false},
		{"unknown-stopped-restart", events.OccupancyUnknown, events.EquipmentStopped, true, false},
		{"stale-running", events.OccupancyStale, events.EquipmentRunning, false, false},
		{"vacant-confirmed-stopped-window", events.OccupancyVacantConfirmed, events.EquipmentStopped, false, true},
		{"vacant-pending-stopped-restart", events.OccupancyVacantPending, events.EquipmentStopped, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Get reference result
			e := newNoDedupEvaluator()
			ctx := makeSimpleCtx("z1", tc.zoneState, "e1", tc.equipState, tc.restart, tc.window)
			refResults := e.Evaluate(ctx)

			// Run 100 times and compare
			for i := 0; i < 100; i++ {
				e2 := newNoDedupEvaluator()
				results := e2.Evaluate(ctx)

				if len(results) != len(refResults) {
					t.Fatalf("iteration %d: got %d results, expected %d", i, len(results), len(refResults))
				}
				for j := range results {
					if results[j].Decision != refResults[j].Decision {
						t.Fatalf("iteration %d result %d: got %s, expected %s",
							i, j, results[j].Decision, refResults[j].Decision)
					}
					if results[j].Rule != refResults[j].Rule {
						t.Fatalf("iteration %d result %d: got rule %s, expected %s",
							i, j, results[j].Rule, refResults[j].Rule)
					}
				}
			}
		})
	}
}

// TestAllStatesWithAllEquipment is an exhaustive combination test.
// [R3] Verifies no unexpected panics or behavior for any valid state combination.
func TestAllStatesWithAllEquipment(t *testing.T) {
	for _, zoneState := range events.ValidOccupancyStates {
		for _, equipState := range events.ValidEquipmentStates {
			for _, restart := range []bool{false, true} {
				for _, window := range []bool{false, true} {
					e := newNoDedupEvaluator()
					ctx := makeSimpleCtx("z1", zoneState, "e1", equipState, restart, window)

					results := e.Evaluate(ctx)
					if len(results) == 0 {
						t.Fatalf("expected at least one result for zone=%s equip=%s restart=%v window=%v",
							zoneState, equipState, restart, window)
					}

					// Verify safety invariants for every combination
					for _, r := range results {
						if !r.Decision.IsValid() {
							t.Fatalf("invalid decision %s for zone=%s equip=%s",
								r.Decision, zoneState, equipState)
						}
					}

					// [R3] UNKNOWN and STALE must NEVER produce SAFE alone when restart is requested
					if (zoneState == events.OccupancyUnknown || zoneState == events.OccupancyStale) && restart {
						if len(results) == 1 && results[0].Decision == DecisionSafe {
							t.Fatalf("[R3 VIOLATION] UNKNOWN/STALE + restart produced only SAFE: zone=%s restart=%v",
								zoneState, restart)
						}
					}

					// [R3] UNKNOWN and STALE must always produce SAFETY_CONFIRMATION_UNAVAILABLE
					if zoneState == events.OccupancyUnknown || zoneState == events.OccupancyStale {
						if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
							t.Fatalf("[R3 VIOLATION] UNKNOWN/STALE did not produce SAFETY_CONFIRMATION_UNAVAILABLE: zone=%s",
								zoneState)
						}
					}

					// [R3] Only VACANT_CONFIRMED can produce a lone SAFE with restart
					if restart && zoneState != events.OccupancyVacantConfirmed {
						if len(results) == 1 && results[0].Decision == DecisionSafe {
							t.Fatalf("[R3 VIOLATION] restart + non-VACANT_CONFIRMED produced only SAFE: zone=%s",
								zoneState)
						}
					}
				}
			}
		}
	}
}

// TestR03PriorityOverR01 verifies that R-03 takes priority over R-01.
// When zone is UNKNOWN and equipment is RUNNING, R-03 fires but R-01 does not.
func TestR03PriorityOverR01(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyUnknown, "eq-1", events.EquipmentRunning, false, false)

	results := e.Evaluate(ctx)
	if containsDecision(results, DecisionStopRequestRequired) {
		t.Fatal("R-01 should not fire when R-03 applies (UNKNOWN zone)")
	}
	if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
		t.Fatal("R-03 should fire for UNKNOWN zone")
	}
}

// TestR03PriorityOverR02 verifies that R-03 takes priority over R-02.
// When zone is STALE and restart is requested, R-03 fires.
func TestR03PriorityOverR02(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyStale, "eq-1", events.EquipmentStopped, true, false)

	results := e.Evaluate(ctx)
	// R-03 should fire and supersede everything
	if !containsDecision(results, DecisionSafetyConfirmationUnavailable) {
		t.Fatal("R-03 should fire for STALE zone even with restart requested")
	}
	// R-02 should not fire because R-03 supersedes it
	if containsDecision(results, DecisionRestartInterlock) {
		t.Fatal("R-02 should not fire when R-03 applies (STALE zone)")
	}
}

// TestMultiZoneEvaluation verifies the evaluator handles multiple zones correctly.
func TestMultiZoneEvaluation(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := EvaluationContext{
		ZoneStates: map[string]events.OccupancyState{
			"zone-A": events.OccupancyOccupied,
			"zone-B": events.OccupancyVacantConfirmed,
			"zone-C": events.OccupancyUnknown,
		},
		EquipmentStates: map[string]events.EquipmentState{
			"eq-A": events.EquipmentRunning,
			"eq-B": events.EquipmentRunning,
			"eq-C": events.EquipmentStopped,
		},
		RestartRequested:  map[string]bool{},
		ActiveWorkWindows: map[string]bool{},
		ZoneEquipmentMap: map[string][]string{
			"zone-A": {"eq-A"},
			"zone-B": {"eq-B"},
			"zone-C": {"eq-C"},
		},
		CorrelationID: "multi-test",
	}

	results := e.Evaluate(ctx)

	// zone-A: OCCUPIED + RUNNING = STOP_REQUEST_REQUIRED
	foundA := false
	for _, r := range results {
		if r.ZoneID == "zone-A" && r.Decision == DecisionStopRequestRequired {
			foundA = true
		}
	}
	if !foundA {
		t.Fatal("expected STOP_REQUEST_REQUIRED for zone-A (OCCUPIED + RUNNING)")
	}

	// zone-B: VACANT_CONFIRMED + RUNNING = SAFE
	foundB := false
	for _, r := range results {
		if r.ZoneID == "zone-B" && r.Decision == DecisionSafe {
			foundB = true
		}
	}
	if !foundB {
		t.Fatal("expected SAFE for zone-B (VACANT_CONFIRMED + RUNNING)")
	}

	// zone-C: UNKNOWN = SAFETY_CONFIRMATION_UNAVAILABLE
	foundC := false
	for _, r := range results {
		if r.ZoneID == "zone-C" && r.Decision == DecisionSafetyConfirmationUnavailable {
			foundC = true
		}
	}
	if !foundC {
		t.Fatal("expected SAFETY_CONFIRMATION_UNAVAILABLE for zone-C (UNKNOWN)")
	}
}

// TestNoAutomaticRestartFromAIVacancy verifies the key safety invariant:
// [R3] No automatic restart from AI vacancy alone.
// Even VACANT_CONFIRMED by itself does not cause restart - it merely
// does not BLOCK restart (produces SAFE, not a restart command).
func TestNoAutomaticRestartFromAIVacancy(t *testing.T) {
	e := newNoDedupEvaluator()
	ctx := makeSimpleCtx("zone-1", events.OccupancyVacantConfirmed, "eq-1", events.EquipmentStopped, false, false)

	results := e.Evaluate(ctx)
	// Should produce SAFE - not a restart command
	if len(results) != 1 || results[0].Decision != DecisionSafe {
		t.Fatalf("VACANT_CONFIRMED without restart request should be SAFE, got: %+v", results)
	}
	// Verify no restart-related decisions
	for _, r := range results {
		if r.Decision == DecisionRestartInterlock {
			t.Fatal("should not see RESTART_INTERLOCK without restart request")
		}
	}
}

// TestEvaluateSimple verifies the convenience function.
func TestEvaluateSimple(t *testing.T) {
	e := newNoDedupEvaluator()
	results := e.EvaluateSimple(
		events.OccupancyOccupied,
		events.EquipmentRunning,
		false, false,
		"z1", "e1", "c1",
	)
	if !containsDecision(results, DecisionStopRequestRequired) {
		t.Fatalf("EvaluateSimple should return STOP_REQUEST_REQUIRED for OCCUPIED+RUNNING")
	}
}

// IsValid checks if the SafetyDecision string is a recognized valid value.
func (d SafetyDecision) IsValid() bool {
	switch d {
	case DecisionSafe, DecisionWarning, DecisionStopRequestRequired,
		DecisionRestartInterlock, DecisionSafetyConfirmationUnavailable,
		DecisionMaintenanceMonitoring:
		return true
	}
	return false
}
