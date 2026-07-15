package safety

// Tests for Rule R-05: Duplicate suppression.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
// Verifies that duplicate decisions within the suppression window are
// correctly filtered, and that distinct decisions are never suppressed.

import (
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// TestDedupSuppressesDuplicate verifies that the same decision for the same
// zone+equipment within the suppression window is marked as duplicate.
func TestDedupSuppressesDuplicate(t *testing.T) {
	f := NewDedupFilter(5 * time.Second)

	result := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	// First call: not a duplicate
	if f.IsDuplicate(result) {
		t.Fatal("first decision should not be duplicate")
	}

	// Second call within window: duplicate
	if !f.IsDuplicate(result) {
		t.Fatal("second identical decision within window should be duplicate")
	}
}

// TestDedupAllowsAfterWindowExpires verifies that decisions are allowed
// again after the suppression window expires.
func TestDedupAllowsAfterWindowExpires(t *testing.T) {
	f := NewDedupFilter(10 * time.Millisecond)

	result := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	// First call: not a duplicate
	if f.IsDuplicate(result) {
		t.Fatal("first decision should not be duplicate")
	}

	// Wait for window to expire
	time.Sleep(15 * time.Millisecond)

	// Should no longer be duplicate
	if f.IsDuplicate(result) {
		t.Fatal("decision after window expiry should not be duplicate")
	}
}

// TestDedupDifferentDecisionNotSuppressed verifies that different decision
// types for the same zone+equipment are NOT suppressed.
func TestDedupDifferentDecisionNotSuppressed(t *testing.T) {
	f := NewDedupFilter(5 * time.Second)

	result1 := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	result2 := &DecisionResult{
		Decision:    DecisionRestartInterlock,
		Rule:        "R-02",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	if f.IsDuplicate(result1) {
		t.Fatal("first R-01 should not be duplicate")
	}
	if f.IsDuplicate(result2) {
		t.Fatal("first R-02 should not be duplicate (different decision type)")
	}
}

// TestDedupDifferentZoneNotSuppressed verifies that same decision type
// for different zones is NOT suppressed.
func TestDedupDifferentZoneNotSuppressed(t *testing.T) {
	f := NewDedupFilter(5 * time.Second)

	result1 := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	result2 := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-2",
		EquipmentID: "eq-1",
	}

	if f.IsDuplicate(result1) {
		t.Fatal("first zone-1 should not be duplicate")
	}
	if f.IsDuplicate(result2) {
		t.Fatal("zone-2 should not be duplicate (different zone)")
	}
}

// TestDedupDifferentEquipmentNotSuppressed verifies that same decision type
// for different equipment is NOT suppressed.
func TestDedupDifferentEquipmentNotSuppressed(t *testing.T) {
	f := NewDedupFilter(5 * time.Second)

	result1 := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	result2 := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-2",
	}

	if f.IsDuplicate(result1) {
		t.Fatal("first eq-1 should not be duplicate")
	}
	if f.IsDuplicate(result2) {
		t.Fatal("eq-2 should not be duplicate (different equipment)")
	}
}

// TestDedupSafeNeverDeduplicated verifies that SAFE decisions are never deduplicated.
func TestDedupSafeNeverDeduplicated(t *testing.T) {
	f := NewDedupFilter(5 * time.Second)

	result := &DecisionResult{
		Decision:    DecisionSafe,
		Rule:        "DEFAULT",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	for i := 0; i < 10; i++ {
		if f.IsDuplicate(result) {
			t.Fatalf("SAFE decision should never be deduplicated (iteration %d)", i)
		}
	}
}

// TestDedupReset verifies that Reset clears all suppression state.
func TestDedupReset(t *testing.T) {
	f := NewDedupFilter(5 * time.Second)

	result := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	f.IsDuplicate(result) // record first
	f.Reset()

	// After reset, should not be duplicate
	if f.IsDuplicate(result) {
		t.Fatal("after Reset, decision should not be duplicate")
	}
}

// TestDedupCleanup verifies that expired entries are removed.
func TestDedupCleanup(t *testing.T) {
	f := NewDedupFilter(10 * time.Millisecond)

	result := &DecisionResult{
		Decision:    DecisionStopRequestRequired,
		Rule:        "R-01",
		ZoneID:      "zone-1",
		EquipmentID: "eq-1",
	}

	f.IsDuplicate(result)
	time.Sleep(15 * time.Millisecond)
	f.Cleanup()

	// After cleanup and expiry, should not be duplicate
	if f.IsDuplicate(result) {
		t.Fatal("after Cleanup and expiry, decision should not be duplicate")
	}
}

// TestDedupConcurrency verifies thread safety of the dedup filter.
func TestDedupConcurrency(t *testing.T) {
	f := NewDedupFilter(100 * time.Millisecond)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			result := &DecisionResult{
				Decision:    DecisionStopRequestRequired,
				Rule:        "R-01",
				ZoneID:      "zone-1",
				EquipmentID: "eq-1",
			}
			f.IsDuplicate(result)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestDedupInEvaluator verifies R-05 integration in the full evaluator.
func TestDedupInEvaluator(t *testing.T) {
	// Use a longer suppression window for this test
	e := NewEvaluator(5 * time.Second)

	ctx := makeSimpleCtx("zone-1", events.OccupancyOccupied, "eq-1", events.EquipmentRunning, false, false)

	// First evaluation: should produce STOP_REQUEST_REQUIRED
	results1 := e.Evaluate(ctx)
	if !containsDecision(results1, DecisionStopRequestRequired) {
		t.Fatalf("first evaluation should produce STOP_REQUEST_REQUIRED, got: %+v", results1)
	}

	// Second evaluation with same context: should be suppressed (empty or reduced)
	results2 := e.Evaluate(ctx)
	if containsDecision(results2, DecisionStopRequestRequired) {
		t.Fatalf("second evaluation should suppress duplicate STOP_REQUEST_REQUIRED, got: %+v", results2)
	}
}
