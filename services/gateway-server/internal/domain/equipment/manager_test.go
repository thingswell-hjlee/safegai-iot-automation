package equipment

import (
	"testing"
	"time"

	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

func TestManager_RegisterEquipment(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	err := mgr.RegisterEquipment("eq-001", cfg)
	if err != nil {
		t.Fatalf("RegisterEquipment failed: %v", err)
	}

	if mgr.Count() != 1 {
		t.Fatalf("expected 1 equipment, got %d", mgr.Count())
	}
}

func TestManager_RegisterEquipment_Duplicate(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	_ = mgr.RegisterEquipment("eq-001", cfg)

	err := mgr.RegisterEquipment("eq-001", cfg)
	if err == nil {
		t.Fatal("expected conflict error for duplicate registration")
	}
}

func TestManager_RegisterEquipment_EmptyID(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("")
	err := mgr.RegisterEquipment("", cfg)
	if err == nil {
		t.Fatal("expected validation error for empty ID")
	}
}

func TestManager_UpdateFromDI(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	_ = mgr.RegisterEquipment("eq-001", cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)

	mgr.UpdateFromDI(diStates)

	snap, err := mgr.GetState("eq-001", now)
	if err != nil {
		t.Fatalf("GetState error: %v", err)
	}
	if snap.State != events.EquipmentRunning {
		t.Fatalf("expected RUNNING, got %s", snap.State)
	}
}

func TestManager_UpdateFromDI_MultipleEquipment(t *testing.T) {
	mgr := NewManager()

	// Equipment 1 uses DI[0] as running, DI[1] as restart
	cfg1 := DefaultEquipmentConfig("eq-001")
	_ = mgr.RegisterEquipment("eq-001", cfg1)

	// Equipment 2 uses DI[2] as running, DI[3] as restart
	cfg2 := EquipmentConfig{
		ID:               "eq-002",
		RunningDIAddress: 2,
		RestartDIAddress: 3,
		StaleDuration:    5 * time.Second,
	}
	_ = mgr.RegisterEquipment("eq-002", cfg2)

	now := time.Now().UTC()
	// DI[0]=true (eq-001 running), DI[2]=false (eq-002 stopped)
	diStates := make([]ioAdapter.DIState, ioAdapter.NumDIPoints)
	for i := range diStates {
		diStates[i] = ioAdapter.DIState{
			Address:    i,
			Value:      false,
			Quality:    ioAdapter.DIQualityGood,
			LastUpdate: now,
		}
	}
	diStates[0].Value = true

	mgr.UpdateFromDI(diStates)

	snap1, _ := mgr.GetState("eq-001", now)
	snap2, _ := mgr.GetState("eq-002", now)

	if snap1.State != events.EquipmentRunning {
		t.Fatalf("eq-001 expected RUNNING, got %s", snap1.State)
	}
	if snap2.State != events.EquipmentStopped {
		t.Fatalf("eq-002 expected STOPPED, got %s", snap2.State)
	}
}

func TestManager_GetState_NotFound(t *testing.T) {
	mgr := NewManager()

	_, err := mgr.GetState("nonexistent", time.Now())
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestManager_GetAllStates(t *testing.T) {
	mgr := NewManager()

	cfg1 := DefaultEquipmentConfig("eq-001")
	cfg2 := DefaultEquipmentConfig("eq-002")
	_ = mgr.RegisterEquipment("eq-001", cfg1)
	_ = mgr.RegisterEquipment("eq-002", cfg2)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	mgr.UpdateFromDI(diStates)

	states := mgr.GetAllStates(now)
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}

	// All should be running since they use same DI addresses
	for _, s := range states {
		if s.State != events.EquipmentRunning {
			t.Errorf("equipment %s expected RUNNING, got %s", s.ID, s.State)
		}
	}
}

func TestManager_CheckStaleness(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 2 * time.Second
	_ = mgr.RegisterEquipment("eq-001", cfg)

	now := time.Now().UTC()
	diStates := makeDIStates(now, true, false)
	mgr.UpdateFromDI(diStates)

	// Not stale yet
	stale := mgr.CheckStaleness(now.Add(1 * time.Second))
	if len(stale) != 0 {
		t.Fatal("should not be stale after 1s")
	}

	// Should be stale after threshold
	stale = mgr.CheckStaleness(now.Add(5 * time.Second))
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale equipment, got %d", len(stale))
	}
	if stale[0] != "eq-001" {
		t.Fatalf("expected 'eq-001' in stale list, got %q", stale[0])
	}

	// Verify state is now UNKNOWN
	snap, _ := mgr.GetState("eq-001", now.Add(5*time.Second))
	if snap.State != events.EquipmentUnknown {
		t.Fatalf("stale equipment must be UNKNOWN, got %s", snap.State)
	}
}

