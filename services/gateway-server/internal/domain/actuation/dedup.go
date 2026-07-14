package actuation

import (
	"sync"
	"time"
)

// DedupTracker prevents duplicate output commands for the same correlationId
// and commandType within a suppression window.
//
// SAFETY: Duplicate suppression ensures that the same safety event does not
// trigger multiple identical output commands to the PLC/Safety Relay.
// This prevents alarm fatigue and unnecessary actuation cycling.
type DedupTracker struct {
	mu                sync.Mutex
	records           map[string]time.Time
	suppressionWindow time.Duration
}

// NewDedupTracker creates a new DedupTracker with the given suppression window.
// Commands with the same correlationId+commandType within the window are suppressed.
func NewDedupTracker(suppressionWindow time.Duration) *DedupTracker {
	return &DedupTracker{
		records:           make(map[string]time.Time),
		suppressionWindow: suppressionWindow,
	}
}

// dedupKey generates the deduplication key from correlationId and commandType.
func dedupKey(correlationID string, commandType CommandType) string {
	return correlationID + "|" + string(commandType)
}

// IsDuplicate checks whether a command with the same correlationId and
// commandType has been executed within the suppression window.
//
// SAFETY: Returns true if a duplicate exists within the window, preventing
// repeated output to the PLC/Safety Relay for the same event.
func (d *DedupTracker) IsDuplicate(correlationID string, commandType CommandType) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := dedupKey(correlationID, commandType)
	lastExec, exists := d.records[key]
	if !exists {
		return false
	}

	// If within suppression window, it is a duplicate.
	return time.Since(lastExec) < d.suppressionWindow
}

// Record stores the execution time for a correlationId+commandType pair.
// Called after successful command execution.
func (d *DedupTracker) Record(correlationID string, commandType CommandType) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := dedupKey(correlationID, commandType)
	d.records[key] = time.Now()
}

// Cleanup removes entries older than the given threshold time.
// This prevents unbounded memory growth in long-running processes.
func (d *DedupTracker) Cleanup(before time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for key, ts := range d.records {
		if ts.Before(before) {
			delete(d.records, key)
		}
	}
}
