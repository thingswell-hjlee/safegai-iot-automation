package actuation

import (
	"sync"
	"time"
)

// ReplayGuard prevents replaying past output commands after a system restart.
//
// SAFETY: After a restart, the gateway must NOT replay past pulse commands.
// Commands created before the current boot time are rejected.
// This prevents stale safety actions from being re-executed in a new context
// where conditions may have changed.
//
// FORBIDDEN: Replaying past output commands after boot is explicitly forbidden.
// A stop request from a previous boot cycle may no longer be valid.
type ReplayGuard struct {
	mu       sync.RWMutex
	bootTime time.Time
}

// NewReplayGuard creates a new ReplayGuard with the boot time set to now.
// Commands created before this time will be rejected.
func NewReplayGuard() *ReplayGuard {
	return &ReplayGuard{
		bootTime: time.Now(),
	}
}

// MarkBoot sets the boot time. Commands created before this time are rejected.
//
// SAFETY: This establishes the boundary between "old" commands (pre-restart)
// and "new" commands (post-restart). Only new commands are executed.
func (rg *ReplayGuard) MarkBoot(bootTime time.Time) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	rg.bootTime = bootTime
}

// ShouldExecute returns true if the command was created at or after boot time.
// Returns false if the command predates the current boot (stale command).
//
// SAFETY: Prevents replaying past output commands after restart.
// A command from a previous boot cycle is NOT executed because conditions
// may have changed and the safety decision may no longer be valid.
func (rg *ReplayGuard) ShouldExecute(cmd ActuationCommand) bool {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	// Commands created at or after boot time are allowed.
	// Commands created strictly before boot time are blocked.
	return !cmd.CreatedAt.Before(rg.bootTime)
}

// GetBootTime returns the current boot time.
func (rg *ReplayGuard) GetBootTime() time.Time {
	rg.mu.RLock()
	defer rg.mu.RUnlock()
	return rg.bootTime
}
