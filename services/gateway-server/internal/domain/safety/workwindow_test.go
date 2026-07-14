package safety

// Tests for WorkWindow management (supporting Rule R-04).
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)

import (
	"testing"
	"time"
)

// TestWorkWindowStart verifies creating a new work window.
func TestWorkWindowStart(t *testing.T) {
	m := NewWorkWindowManager()
	w := m.Start("ww-1", "zone-1", "operator-A", 1*time.Hour)

	if w.ID != "ww-1" {
		t.Fatalf("expected ID ww-1, got %s", w.ID)
	}
	if w.ZoneID != "zone-1" {
		t.Fatalf("expected ZoneID zone-1, got %s", w.ZoneID)
	}
	if w.RequestedBy != "operator-A" {
		t.Fatalf("expected RequestedBy operator-A, got %s", w.RequestedBy)
	}
	if w.Status != WorkWindowStatusActive {
		t.Fatalf("expected status ACTIVE, got %s", w.Status)
	}
	if w.StartedAt.IsZero() {
		t.Fatal("StartedAt should not be zero")
	}
	if w.ExpiresAt.Before(w.StartedAt) {
		t.Fatal("ExpiresAt should be after StartedAt")
	}
}

// TestWorkWindowIsActive verifies IsActive returns true for active windows.
func TestWorkWindowIsActive(t *testing.T) {
	m := NewWorkWindowManager()
	m.Start("ww-1", "zone-1", "operator-A", 1*time.Hour)

	if !m.IsActive("zone-1") {
		t.Fatal("zone-1 should have active work window")
	}
	if m.IsActive("zone-2") {
		t.Fatal("zone-2 should not have active work window")
	}
}

// TestWorkWindowClose verifies closing a work window.
func TestWorkWindowClose(t *testing.T) {
	m := NewWorkWindowManager()
	m.Start("ww-1", "zone-1", "operator-A", 1*time.Hour)

	ok := m.Close("ww-1")
	if !ok {
		t.Fatal("Close should return true for existing window")
	}
	if m.IsActive("zone-1") {
		t.Fatal("zone-1 should not be active after Close")
	}
}

// TestWorkWindowCloseNonExistent verifies closing a non-existent window.
func TestWorkWindowCloseNonExistent(t *testing.T) {
	m := NewWorkWindowManager()
	ok := m.Close("non-existent")
	if ok {
		t.Fatal("Close should return false for non-existent window")
	}
}

// TestWorkWindowExpiration verifies windows expire after duration.
func TestWorkWindowExpiration(t *testing.T) {
	m := NewWorkWindowManager()
	m.Start("ww-1", "zone-1", "operator-A", 10*time.Millisecond)

	// Should be active initially
	if !m.IsActive("zone-1") {
		t.Fatal("zone-1 should be active initially")
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// Should no longer be active
	if m.IsActive("zone-1") {
		t.Fatal("zone-1 should not be active after expiration")
	}
}

// TestWorkWindowGetActive verifies GetActive returns only active windows.
func TestWorkWindowGetActive(t *testing.T) {
	m := NewWorkWindowManager()
	m.Start("ww-1", "zone-1", "operator-A", 1*time.Hour)
	m.Start("ww-2", "zone-1", "operator-B", 1*time.Hour)
	m.Start("ww-3", "zone-2", "operator-C", 1*time.Hour)

	active := m.GetActive("zone-1")
	if len(active) != 2 {
		t.Fatalf("expected 2 active windows for zone-1, got %d", len(active))
	}

	active2 := m.GetActive("zone-2")
	if len(active2) != 1 {
		t.Fatalf("expected 1 active window for zone-2, got %d", len(active2))
	}

	active3 := m.GetActive("zone-3")
	if len(active3) != 0 {
		t.Fatalf("expected 0 active windows for zone-3, got %d", len(active3))
	}
}

// TestWorkWindowGetActiveZones verifies GetActiveZones returns correct zone set.
func TestWorkWindowGetActiveZones(t *testing.T) {
	m := NewWorkWindowManager()
	m.Start("ww-1", "zone-1", "operator-A", 1*time.Hour)
	m.Start("ww-2", "zone-2", "operator-B", 1*time.Hour)

	zones := m.GetActiveZones()
	if !zones["zone-1"] {
		t.Fatal("zone-1 should be in active zones")
	}
	if !zones["zone-2"] {
		t.Fatal("zone-2 should be in active zones")
	}
	if zones["zone-3"] {
		t.Fatal("zone-3 should not be in active zones")
	}
}

// TestWorkWindowIsActiveMethod verifies the WorkWindow.IsActive method.
func TestWorkWindowIsActiveMethod(t *testing.T) {
	now := time.Now()
	w := &WorkWindow{
		ID:        "ww-1",
		ZoneID:    "zone-1",
		StartedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
		Status:    WorkWindowStatusActive,
	}

	if !w.IsActive(now) {
		t.Fatal("window should be active at start time")
	}
	if !w.IsActive(now.Add(30 * time.Minute)) {
		t.Fatal("window should be active within duration")
	}
	if w.IsActive(now.Add(2 * time.Hour)) {
		t.Fatal("window should not be active after expiration")
	}

	// Closed window
	w.Status = WorkWindowStatusClosed
	if w.IsActive(now) {
		t.Fatal("closed window should not be active")
	}
}

// TestWorkWindowMultiplePerZone verifies multiple windows can exist for one zone.
func TestWorkWindowMultiplePerZone(t *testing.T) {
	m := NewWorkWindowManager()
	m.Start("ww-1", "zone-1", "op-A", 1*time.Hour)
	m.Start("ww-2", "zone-1", "op-B", 1*time.Hour)

	// Close first one
	m.Close("ww-1")

	// Should still be active (second window)
	if !m.IsActive("zone-1") {
		t.Fatal("zone-1 should still be active (second window exists)")
	}

	// Close second one
	m.Close("ww-2")

	// Now should not be active
	if m.IsActive("zone-1") {
		t.Fatal("zone-1 should not be active (all windows closed)")
	}
}

// TestWorkWindowConcurrency verifies thread safety of WorkWindowManager.
func TestWorkWindowConcurrency(t *testing.T) {
	m := NewWorkWindowManager()
	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func(id int) {
			m.Start("ww-"+string(rune('a'+id)), "zone-1", "op", 1*time.Hour)
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		go func() {
			m.IsActive("zone-1")
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}
