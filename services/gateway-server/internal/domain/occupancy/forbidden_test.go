package occupancy

import (
	"testing"
	"time"
)

// R3 SAFETY CRITICAL: Forbidden transition tests.
// These tests explicitly verify that safety-violating transitions are REJECTED.
// Every test in this file MUST pass for the system to be considered safe.
//
// Forbidden transitions:
//   - Data timeout -> VACANT_CONFIRMED (must go to STALE instead)
//   - Camera offline -> VACANT_CONFIRMED (must go to UNKNOWN instead)
//   - Count parse failure -> VACANT_CONFIRMED (must go to UNKNOWN instead)
//   - Error event -> VACANT_CONFIRMED (must go to UNKNOWN instead)

func TestCameraOfflineCannotProduceVacant(t *testing.T) {
	// R3 SAFETY CRITICAL: Camera offline from ANY state MUST NOT produce VACANT_CONFIRMED.
	// Camera offline always produces UNKNOWN.
	states := []struct {
		name  string
		setup func(sm *StateMachine, now time.Time)
	}{
		{
			name:  "from_UNKNOWN",
			setup: func(sm *StateMachine, now time.Time) {},
		},
		{
			name: "from_OCCUPIED",
			setup: func(sm *StateMachine, now time.Time) {
				sm.ProcessEvent(CameraEvent{
					ZoneID:      "zone-1",
					Type:        EventPersonDetected,
					PersonCount: 1,
					Timestamp:   now,
					CameraID:    "cam-1",
				})
			},
		},
		{
			name: "from_VACANT_PENDING",
			setup: func(sm *StateMachine, now time.Time) {
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
			},
		},
		{
			name: "from_VACANT_CONFIRMED",
			setup: func(sm *StateMachine, now time.Time) {
				// Drive to VACANT_CONFIRMED.
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
			},
		},
		{
			name: "from_STALE",
			setup: func(sm *StateMachine, now time.Time) {
				sm.CheckStaleness(now.Add(11 * time.Second))
			},
		},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewStateMachine("zone-1", DefaultConfig())
			now := time.Now()
			tc.setup(sm, now)

			// Camera goes offline.
			transition, err := sm.ProcessEvent(CameraEvent{
				ZoneID:    "zone-1",
				Type:      EventCameraOffline,
				Timestamp: now.Add(5 * time.Second),
				CameraID:  "cam-1",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if transition.To == StateVacantConfirmed {
				t.Errorf("R3 SAFETY VIOLATION: camera offline produced VACANT_CONFIRMED from %s", tc.name)
			}

			state := sm.GetState()
			if state.IsSafeVacant() {
				t.Errorf("R3 SAFETY VIOLATION: IsSafeVacant() is true after camera offline from %s", tc.name)
			}

			// Verify state is UNKNOWN after camera offline.
			if state.CurrentState != StateUnknown {
				t.Errorf("camera offline should produce UNKNOWN, got %q from %s", state.CurrentState, tc.name)
			}
		})
	}
}

func TestStaleCannotProduceVacant(t *testing.T) {
	// R3 SAFETY CRITICAL: Stale timeout from ANY state MUST NOT produce VACANT_CONFIRMED.
	// Staleness always produces STALE.
	states := []struct {
		name  string
		setup func(sm *StateMachine, now time.Time)
	}{
		{
			name:  "from_UNKNOWN",
			setup: func(sm *StateMachine, now time.Time) {},
		},
		{
			name: "from_OCCUPIED",
			setup: func(sm *StateMachine, now time.Time) {
				sm.ProcessEvent(CameraEvent{
					ZoneID:      "zone-1",
					Type:        EventPersonDetected,
					PersonCount: 1,
					Timestamp:   now,
					CameraID:    "cam-1",
				})
			},
		},
		{
			name: "from_VACANT_PENDING",
			setup: func(sm *StateMachine, now time.Time) {
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
			},
		},
		{
			name: "from_VACANT_CONFIRMED",
			setup: func(sm *StateMachine, now time.Time) {
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
			},
		},
	}

	cfg := DefaultConfig()
	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewStateMachine("zone-1", cfg)
			now := time.Now()
			tc.setup(sm, now)

			// Trigger staleness: use a time far enough from last event.
			// The VACANT_CONFIRMED setup ends with an event at now+4s,
			// so we use now+15s to guarantee staleness from any setup.
			staleTime := now.Add(15 * time.Second)
			transition, stale := sm.CheckStaleness(staleTime)

			if stale {
				if transition.To == StateVacantConfirmed {
					t.Errorf("R3 SAFETY VIOLATION: staleness produced VACANT_CONFIRMED from %s", tc.name)
				}
			}

			state := sm.GetState()
			if state.IsSafeVacant() {
				t.Errorf("R3 SAFETY VIOLATION: IsSafeVacant() is true after staleness from %s", tc.name)
			}
		})
	}
}

