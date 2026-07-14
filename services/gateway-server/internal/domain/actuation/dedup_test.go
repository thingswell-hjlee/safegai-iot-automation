package actuation

import (
	"testing"
	"time"
)

// TestFirstCommandPasses verifies that the first command is not a duplicate.
// The first occurrence of any correlationId+commandType passes through.
func TestFirstCommandPasses(t *testing.T) {
	dedup := NewDedupTracker(10 * time.Second)

	if dedup.IsDuplicate("event-001", CommandWarningLight) {
		t.Error("first command should not be a duplicate")
	}

	if dedup.IsDuplicate("event-001", CommandStopRequestPulse) {
		t.Error("different command type for same correlation should not be duplicate")
	}
}

// TestDuplicateWithinWindow verifies that the same correlation+type within
// the suppression window is blocked.
//
// SAFETY: Prevents repeated output to PLC/Safety Relay for the same event.
func TestDuplicateWithinWindow(t *testing.T) {
	dedup := NewDedupTracker(10 * time.Second)

	// Record the first execution.
	dedup.Record("event-001", CommandWarningLight)

	// Same correlation+type should be duplicate within window.
	if !dedup.IsDuplicate("event-001", CommandWarningLight) {
		t.Error("same correlation+type within window should be duplicate")
	}
}

// TestAfterWindowPasses verifies that the same correlation+type after the
// suppression window has elapsed passes through.
func TestAfterWindowPasses(t *testing.T) {
	// Use a very short suppression window for testing.
	dedup := NewDedupTracker(50 * time.Millisecond)

	// Record the first execution.
	dedup.Record("event-001", CommandWarningLight)

	// Immediately should be duplicate.
	if !dedup.IsDuplicate("event-001", CommandWarningLight) {
		t.Error("should be duplicate immediately after recording")
	}

	// Wait for window to expire.
	time.Sleep(60 * time.Millisecond)

	// After window, should pass through.
	if dedup.IsDuplicate("event-001", CommandWarningLight) {
		t.Error("should not be duplicate after window expires")
	}
}

// TestDifferentCorrelationPasses verifies that different events are independent.
// Each correlationId is tracked separately.
func TestDifferentCorrelationPasses(t *testing.T) {
	dedup := NewDedupTracker(10 * time.Second)

	// Record event-001.
	dedup.Record("event-001", CommandWarningLight)

	// Different correlation should pass.
	if dedup.IsDuplicate("event-002", CommandWarningLight) {
		t.Error("different correlationId should not be duplicate")
	}

	// Same correlation but different type should pass.
	if dedup.IsDuplicate("event-001", CommandSiren) {
		t.Error("same correlationId but different commandType should not be duplicate")
	}
}

// TestCleanup verifies that old entries are removed by Cleanup.
func TestCleanup(t *testing.T) {
	dedup := NewDedupTracker(1 * time.Hour)

	// Record some entries.
	dedup.Record("event-001", CommandWarningLight)
	dedup.Record("event-002", CommandSiren)

	// Both should be duplicates.
	if !dedup.IsDuplicate("event-001", CommandWarningLight) {
		t.Error("event-001 should be duplicate before cleanup")
	}
	if !dedup.IsDuplicate("event-002", CommandSiren) {
		t.Error("event-002 should be duplicate before cleanup")
	}

	// Cleanup entries older than now+1s (effectively all entries since they were recorded "now").
	dedup.Cleanup(time.Now().Add(1 * time.Second))

	// After cleanup, nothing should be duplicate.
	if dedup.IsDuplicate("event-001", CommandWarningLight) {
		t.Error("event-001 should not be duplicate after cleanup")
	}
	if dedup.IsDuplicate("event-002", CommandSiren) {
		t.Error("event-002 should not be duplicate after cleanup")
	}
}

// TestDedupConcurrency verifies thread safety of DedupTracker.
func TestDedupConcurrency(t *testing.T) {
	dedup := NewDedupTracker(10 * time.Second)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			dedup.Record("event-concurrent", CommandWarningLight)
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		dedup.IsDuplicate("event-concurrent", CommandWarningLight)
	}

	<-done
}
