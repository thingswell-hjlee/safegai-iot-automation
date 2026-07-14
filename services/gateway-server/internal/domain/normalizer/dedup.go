package normalizer

import (
	"fmt"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
)

// DefaultDedupWindow is the default time window for duplicate suppression (2 seconds).
const DefaultDedupWindow = 2 * time.Second

// DedupConfig holds configuration for the duplicate suppressor.
type DedupConfig struct {
	// Window is the time duration within which duplicate events are suppressed.
	Window time.Duration
}

// DefaultDedupConfig returns a default dedup configuration with a 2-second window.
func DefaultDedupConfig() DedupConfig {
	return DedupConfig{
		Window: DefaultDedupWindow,
	}
}

// dedupEntry tracks when an event key was last seen.
type dedupEntry struct {
	lastSeen time.Time
}

// DuplicateSuppressor detects and suppresses duplicate camera events
// within a configurable time window.
// An event is considered a duplicate if another event with the same key
// (cameraId + zoneId + eventType) was seen within the dedup window,
// where the key's observedAt timestamp falls within the window.
type DuplicateSuppressor struct {
	mu      sync.Mutex
	config  DedupConfig
	entries map[string]dedupEntry
}

// NewDuplicateSuppressor creates a new duplicate suppressor with the given configuration.
func NewDuplicateSuppressor(cfg DedupConfig) *DuplicateSuppressor {
	return &DuplicateSuppressor{
		config:  cfg,
		entries: make(map[string]dedupEntry),
	}
}

// IsDuplicate checks whether the given event is a duplicate within the configured window.
// Returns true if the event should be suppressed (is a duplicate).
// Returns false if the event should be passed through (first occurrence or outside window).
func (d *DuplicateSuppressor) IsDuplicate(evt camera.RawCameraEvent) bool {
	key := d.eventKey(evt)

	d.mu.Lock()
	defer d.mu.Unlock()

	entry, exists := d.entries[key]
	if !exists {
		// First time seeing this event key
		d.entries[key] = dedupEntry{lastSeen: evt.Timestamp}
		return false
	}

	// Check if the event falls within the dedup window of the last seen event
	elapsed := evt.Timestamp.Sub(entry.lastSeen)
	if elapsed < 0 {
		// Negative elapsed means out-of-order; use absolute value
		elapsed = -elapsed
	}

	if elapsed < d.config.Window {
		// Within window: this is a duplicate
		return true
	}

	// Outside window: not a duplicate, update entry
	d.entries[key] = dedupEntry{lastSeen: evt.Timestamp}
	return false
}

// eventKey generates a composite key for duplicate detection.
// Key = cameraId + zoneId + eventType
func (d *DuplicateSuppressor) eventKey(evt camera.RawCameraEvent) string {
	return fmt.Sprintf("%s:%s:%s", evt.CameraID, evt.ZoneID, evt.EventType)
}

// Cleanup removes expired entries from the dedup map.
// Should be called periodically to prevent unbounded memory growth.
func (d *DuplicateSuppressor) Cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	for key, entry := range d.entries {
		if now.Sub(entry.lastSeen) > d.config.Window*10 {
			delete(d.entries, key)
		}
	}
}

// Size returns the number of tracked event keys.
func (d *DuplicateSuppressor) Size() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.entries)
}
