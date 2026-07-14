package equipment

import (
	"testing"
	"time"

	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

func TestEquipmentState_InitialState_IsUnknown(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	if es.GetState() != events.EquipmentUnknown {
		t.Fatalf("initial state should be UNKNOWN, got %s", es.GetState())
	}
	if es.GetQuality() != ioAdapter.DIQualityBad {
		t.Fatalf("initial quality should be BAD, got %s", es.GetQuality())
	}
}

func TestEquipmentState_Update_Running(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)

	es.Update(diStates)

	if es.GetState() != events.EquipmentRunning {
		t.Fatalf("expected RUNNING, got %s", es.GetState())
	}
	if es.GetQuality() != ioAdapter.DIQualityGood {
		t.Fatalf("expected GOOD quality, got %s", es.GetQuality())
	}
}

func TestEquipmentState_Update_Stopped(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, false, false)

	es.Update(diStates)

	if es.GetState() != events.EquipmentStopped {
		t.Fatalf("expected STOPPED, got %s", es.GetState())
	}
}

func TestEquipmentState_Update_RestartRequested(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, false, true)

	es.Update(diStates)

	if es.GetState() != events.EquipmentRestartRequested {
		t.Fatalf("expected RESTART_REQUESTED, got %s", es.GetState())
	}
}

func TestEquipmentState_Update_RestartTakesPriority(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	// Both running and restart signals active
	diStates := makeDIStates(now, true, true)

	es.Update(diStates)

	if es.GetState() != events.EquipmentRestartRequested {
		t.Fatalf("RESTART_REQUESTED should take priority over RUNNING, got %s", es.GetState())
	}
}

func TestEquipmentState_Update_BadQuality_ProducesUnknown(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	// First set to running
	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	es.Update(diStates)

	if es.GetState() != events.EquipmentRunning {
		t.Fatalf("expected RUNNING first, got %s", es.GetState())
	}

	// Now provide bad quality data: I/O failure must not be treated as normal
	badStates := makeDIStatesWithQuality(now, true, false, ioAdapter.DIQualityBad)
	es.Update(badStates)

	if es.GetState() != events.EquipmentUnknown {
		t.Fatalf("bad quality DI must produce UNKNOWN, got %s", es.GetState())
	}
}

func TestEquipmentState_Update_StaleQuality_ProducesUnknown(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	staleStates := makeDIStatesWithQuality(now, true, false, ioAdapter.DIQualityStale)
	es.Update(staleStates)

	if es.GetState() != events.EquipmentUnknown {
		t.Fatalf("stale quality DI must produce UNKNOWN, got %s", es.GetState())
	}
}

func TestEquipmentState_Update_EmptyDI_ProducesUnknown(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	es.Update(nil)
	if es.GetState() != events.EquipmentUnknown {
		t.Fatalf("empty DI must produce UNKNOWN, got %s", es.GetState())
	}

	es.Update([]ioAdapter.DIState{})
	if es.GetState() != events.EquipmentUnknown {
		t.Fatalf("empty DI must produce UNKNOWN, got %s", es.GetState())
	}
}

func TestEquipmentState_IsStale_NeverUpdated(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	if !es.IsStale(time.Now()) {
		t.Fatal("equipment with no update should be stale")
	}
}

func TestEquipmentState_IsStale_RecentUpdate(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 5 * time.Second
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	es.Update(diStates)

	// Check immediately after update
	if es.IsStale(now.Add(1 * time.Second)) {
		t.Fatal("should not be stale 1s after update with 5s threshold")
	}
}

func TestEquipmentState_IsStale_OldUpdate(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 5 * time.Second
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	es.Update(diStates)

	// Check 10 seconds later
	if !es.IsStale(now.Add(10 * time.Second)) {
		t.Fatal("should be stale 10s after update with 5s threshold")
	}
}

func TestEquipmentState_MarkStale(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	// First set to running
	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	es.Update(diStates)

	if es.GetState() != events.EquipmentRunning {
		t.Fatalf("expected RUNNING, got %s", es.GetState())
	}

	// Mark stale: this MUST produce UNKNOWN
	es.MarkStale()

	if es.GetState() != events.EquipmentUnknown {
		t.Fatalf("stale DI input must produce UNKNOWN, got %s", es.GetState())
	}
	if es.GetQuality() != ioAdapter.DIQualityStale {
		t.Fatalf("expected STALE quality after MarkStale, got %s", es.GetQuality())
	}
}

func TestEquipmentState_GetSnapshot_Stale(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 2 * time.Second
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	es.Update(diStates)

	// Snapshot taken well within staleness
	snap := es.GetSnapshot(now.Add(1 * time.Second))
	if snap.State != events.EquipmentRunning {
		t.Fatalf("expected RUNNING in recent snapshot, got %s", snap.State)
	}
	if snap.IsStale {
		t.Fatal("expected not stale in recent snapshot")
	}

	// Snapshot taken after staleness threshold
	snap = es.GetSnapshot(now.Add(5 * time.Second))
	if snap.State != events.EquipmentUnknown {
		t.Fatalf("stale snapshot must show UNKNOWN, got %s", snap.State)
	}
	if !snap.IsStale {
		t.Fatal("expected stale flag in old snapshot")
	}
	if snap.Quality != ioAdapter.DIQualityStale {
		t.Fatalf("expected STALE quality in old snapshot, got %s", snap.Quality)
	}
}

func TestEquipmentState_StateTransition_Running_To_Stopped(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()

	// Start running
	es.Update(makeDIStates(now, true, false))
	if es.GetState() != events.EquipmentRunning {
		t.Fatalf("expected RUNNING, got %s", es.GetState())
	}

	// Transition to stopped
	es.Update(makeDIStates(now.Add(1*time.Second), false, false))
	if es.GetState() != events.EquipmentStopped {
		t.Fatalf("expected STOPPED, got %s", es.GetState())
	}
}

func TestEquipmentState_StateTransition_Running_To_Unknown_Via_Stale(t *testing.T) {
	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 2 * time.Second
	es := NewEquipmentState(cfg)

	now := time.Now().UTC()

	// Start running
	es.Update(makeDIStates(now, true, false))
	if es.GetState() != events.EquipmentRunning {
		t.Fatalf("expected RUNNING, got %s", es.GetState())
	}

	// No more updates - check staleness
	later := now.Add(5 * time.Second)
	if !es.IsStale(later) {
		t.Fatal("should be stale after 5s with 2s threshold")
	}

	// GetSnapshot enforces UNKNOWN for stale states
	snap := es.GetSnapshot(later)
	if snap.State != events.EquipmentUnknown {
		t.Fatalf("stale DI input must produce UNKNOWN state, got %s", snap.State)
	}
}

// --- Test helpers ---

func makeDIStates(timestamp time.Time, running bool, restart bool) []ioAdapter.DIState {
	states := make([]ioAdapter.DIState, ioAdapter.NumDIPoints)
	for i := range states {
		states[i] = ioAdapter.DIState{
			Address:    i,
			Value:      false,
			Quality:    ioAdapter.DIQualityGood,
			LastUpdate: timestamp,
		}
	}
	states[0].Value = running
	states[1].Value = restart
	return states
}

func makeDIStatesWithQuality(timestamp time.Time, running bool, restart bool, quality ioAdapter.DIQuality) []ioAdapter.DIState {
	states := make([]ioAdapter.DIState, ioAdapter.NumDIPoints)
	for i := range states {
		states[i] = ioAdapter.DIState{
			Address:    i,
			Value:      false,
			Quality:    quality,
			LastUpdate: timestamp,
		}
	}
	states[0].Value = running
	states[1].Value = restart
	return states
}