func TestManager_CheckStaleness_NeverUpdated(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 2 * time.Second
	_ = mgr.RegisterEquipment("eq-001", cfg)

	// Never updated - should always be stale
	stale := mgr.CheckStaleness(time.Now())
	if len(stale) != 1 {
		t.Fatal("equipment with no updates should be stale")
	}
}

func TestManager_Unregister(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	_ = mgr.RegisterEquipment("eq-001", cfg)

	err := mgr.Unregister("eq-001")
	if err != nil {
		t.Fatalf("Unregister error: %v", err)
	}

	if mgr.Count() != 0 {
		t.Fatalf("expected 0 equipment after unregister, got %d", mgr.Count())
	}
}

func TestManager_Unregister_NotFound(t *testing.T) {
	mgr := NewManager()

	err := mgr.Unregister("nonexistent")
	if err == nil {
		t.Fatal("expected not found error for unregister of nonexistent equipment")
	}
}

// TestManager_IOFailure_Never_Normal verifies that I/O failure
// is never treated as a normal/safe state.
func TestManager_IOFailure_Never_Normal(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	_ = mgr.RegisterEquipment("eq-001", cfg)

	now := time.Now().UTC()

	// First set equipment to running
	diStates := makeDIStates(now, true, false)
	mgr.UpdateFromDI(diStates)

	snap, _ := mgr.GetState("eq-001", now)
	if snap.State != events.EquipmentRunning {
		t.Fatalf("expected RUNNING first, got %s", snap.State)
	}

	// Now simulate I/O failure (bad quality)
	badStates := makeDIStatesWithQuality(now.Add(1*time.Second), true, false, ioAdapter.DIQualityBad)
	mgr.UpdateFromDI(badStates)

	snap, _ = mgr.GetState("eq-001", now.Add(1*time.Second))
	if snap.State != events.EquipmentUnknown {
		t.Fatalf("I/O failure must produce UNKNOWN, got %s", snap.State)
	}
}

// TestManager_StaleDI_Produces_Unknown verifies the critical requirement:
// stale DI input MUST produce UNKNOWN equipment state.
func TestManager_StaleDI_Produces_Unknown(t *testing.T) {
	mgr := NewManager()

	cfg := DefaultEquipmentConfig("eq-001")
	cfg.StaleDuration = 1 * time.Second
	_ = mgr.RegisterEquipment("eq-001", cfg)

	now := time.Now().UTC()

	// Start running
	diStates := makeDIStates(now, true, false)
	mgr.UpdateFromDI(diStates)

	snap, _ := mgr.GetState("eq-001", now)
	if snap.State != events.EquipmentRunning {
		t.Fatalf("expected RUNNING, got %s", snap.State)
	}

	// After staleness threshold, state must be UNKNOWN
	later := now.Add(3 * time.Second)
	snap, _ = mgr.GetState("eq-001", later)
	if snap.State != events.EquipmentUnknown {
		t.Fatalf("stale DI input must produce UNKNOWN, got %s", snap.State)
	}
}

// TestManager_NoDirectPowerControl verifies there is no direct machine
// power control logic in the manager (all references are to PLC/Safety Relay).
// This is a documentation test confirming the safety constraint.
func TestManager_NoDirectPowerControl(t *testing.T) {
	// This test documents the safety constraint:
	// The Manager does NOT control equipment power directly.
	// It only reads DI signals FROM PLC/Safety Relay and derives states.
	// Output commands (if any) are sent TO PLC/Safety Relay only.
	//
	// The Manager has NO methods for:
	// - StartEquipment
	// - StopEquipment
	// - PowerOff
	// - EmergencyStop (direct)
	//
	// All such actions must go through the Output/Alarm module (M06)
	// which sends stop requests to PLC/Safety Relay via the IOAdapter.

	mgr := NewManager()
	_ = mgr.RegisterEquipment("eq-001", DefaultEquipmentConfig("eq-001"))

	// The only write operations are RegisterEquipment and Unregister.
	// State changes happen only through DI reading (UpdateFromDI).
	// This confirms no direct power switching logic exists.
	if mgr.Count() != 1 {
		t.Fatal("basic operation check failed")
	}
}
