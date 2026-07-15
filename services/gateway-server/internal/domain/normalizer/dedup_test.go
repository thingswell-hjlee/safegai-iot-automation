package normalizer

import (
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
)

func baseEvent(t time.Time) camera.RawCameraEvent {
	return camera.RawCameraEvent{
		CameraID:    "cam-001",
		ZoneID:      "zone-A",
		EventType:   "person_detected",
		PersonCount: 1,
		Confidence:  0.95,
		Timestamp:   t,
	}
}

func TestDedup_FirstEventPasses(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	evt := baseEvent(time.Now().UTC())

	if d.IsDuplicate(evt) {
		t.Error("first event should not be flagged as duplicate")
	}
}

func TestDedup_DuplicateWithinWindowIsSuppressed(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(100 * time.Millisecond))

	if d.IsDuplicate(evt1) {
		t.Error("first event should not be duplicate")
	}
	if !d.IsDuplicate(evt2) {
		t.Error("second event within 2s window should be flagged as duplicate")
	}
}

func TestDedup_EventAfterWindowPasses(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(3 * time.Second)) // After 2s window

	if d.IsDuplicate(evt1) {
		t.Error("first event should not be duplicate")
	}
	if d.IsDuplicate(evt2) {
		t.Error("event after 2s window should not be flagged as duplicate")
	}
}

func TestDedup_DifferentCameraIDNotDuplicate(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(100 * time.Millisecond))
	evt2.CameraID = "cam-002"

	if d.IsDuplicate(evt1) {
		t.Error("first event should not be duplicate")
	}
	if d.IsDuplicate(evt2) {
		t.Error("event from different camera should not be duplicate")
	}
}

func TestDedup_DifferentZoneNotDuplicate(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(100 * time.Millisecond))
	evt2.ZoneID = "zone-B"

	if d.IsDuplicate(evt1) {
		t.Error("first event should not be duplicate")
	}
	if d.IsDuplicate(evt2) {
		t.Error("event from different zone should not be duplicate")
	}
}

func TestDedup_DifferentEventTypeNotDuplicate(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(100 * time.Millisecond))
	evt2.EventType = "person_not_detected"

	if d.IsDuplicate(evt1) {
		t.Error("first event should not be duplicate")
	}
	if d.IsDuplicate(evt2) {
		t.Error("event with different type should not be duplicate")
	}
}

func TestDedup_MultipleDuplicatesWithinWindow(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(200 * time.Millisecond))
	evt3 := baseEvent(now.Add(400 * time.Millisecond))
	evt4 := baseEvent(now.Add(1500 * time.Millisecond))

	if d.IsDuplicate(evt1) {
		t.Error("first event should pass")
	}
	if !d.IsDuplicate(evt2) {
		t.Error("second event should be suppressed")
	}
	if !d.IsDuplicate(evt3) {
		t.Error("third event should be suppressed")
	}
	if !d.IsDuplicate(evt4) {
		t.Error("fourth event still within 2s of first should be suppressed")
	}
}

func TestDedup_EventExactlyAtWindowBoundaryPasses(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(DefaultDedupWindow)) // Exactly at 2s boundary

	if d.IsDuplicate(evt1) {
		t.Error("first event should pass")
	}
	if d.IsDuplicate(evt2) {
		t.Error("event exactly at window boundary should pass (not strictly less than)")
	}
}

func TestDedup_CustomWindowSize(t *testing.T) {
	cfg := DedupConfig{Window: 500 * time.Millisecond}
	d := NewDuplicateSuppressor(cfg)
	now := time.Now().UTC()

	evt1 := baseEvent(now)
	evt2 := baseEvent(now.Add(300 * time.Millisecond))
	evt3 := baseEvent(now.Add(600 * time.Millisecond))

	if d.IsDuplicate(evt1) {
		t.Error("first event should pass")
	}
	if !d.IsDuplicate(evt2) {
		t.Error("event within 500ms window should be suppressed")
	}
	if d.IsDuplicate(evt3) {
		t.Error("event after 500ms window should pass")
	}
}

func TestDedup_CleanupRemovesExpiredEntries(t *testing.T) {
	cfg := DedupConfig{Window: 1 * time.Millisecond}
	d := NewDuplicateSuppressor(cfg)

	// Add some events
	now := time.Now().UTC().Add(-1 * time.Hour) // far in the past
	evt := baseEvent(now)
	d.IsDuplicate(evt)

	if d.Size() != 1 {
		t.Fatalf("expected 1 entry, got %d", d.Size())
	}

	// Cleanup should remove entries older than window*10
	d.Cleanup()

	if d.Size() != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", d.Size())
	}
}

func TestDedup_ConcurrentAccess(t *testing.T) {
	d := NewDuplicateSuppressor(DefaultDedupConfig())
	now := time.Now().UTC()

	// Run concurrent checks to verify no data race
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			evt := baseEvent(now.Add(time.Duration(i) * time.Millisecond))
			d.IsDuplicate(evt)
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
