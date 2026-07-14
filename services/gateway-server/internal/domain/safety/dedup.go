package safety

// This file implements Rule R-05: Duplicate event suppression.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
//
// Rule R-05: duplicate events within same camera/zone/type/time-window
// must be consolidated - no duplicate output.
//
// The DedupFilter operates on the output side: if the same decision
// (same zone + equipment + decision type) was already emitted within
// the suppression window, the duplicate is suppressed.
//
// [R3] Suppression only affects OUTPUT commands.
// [R3] The underlying evaluation is still performed every cycle.
// [R3] A new distinct decision (different type) is never suppressed.

import (
	"fmt"
	"sync"
	"time"
)

// dedupKey uniquely identifies a decision for suppression purposes.
type dedupKey struct {
	ZoneID      string
	EquipmentID string
	Decision    SafetyDecision
}

// dedupEntry tracks when a decision was last emitted.
type dedupEntry struct {
	lastEmitted time.Time
}

// DedupFilter implements Rule R-05 duplicate suppression.
// [R3] Thread-safe via internal mutex.
type DedupFilter struct {
	mu                sync.Mutex
	suppressionWindow time.Duration
	entries           map[string]*dedupEntry
}

// NewDedupFilter creates a new DedupFilter with the given suppression window.
// Decisions for the same zone+equipment+type within this window are suppressed.
func NewDedupFilter(suppressionWindow time.Duration) *DedupFilter {
	return &DedupFilter{
		suppressionWindow: suppressionWindow,
		entries:           make(map[string]*dedupEntry),
	}
}

// IsDuplicate checks if the given decision result is a duplicate.
// If it is NOT a duplicate, it records the emission and returns false.
// If it IS a duplicate (same key within suppression window), returns true.
//
// [R3] SAFE decisions are never deduplicated (they represent normal state).
// [R3] This does not prevent evaluation, only duplicate output.
func (f *DedupFilter) IsDuplicate(result *DecisionResult) bool {
	// SAFE decisions are always passed through (they're the default)
	if result.Decision == DecisionSafe {
		return false
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	key := f.makeKey(result.ZoneID, result.EquipmentID, result.Decision)
	now := time.Now()

	entry, exists := f.entries[key]
	if exists && now.Sub(entry.lastEmitted) < f.suppressionWindow {
		// Duplicate within suppression window
		return true
	}

	// Record emission
	f.entries[key] = &dedupEntry{lastEmitted: now}
	return false
}

// Reset clears all dedup state. Used for testing or when context changes significantly.
func (f *DedupFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = make(map[string]*dedupEntry)
}

// Cleanup removes expired entries older than the suppression window.
// This should be called periodically to prevent unbounded memory growth.
func (f *DedupFilter) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	for key, entry := range f.entries {
		if now.Sub(entry.lastEmitted) >= f.suppressionWindow {
			delete(f.entries, key)
		}
	}
}

// makeKey creates a string key for the dedup map.
func (f *DedupFilter) makeKey(zoneID, equipmentID string, decision SafetyDecision) string {
	return fmt.Sprintf("%s|%s|%s", zoneID, equipmentID, string(decision))
}
