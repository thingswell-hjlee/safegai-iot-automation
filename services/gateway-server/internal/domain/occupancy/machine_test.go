package occupancy

import (
	"testing"
	"time"
)

// R3 SAFETY CRITICAL: Comprehensive truth-table tests for the occupancy state machine.
// These tests verify all valid state transitions and the fail-safe behavior
// that prevents false vacancy detection.

func TestInitialStateIsUnknown(t *testing.T) {
	// R3 SAFETY: New state machines MUST start in UNKNOWN state.
	// A zone without detection data cannot be assumed vacant.
	sm := NewStateMachine("zone-1", DefaultConfig())
	state := sm.GetState()

	if state.CurrentState != StateUnknown {
		t.Errorf("initial state: got %q, want %q", state.CurrentState, StateUnknown)
	}
	if state.ZoneID != "zone-1" {
		t.Errorf("zone ID: got %q, want %q", state.ZoneID, "zone-1")
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: initial UNKNOWN state must NOT be safe vacant")
	}
}

func TestUnknownToOccupied(t *testing.T) {
	// Valid transition: UNKNOWN -> OCCUPIED when person is detected.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	event := CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	}

	transition, err := sm.ProcessEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateUnknown {
		t.Errorf("from: got %q, want %q", transition.From, StateUnknown)
	}
	if transition.To != StateOccupied {
		t.Errorf("to: got %q, want %q", transition.To, StateOccupied)
	}

	state := sm.GetState()
	if state.CurrentState != StateOccupied {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateOccupied)
	}
	if state.PersonCount != 1 {
		t.Errorf("person count: got %d, want %d", state.PersonCount, 1)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: OCCUPIED state must NOT be safe vacant")
	}
}

func TestUnknownToVacantPending(t *testing.T) {
	// Valid transition: UNKNOWN -> VACANT_PENDING when no person detected.
	// Note: This does NOT go directly to VACANT_CONFIRMED.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	event := CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now,
		CameraID:  "cam-1",
	}

	transition, err := sm.ProcessEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateUnknown {
		t.Errorf("from: got %q, want %q", transition.From, StateUnknown)
	}
	if transition.To != StateVacantPending {
		t.Errorf("to: got %q, want %q", transition.To, StateVacantPending)
	}

	state := sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateVacantPending)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: VACANT_PENDING must NOT be safe vacant")
	}
}

func TestOccupiedToVacantPending(t *testing.T) {
	// Valid transition: OCCUPIED -> VACANT_PENDING when person leaves.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	// First: person enters.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// Then: person leaves.
	event := CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(1 * time.Second),
		CameraID:  "cam-1",
	}

	transition, err := sm.ProcessEvent(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateOccupied {
		t.Errorf("from: got %q, want %q", transition.From, StateOccupied)
	}
	if transition.To != StateVacantPending {
		t.Errorf("to: got %q, want %q", transition.To, StateVacantPending)
	}

	state := sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateVacantPending)
	}
	if state.ConsecutiveVacantSamples != 1 {
		t.Errorf("consecutive vacant samples: got %d, want %d", state.ConsecutiveVacantSamples, 1)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: VACANT_PENDING must NOT be safe vacant")
	}
}

func TestVacantPendingToConfirmed(t *testing.T) {
	// Valid transition: VACANT_PENDING -> VACANT_CONFIRMED after 3 consecutive
	// no-person samples within 3s (meeting BOTH criteria).
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Transition to OCCUPIED first, then to VACANT_PENDING.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// First NO_PERSON: enters VACANT_PENDING.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(1 * time.Second),
		CameraID:  "cam-1",
	})

	state := sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Fatalf("expected VACANT_PENDING, got %q", state.CurrentState)
	}

	// Second NO_PERSON: still pending (only 2 samples, need 3).
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(2 * time.Second),
		CameraID:  "cam-1",
	})

	state = sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Fatalf("expected VACANT_PENDING after 2 samples, got %q", state.CurrentState)
	}

	// Third NO_PERSON at t=4s: meets BOTH criteria (3 samples + 3s elapsed).
	transition, err := sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(4 * time.Second),
		CameraID:  "cam-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.To != StateVacantConfirmed {
		t.Errorf("to: got %q, want %q", transition.To, StateVacantConfirmed)
	}

	state = sm.GetState()
	if state.CurrentState != StateVacantConfirmed {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateVacantConfirmed)
	}
	if !state.IsSafeVacant() {
		t.Error("R3 SAFETY: VACANT_CONFIRMED must be safe vacant")
	}
}

