package occupancy

import (
	"testing"
	"time"
)

// R3 SAFETY CRITICAL: Tests for the StaleChecker component.
// The staleness checker ensures zones without recent data transition to STALE.
// STALE does NOT satisfy vacancy and blocks equipment restart.

func TestStaleCheckerRegistration(t *testing.T) {
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)

	if sc.MachineCount() != 0 {
		t.Errorf("expected 0 machines, got %d", sc.MachineCount())
	}

	sm1 := NewStateMachine("zone-1", cfg)
	sm2 := NewStateMachine("zone-2", cfg)

	sc.Register(sm1)
	sc.Register(sm2)

	if sc.MachineCount() != 2 {
		t.Errorf("expected 2 machines, got %d", sc.MachineCount())
	}

	sc.Unregister("zone-1")
	if sc.MachineCount() != 1 {
		t.Errorf("expected 1 machine after unregister, got %d", sc.MachineCount())
	}
}

func TestStaleCheckerDetectsStaleZone(t *testing.T) {
	// R3 SAFETY: Zones without events beyond the timeout must become STALE.
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)
	now := time.Now()

	sm := NewStateMachine("zone-1", cfg)
	// Process an event so the zone has a known last event time.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	sc.Register(sm)

	// Check before timeout - should not transition.
	transitions := sc.Check(now.Add(5 * time.Second))
	if len(transitions) != 0 {
		t.Errorf("expected 0 transitions before timeout, got %d", len(transitions))
	}

	// Check after timeout - should transition to STALE.
	transitions = sc.Check(now.Add(cfg.StaleTimeout + 1*time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition after timeout, got %d", len(transitions))
	}

	if transitions[0].To != StateStale {
		t.Errorf("transition to: got %q, want %q", transitions[0].To, StateStale)
	}
	if transitions[0].ZoneID != "zone-1" {
		t.Errorf("zone ID: got %q, want %q", transitions[0].ZoneID, "zone-1")
	}

	state := sm.GetState()
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: STALE must NOT be safe vacant")
	}
}

