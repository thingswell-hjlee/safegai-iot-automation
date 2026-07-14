package actuation

import (
	"testing"
	"time"
)

// TestOldCommandBlockedAfterRestart verifies that a command from before boot
// is rejected by the replay guard.
//
// SAFETY: After restart, past pulse commands are NOT replayed.
// A stop request from a previous boot cycle may no longer be valid.
// This prevents stale safety actions from being re-executed.
func TestOldCommandBlockedAfterRestart(t *testing.T) {
	rg := NewReplayGuard()

	// Mark boot at a known time.
	bootTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	rg.MarkBoot(bootTime)

	// Command created BEFORE boot.
	oldCmd := ActuationCommand{
		ID:            "cmd-old-001",
		CorrelationID: "event-old-001",
		CommandType:   CommandStopRequestPulse,
		TargetAddress: "plc-relay://zone1/stop-request-input",
		CreatedAt:     bootTime.Add(-1 * time.Hour), // 1 hour before boot
	}

	if rg.ShouldExecute(oldCmd) {
		t.Error("command from before boot should be REJECTED by replay guard")
	}

	// Even 1 nanosecond before boot should be rejected.
	barelyOld := ActuationCommand{
		ID:            "cmd-old-002",
		CorrelationID: "event-old-002",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		CreatedAt:     bootTime.Add(-1 * time.Nanosecond),
	}

	if rg.ShouldExecute(barelyOld) {
		t.Error("command 1ns before boot should be REJECTED by replay guard")
	}
}

// TestNewCommandAllowed verifies that a command after boot is allowed.
// Normal operation: commands created after boot execute normally.
func TestNewCommandAllowed(t *testing.T) {
	rg := NewReplayGuard()

	// Mark boot at a known time.
	bootTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	rg.MarkBoot(bootTime)

	// Command created AFTER boot.
	newCmd := ActuationCommand{
		ID:            "cmd-new-001",
		CorrelationID: "event-new-001",
		CommandType:   CommandStopRequestPulse,
		TargetAddress: "plc-relay://zone1/stop-request-input",
		CreatedAt:     bootTime.Add(1 * time.Second), // 1 second after boot
	}

	if !rg.ShouldExecute(newCmd) {
		t.Error("command from after boot should be ALLOWED by replay guard")
	}

	// Command much later should also pass.
	laterCmd := ActuationCommand{
		ID:            "cmd-new-002",
		CorrelationID: "event-new-002",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		CreatedAt:     bootTime.Add(24 * time.Hour), // 1 day after boot
	}

	if !rg.ShouldExecute(laterCmd) {
		t.Error("command from much later should be ALLOWED by replay guard")
	}
}

// TestExactBootTimeCommand verifies the edge case where command is created at
// exactly the boot time. This should be allowed (at or after boot).
func TestExactBootTimeCommand(t *testing.T) {
	rg := NewReplayGuard()

	bootTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	rg.MarkBoot(bootTime)

	// Command created at EXACTLY boot time.
	exactCmd := ActuationCommand{
		ID:            "cmd-exact-001",
		CorrelationID: "event-exact-001",
		CommandType:   CommandStopRequestPulse,
		TargetAddress: "plc-relay://zone1/stop-request-input",
		CreatedAt:     bootTime, // exactly at boot time
	}

	if !rg.ShouldExecute(exactCmd) {
		t.Error("command at exact boot time should be ALLOWED (boundary: at or after)")
	}
}

// TestReplayGuardGetBootTime verifies GetBootTime returns the set boot time.
func TestReplayGuardGetBootTime(t *testing.T) {
	rg := NewReplayGuard()

	bootTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	rg.MarkBoot(bootTime)

	if !rg.GetBootTime().Equal(bootTime) {
		t.Errorf("expected boot time %v, got %v", bootTime, rg.GetBootTime())
	}
}

// TestReplayGuardDefaultBootTime verifies that NewReplayGuard sets boot to now.
func TestReplayGuardDefaultBootTime(t *testing.T) {
	before := time.Now()
	rg := NewReplayGuard()
	after := time.Now()

	bt := rg.GetBootTime()
	if bt.Before(before) || bt.After(after) {
		t.Errorf("default boot time should be between %v and %v, got %v", before, after, bt)
	}
}