func TestVacantPendingToOccupied(t *testing.T) {
	// Valid transition: VACANT_PENDING -> OCCUPIED when person returns.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	// Enter OCCUPIED.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// Enter VACANT_PENDING.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(1 * time.Second),
		CameraID:  "cam-1",
	})

	state := sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Fatalf("expected VACANT_PENDING, got %q", state.CurrentState)
	}

	// Person returns during pending period.
	transition, err := sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now.Add(2 * time.Second),
		CameraID:    "cam-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateVacantPending {
		t.Errorf("from: got %q, want %q", transition.From, StateVacantPending)
	}
	if transition.To != StateOccupied {
		t.Errorf("to: got %q, want %q", transition.To, StateOccupied)
	}

	state = sm.GetState()
	if state.CurrentState != StateOccupied {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateOccupied)
	}
	if state.ConsecutiveVacantSamples != 0 {
		t.Errorf("consecutive vacant samples should be reset: got %d", state.ConsecutiveVacantSamples)
	}
}

func TestAnyToStale(t *testing.T) {
	// Valid transition: ANY -> STALE when no event within timeout.
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Enter OCCUPIED state.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	state := sm.GetState()
	if state.CurrentState != StateOccupied {
		t.Fatalf("expected OCCUPIED, got %q", state.CurrentState)
	}

	// Check staleness after timeout.
	staleTime := now.Add(cfg.StaleTimeout + 1*time.Second)
	transition, stale := sm.CheckStaleness(staleTime)
	if !stale {
		t.Fatal("expected zone to go stale")
	}

	if transition.From != StateOccupied {
		t.Errorf("from: got %q, want %q", transition.From, StateOccupied)
	}
	if transition.To != StateStale {
		t.Errorf("to: got %q, want %q", transition.To, StateStale)
	}

	state = sm.GetState()
	if state.CurrentState != StateStale {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateStale)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: STALE state must NOT be safe vacant")
	}
}

func TestStaleToOccupied(t *testing.T) {
	// Valid transition: STALE -> OCCUPIED when fresh person detection arrives.
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Enter OCCUPIED, then go STALE.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	staleTime := now.Add(cfg.StaleTimeout + 1*time.Second)
	sm.CheckStaleness(staleTime)

	state := sm.GetState()
	if state.CurrentState != StateStale {
		t.Fatalf("expected STALE, got %q", state.CurrentState)
	}

	// Fresh person detection from STALE.
	transition, err := sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 2,
		Timestamp:   staleTime.Add(1 * time.Second),
		CameraID:    "cam-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateStale {
		t.Errorf("from: got %q, want %q", transition.From, StateStale)
	}
	if transition.To != StateOccupied {
		t.Errorf("to: got %q, want %q", transition.To, StateOccupied)
	}

	state = sm.GetState()
	if state.CurrentState != StateOccupied {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateOccupied)
	}
	if state.PersonCount != 2 {
		t.Errorf("person count: got %d, want %d", state.PersonCount, 2)
	}
}

func TestStaleToVacantPending(t *testing.T) {
	// Valid transition: STALE -> VACANT_PENDING when fresh no-person data arrives.
	// Note: Does NOT go directly to VACANT_CONFIRMED.
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Enter OCCUPIED, then go STALE.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	staleTime := now.Add(cfg.StaleTimeout + 1*time.Second)
	sm.CheckStaleness(staleTime)

	state := sm.GetState()
	if state.CurrentState != StateStale {
		t.Fatalf("expected STALE, got %q", state.CurrentState)
	}

	// Fresh no-person from STALE.
	transition, err := sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: staleTime.Add(1 * time.Second),
		CameraID:  "cam-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateStale {
		t.Errorf("from: got %q, want %q", transition.From, StateStale)
	}
	if transition.To != StateVacantPending {
		t.Errorf("to: got %q, want %q", transition.To, StateVacantPending)
	}

	state = sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateVacantPending)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: VACANT_PENDING from STALE must NOT be safe vacant")
	}
}

func TestVacantConfirmedToOccupied(t *testing.T) {
	// Valid transition: VACANT_CONFIRMED -> OCCUPIED when person enters.
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Drive to VACANT_CONFIRMED state.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// Enter VACANT_PENDING.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(1 * time.Second),
		CameraID:  "cam-1",
	})

	// Second no-person.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(2 * time.Second),
		CameraID:  "cam-1",
	})

	// Third no-person at t=4s: confirms vacancy.
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

	// Person enters the confirmed-vacant zone.
	transition, err := sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now.Add(5 * time.Second),
		CameraID:    "cam-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateVacantConfirmed {
		t.Errorf("from: got %q, want %q", transition.From, StateVacantConfirmed)
	}
	if transition.To != StateOccupied {
		t.Errorf("to: got %q, want %q", transition.To, StateOccupied)
	}

	state = sm.GetState()
	if state.CurrentState != StateOccupied {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateOccupied)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: OCCUPIED must NOT be safe vacant")
	}
}