func TestStaleCheckerMultipleZones(t *testing.T) {
	// R3 SAFETY: All zones must be checked for staleness independently.
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)
	now := time.Now()

	sm1 := NewStateMachine("zone-1", cfg)
	sm2 := NewStateMachine("zone-2", cfg)
	sm3 := NewStateMachine("zone-3", cfg)

	// zone-1: event at now (will go stale at now+10s).
	sm1.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// zone-2: event at now+5s (will go stale at now+15s).
	sm2.ProcessEvent(CameraEvent{
		ZoneID:      "zone-2",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now.Add(5 * time.Second),
		CameraID:    "cam-2",
	})

	// zone-3: event at now+8s (will go stale at now+18s).
	sm3.ProcessEvent(CameraEvent{
		ZoneID:    "zone-3",
		Type:      EventNoPerson,
		Timestamp: now.Add(8 * time.Second),
		CameraID:  "cam-3",
	})

	sc.Register(sm1)
	sc.Register(sm2)
	sc.Register(sm3)

	// At now+11s: only zone-1 should be stale.
	transitions := sc.Check(now.Add(11 * time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 stale zone at t+11s, got %d", len(transitions))
	}
	if transitions[0].ZoneID != "zone-1" {
		t.Errorf("expected zone-1 stale, got %q", transitions[0].ZoneID)
	}

	// At now+16s: zone-2 should also be stale.
	transitions = sc.Check(now.Add(16 * time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 new stale zone at t+16s, got %d", len(transitions))
	}
	if transitions[0].ZoneID != "zone-2" {
		t.Errorf("expected zone-2 stale, got %q", transitions[0].ZoneID)
	}

	// At now+19s: zone-3 should also be stale.
	transitions = sc.Check(now.Add(19 * time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 new stale zone at t+19s, got %d", len(transitions))
	}
	if transitions[0].ZoneID != "zone-3" {
		t.Errorf("expected zone-3 stale, got %q", transitions[0].ZoneID)
	}
}

func TestStaleCheckerAlreadyStaleNotReported(t *testing.T) {
	// Once a zone is STALE, it should not produce repeated transitions.
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)
	now := time.Now()

	sm := NewStateMachine("zone-1", cfg)
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	sc.Register(sm)

	// First check after timeout.
	transitions := sc.Check(now.Add(cfg.StaleTimeout + 1*time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}

	// Second check - already stale, no new transition.
	transitions = sc.Check(now.Add(cfg.StaleTimeout + 5*time.Second))
	if len(transitions) != 0 {
		t.Errorf("expected 0 transitions for already-stale zone, got %d", len(transitions))
	}
}

func TestStaleCheckerRecovery(t *testing.T) {
	// After a zone recovers from STALE (receives new event), it can go stale again.
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)
	now := time.Now()

	sm := NewStateMachine("zone-1", cfg)
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	sc.Register(sm)

	// Go stale.
	transitions := sc.Check(now.Add(cfg.StaleTimeout + 1*time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}

	state := sm.GetState()
	if state.CurrentState != StateStale {
		t.Fatalf("expected STALE, got %q", state.CurrentState)
	}

	// Zone recovers with new event.
	recoveryTime := now.Add(cfg.StaleTimeout + 2*time.Second)
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   recoveryTime,
		CameraID:    "cam-1",
	})

	state = sm.GetState()
	if state.CurrentState != StateOccupied {
		t.Fatalf("expected OCCUPIED after recovery, got %q", state.CurrentState)
	}

	// Check again before new timeout - should not be stale.
	transitions = sc.Check(recoveryTime.Add(5 * time.Second))
	if len(transitions) != 0 {
		t.Errorf("expected 0 transitions after recovery, got %d", len(transitions))
	}

	// Go stale again.
	transitions = sc.Check(recoveryTime.Add(cfg.StaleTimeout + 1*time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition on second staleness, got %d", len(transitions))
	}
}

func TestCheckZonesStateless(t *testing.T) {
	// R3 SAFETY: CheckZones provides stateless staleness detection.
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)
	now := time.Now()

	zones := []ZoneState{
		{
			ZoneID:       "zone-1",
			CurrentState: StateOccupied,
			LastEventAt:  now.Add(-15 * time.Second), // Stale (>10s ago).
		},
		{
			ZoneID:       "zone-2",
			CurrentState: StateVacantPending,
			LastEventAt:  now.Add(-5 * time.Second), // Not stale yet.
		},
		{
			ZoneID:       "zone-3",
			CurrentState: StateStale, // Already stale, should be skipped.
			LastEventAt:  now.Add(-20 * time.Second),
		},
		{
			ZoneID:       "zone-4",
			CurrentState: StateUnknown,
			LastEventAt:  now.Add(-12 * time.Second), // Stale (>10s ago).
		},
	}

	transitions := sc.CheckZones(zones, now)

	if len(transitions) != 2 {
		t.Fatalf("expected 2 stale transitions, got %d", len(transitions))
	}

	// Verify zone-1 and zone-4 are flagged.
	found := map[string]bool{}
	for _, tr := range transitions {
		found[tr.ZoneID] = true
		if tr.To != StateStale {
			t.Errorf("zone %s: expected STALE, got %q", tr.ZoneID, tr.To)
		}
	}

	if !found["zone-1"] {
		t.Error("zone-1 should be in stale transitions")
	}
	if !found["zone-4"] {
		t.Error("zone-4 should be in stale transitions")
	}
	if found["zone-2"] {
		t.Error("zone-2 should NOT be stale yet")
	}
	if found["zone-3"] {
		t.Error("zone-3 is already STALE, should be skipped")
	}
}

func TestStaleCheckerSafetyInvariant(t *testing.T) {
	// R3 SAFETY: After staleness, IsSafeVacant must be false.
	cfg := DefaultConfig()
	sc := NewStaleChecker(cfg)
	now := time.Now()

	sm := NewStateMachine("zone-1", cfg)
	// Drive to VACANT_CONFIRMED first.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(1 * time.Second),
		CameraID:  "cam-1",
	})
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(2 * time.Second),
		CameraID:  "cam-1",
	})
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(4 * time.Second),
		CameraID:  "cam-1",
	})

	state := sm.GetState()
	if state.CurrentState != StateVacantConfirmed {
		t.Fatalf("expected VACANT_CONFIRMED, got %q", state.CurrentState)
	}
	if !state.IsSafeVacant() {
		t.Fatal("VACANT_CONFIRMED should be safe vacant")
	}

	sc.Register(sm)

	// Now zone goes stale - should lose safe vacant status.
	transitions := sc.Check(now.Add(4*time.Second + cfg.StaleTimeout + 1*time.Second))
	if len(transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(transitions))
	}

	state = sm.GetState()
	if state.CurrentState != StateStale {
		t.Errorf("expected STALE, got %q", state.CurrentState)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: zone should NOT be safe vacant after going stale from VACANT_CONFIRMED")
	}
}