func TestDataTimeoutCannotProduceVacant(t *testing.T) {
	// R3 SAFETY CRITICAL: Data timeout (staleness) must transition to STALE, not VACANT.
	// This is the explicit test that timeout -> VACANT_CONFIRMED is forbidden.
	cfg := DefaultConfig()
	sm := NewStateMachine("zone-1", cfg)
	now := time.Now()

	// Even if zone was in VACANT_PENDING with 2/3 samples,
	// a timeout must NOT complete the vacancy confirmation.
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

	state := sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Fatalf("expected VACANT_PENDING, got %q", state.CurrentState)
	}
	if state.ConsecutiveVacantSamples != 2 {
		t.Fatalf("expected 2 samples, got %d", state.ConsecutiveVacantSamples)
	}

	// Data timeout occurs - zone goes STALE, NOT VACANT_CONFIRMED.
	staleTime := now.Add(2*time.Second + cfg.StaleTimeout + 1*time.Second)
	transition, stale := sm.CheckStaleness(staleTime)

	if !stale {
		t.Fatal("expected staleness transition")
	}
	if transition.To == StateVacantConfirmed {
		t.Error("R3 SAFETY VIOLATION: data timeout produced VACANT_CONFIRMED")
	}
	if transition.To != StateStale {
		t.Errorf("data timeout should produce STALE, got %q", transition.To)
	}

	state = sm.GetState()
	if state.CurrentState != StateStale {
		t.Errorf("current state after timeout: got %q, want STALE", state.CurrentState)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: IsSafeVacant() true after data timeout")
	}
}

func TestErrorCannotProduceVacant(t *testing.T) {
	// R3 SAFETY CRITICAL: Error events from ANY state MUST NOT produce VACANT_CONFIRMED.
	// Parse failures, communication errors, etc. always produce UNKNOWN.
	states := []struct {
		name  string
		setup func(sm *StateMachine, now time.Time)
	}{
		{
			name:  "from_UNKNOWN",
			setup: func(sm *StateMachine, now time.Time) {},
		},
		{
			name: "from_OCCUPIED",
			setup: func(sm *StateMachine, now time.Time) {
				sm.ProcessEvent(CameraEvent{
					ZoneID:      "zone-1",
					Type:        EventPersonDetected,
					PersonCount: 1,
					Timestamp:   now,
					CameraID:    "cam-1",
				})
			},
		},
		{
			name: "from_VACANT_PENDING",
			setup: func(sm *StateMachine, now time.Time) {
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
			},
		},
		{
			name: "from_VACANT_CONFIRMED",
			setup: func(sm *StateMachine, now time.Time) {
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
			},
		},
		{
			name: "from_STALE",
			setup: func(sm *StateMachine, now time.Time) {
				sm.CheckStaleness(now.Add(11 * time.Second))
			},
		},
	}

	for _, tc := range states {
		t.Run(tc.name, func(t *testing.T) {
			sm := NewStateMachine("zone-1", DefaultConfig())
			now := time.Now()
			tc.setup(sm, now)

			// Send error event.
			transition, err := sm.ProcessEvent(CameraEvent{
				ZoneID:       "zone-1",
				Type:         EventError,
				Timestamp:    now.Add(5 * time.Second),
				CameraID:     "cam-1",
				ErrorMessage: "count parse failure: invalid integer",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if transition.To == StateVacantConfirmed {
				t.Errorf("R3 SAFETY VIOLATION: error event produced VACANT_CONFIRMED from %s", tc.name)
			}

			state := sm.GetState()
			if state.IsSafeVacant() {
				t.Errorf("R3 SAFETY VIOLATION: IsSafeVacant() true after error from %s", tc.name)
			}

			// Error should produce UNKNOWN.
			if state.CurrentState != StateUnknown {
				t.Errorf("error should produce UNKNOWN, got %q from %s", state.CurrentState, tc.name)
			}
		})
	}
}

func TestIsSafeVacantFalseForUnknown(t *testing.T) {
	// R3 SAFETY CRITICAL: UNKNOWN is NOT safe vacant.
	zs := &ZoneState{
		ZoneID:       "zone-1",
		CurrentState: StateUnknown,
	}
	if zs.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: UNKNOWN must NOT be safe vacant")
	}
}

func TestIsSafeVacantFalseForStale(t *testing.T) {
	// R3 SAFETY CRITICAL: STALE is NOT safe vacant.
	zs := &ZoneState{
		ZoneID:       "zone-1",
		CurrentState: StateStale,
	}
	if zs.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: STALE must NOT be safe vacant")
	}
}