func TestIsSafeVacantOnlyForConfirmed(t *testing.T) {
	// R3 SAFETY CRITICAL: IsSafeVacant() must return true ONLY for VACANT_CONFIRMED.
	// All other states must return false.
	tests := []struct {
		state OccupancyState
		want  bool
	}{
		{StateUnknown, false},
		{StateOccupied, false},
		{StateVacantPending, false},
		{StateVacantConfirmed, true},
		{StateStale, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			zs := &ZoneState{
				ZoneID:       "zone-test",
				CurrentState: tt.state,
			}
			got := zs.IsSafeVacant()
			if got != tt.want {
				if tt.want {
					t.Errorf("IsSafeVacant() for %s: got %v, want %v", tt.state, got, tt.want)
				} else {
					t.Errorf("R3 SAFETY VIOLATION: IsSafeVacant() for %s: got %v, want %v", tt.state, got, tt.want)
				}
			}
		})
	}
}

func TestVacancyRequiresBothTimeAndSamples(t *testing.T) {
	// R3 SAFETY: Vacancy confirmation requires BOTH time elapsed AND sample count.
	// Having enough samples but not enough time must NOT confirm vacancy.
	cfg := Config{
		VacancyConfirmDuration: 5 * time.Second, // Need 5s
		VacancyConfirmSamples:  3,               // Need 3 samples
		StaleTimeout:           10 * time.Second,
		CameraOfflineTimeout:   30 * time.Second,
		DedupWindow:            2 * time.Second,
	}
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Enter OCCUPIED.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// First NO_PERSON at t=1s.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(1 * time.Second),
		CameraID:  "cam-1",
	})

	// Second NO_PERSON at t=2s.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(2 * time.Second),
		CameraID:  "cam-1",
	})

	// Third NO_PERSON at t=3s: 3 samples but only 2s elapsed (need 5s).
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(3 * time.Second),
		CameraID:  "cam-1",
	})

	state := sm.GetState()
	if state.CurrentState == StateVacantConfirmed {
		t.Error("R3 SAFETY VIOLATION: must NOT confirm vacancy when time requirement not met")
	}
	if state.CurrentState != StateVacantPending {
		t.Errorf("expected VACANT_PENDING, got %q", state.CurrentState)
	}

	// Fourth NO_PERSON at t=7s: now 4 samples AND >5s elapsed.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(7 * time.Second),
		CameraID:  "cam-1",
	})

	state = sm.GetState()
	if state.CurrentState != StateVacantConfirmed {
		t.Errorf("expected VACANT_CONFIRMED when both criteria met, got %q", state.CurrentState)
	}
}

func TestWrongZoneIDReturnsError(t *testing.T) {
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	_, err := sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-other",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	if err == nil {
		t.Fatal("expected error for wrong zone ID")
	}
}

func TestCameraOnlineDoesNotChangeState(t *testing.T) {
	// CAMERA_ONLINE is informational - does not change state.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	transition, err := sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventCameraOnline,
		Timestamp: now,
		CameraID:  "cam-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transition.From != StateUnknown {
		t.Errorf("from: got %q, want %q", transition.From, StateUnknown)
	}
	if transition.To != StateUnknown {
		t.Errorf("to: got %q, want %q (camera online should not change state)", transition.To, StateUnknown)
	}

	state := sm.GetState()
	if state.CurrentState != StateUnknown {
		t.Errorf("current state: got %q, want %q", state.CurrentState, StateUnknown)
	}
}

func TestAlreadyStaleDoesNotRetransition(t *testing.T) {
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Go stale.
	staleTime := now.Add(cfg.StaleTimeout + 1*time.Second)
	sm.CheckStaleness(staleTime)

	state := sm.GetState()
	if state.CurrentState != StateStale {
		t.Fatalf("expected STALE, got %q", state.CurrentState)
	}

	// Check staleness again - should not transition again.
	_, stale := sm.CheckStaleness(staleTime.Add(5 * time.Second))
	if stale {
		t.Error("already stale zone should not produce a new transition")
	}
}

func TestConsecutiveSamplesResetOnPersonDetection(t *testing.T) {
	// If person returns during VACANT_PENDING, consecutive samples must reset.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	// Enter OCCUPIED.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now,
		CameraID:    "cam-1",
	})

	// Enter VACANT_PENDING with 2 samples.
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

	state := sm.GetState()
	if state.ConsecutiveVacantSamples != 2 {
		t.Fatalf("expected 2 samples, got %d", state.ConsecutiveVacantSamples)
	}

	// Person returns - resets samples.
	sm.ProcessEvent(CameraEvent{
		ZoneID:      "zone-1",
		Type:        EventPersonDetected,
		PersonCount: 1,
		Timestamp:   now.Add(2500 * time.Millisecond),
		CameraID:    "cam-1",
	})

	state = sm.GetState()
	if state.ConsecutiveVacantSamples != 0 {
		t.Errorf("samples should be 0 after person returns, got %d", state.ConsecutiveVacantSamples)
	}

	// New VACANT_PENDING starts from scratch.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(3 * time.Second),
		CameraID:  "cam-1",
	})

	state = sm.GetState()
	if state.ConsecutiveVacantSamples != 1 {
		t.Errorf("expected fresh start with 1 sample, got %d", state.ConsecutiveVacantSamples)
	}
}