func TestIsSafeVacantFalseForOccupied(t *testing.T) {
	// R3 SAFETY CRITICAL: OCCUPIED is NOT safe vacant.
	zs := &ZoneState{
		ZoneID:       "zone-1",
		CurrentState: StateOccupied,
	}
	if zs.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: OCCUPIED must NOT be safe vacant")
	}
}

func TestIsSafeVacantFalseForPending(t *testing.T) {
	// R3 SAFETY CRITICAL: VACANT_PENDING is NOT safe vacant.
	zs := &ZoneState{
		ZoneID:       "zone-1",
		CurrentState: StateVacantPending,
	}
	if zs.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: VACANT_PENDING must NOT be safe vacant")
	}
}

func TestMultipleCameraOfflineEventsRemainUnknown(t *testing.T) {
	// R3 SAFETY: Repeated camera offline events must keep state at UNKNOWN.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	for i := 0; i < 5; i++ {
		sm.ProcessEvent(CameraEvent{
			ZoneID:    "zone-1",
			Type:      EventCameraOffline,
			Timestamp: now.Add(time.Duration(i) * time.Second),
			CameraID:  "cam-1",
		})

		state := sm.GetState()
		if state.CurrentState != StateUnknown {
			t.Errorf("iteration %d: expected UNKNOWN, got %q", i, state.CurrentState)
		}
		if state.IsSafeVacant() {
			t.Errorf("R3 SAFETY VIOLATION: iteration %d: IsSafeVacant() true during offline", i)
		}
	}
}

func TestErrorResetsVacancyProgress(t *testing.T) {
	// R3 SAFETY: Error during VACANT_PENDING must reset vacancy progress.
	// After error resolution, vacancy confirmation must start from scratch.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	// Enter OCCUPIED, then VACANT_PENDING with 2 samples.
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

	state := sm.GetState()
	if state.ConsecutiveVacantSamples != 2 {
		t.Fatalf("expected 2 samples, got %d", state.ConsecutiveVacantSamples)
	}

	// Error occurs - resets to UNKNOWN.
	sm.ProcessEvent(CameraEvent{
		ZoneID:       "zone-1",
		Type:         EventError,
		Timestamp:    now.Add(3 * time.Second),
		CameraID:     "cam-1",
		ErrorMessage: "communication timeout",
	})

	state = sm.GetState()
	if state.CurrentState != StateUnknown {
		t.Errorf("expected UNKNOWN after error, got %q", state.CurrentState)
	}
	if state.ConsecutiveVacantSamples != 0 {
		t.Errorf("consecutive samples should be 0 after error, got %d", state.ConsecutiveVacantSamples)
	}

	// New NO_PERSON events must start from scratch.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventNoPerson,
		Timestamp: now.Add(4 * time.Second),
		CameraID:  "cam-1",
	})

	state = sm.GetState()
	if state.CurrentState != StateVacantPending {
		t.Errorf("expected VACANT_PENDING, got %q", state.CurrentState)
	}
	if state.ConsecutiveVacantSamples != 1 {
		t.Errorf("expected 1 sample (fresh start), got %d", state.ConsecutiveVacantSamples)
	}
}

func TestCameraOfflineResetsVacancyProgress(t *testing.T) {
	// R3 SAFETY: Camera offline during VACANT_PENDING must reset vacancy progress.
	sm := NewStateMachine("zone-1", DefaultConfig())
	now := time.Now()

	// Enter OCCUPIED, then VACANT_PENDING with 2 samples.
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

	state := sm.GetState()
	if state.ConsecutiveVacantSamples != 2 {
		t.Fatalf("expected 2 samples, got %d", state.ConsecutiveVacantSamples)
	}

	// Camera goes offline - resets to UNKNOWN.
	sm.ProcessEvent(CameraEvent{
		ZoneID:    "zone-1",
		Type:      EventCameraOffline,
		Timestamp: now.Add(3 * time.Second),
		CameraID:  "cam-1",
	})

	state = sm.GetState()
	if state.CurrentState != StateUnknown {
		t.Errorf("expected UNKNOWN after camera offline, got %q", state.CurrentState)
	}
	if state.ConsecutiveVacantSamples != 0 {
		t.Errorf("samples should be 0 after camera offline, got %d", state.ConsecutiveVacantSamples)
	}
	if state.IsSafeVacant() {
		t.Error("R3 SAFETY VIOLATION: IsSafeVacant() true after camera offline")
	}
}
