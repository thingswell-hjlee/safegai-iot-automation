package normalizer

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

func validRawEvent() camera.RawCameraEvent {
	return camera.RawCameraEvent{
		CameraID:    "cam-001",
		ZoneID:      "zone-A",
		EventType:   "person_detected",
		PersonCount: 1,
		Confidence:  0.95,
		Timestamp:   time.Now().UTC().Add(-100 * time.Millisecond),
	}
}

func TestNormalize_ValidEvent(t *testing.T) {
	cfg := DefaultNormalizerConfig()
	cfg.GatewayID = "gw-test"
	cfg.TenantID = "tenant-test"
	cfg.SiteID = "site-test"
	n := NewNormalizer(cfg)

	raw := validRawEvent()
	result := n.Normalize(raw)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Envelope == nil {
		t.Fatal("expected non-nil envelope")
	}

	env := result.Envelope

	// Check required fields
	if env.SchemaVersion != "1.0.0" {
		t.Errorf("schemaVersion = %q, want %q", env.SchemaVersion, "1.0.0")
	}
	if env.EventID == "" {
		t.Error("eventId should be generated")
	}
	if env.CorrelationID == "" {
		t.Error("correlationId should be generated")
	}
	if env.TenantID != "tenant-test" {
		t.Errorf("tenantId = %q, want %q", env.TenantID, "tenant-test")
	}
	if env.SiteID != "site-test" {
		t.Errorf("siteId = %q, want %q", env.SiteID, "site-test")
	}
	if env.GatewayID != "gw-test" {
		t.Errorf("gatewayId = %q, want %q", env.GatewayID, "gw-test")
	}
	if env.DeviceID != raw.CameraID {
		t.Errorf("deviceId = %q, want %q", env.DeviceID, raw.CameraID)
	}
	if env.ZoneID != raw.ZoneID {
		t.Errorf("zoneId = %q, want %q", env.ZoneID, raw.ZoneID)
	}
	if env.ObservedAt != raw.Timestamp {
		t.Errorf("observedAt mismatch")
	}
	if env.ReceivedAt.IsZero() {
		t.Error("receivedAt should be set")
	}
	if env.ReceivedAt.Before(env.ObservedAt) {
		t.Error("receivedAt should be after observedAt")
	}
	if env.SequenceNo < 1 {
		t.Errorf("sequenceNo should be positive, got %d", env.SequenceNo)
	}
	if env.Source != "camera-adapter" {
		t.Errorf("source = %q, want %q", env.Source, "camera-adapter")
	}
	if env.Quality != events.QualityGood {
		t.Errorf("quality = %q, want %q (confidence=0.95)", env.Quality, events.QualityGood)
	}

	// Validate envelope passes domain validation
	errs := env.Validate()
	if len(errs) > 0 {
		t.Errorf("normalized envelope has validation errors: %v", errs)
	}
}

func TestNormalize_MissingCameraID(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.CameraID = ""

	result := n.Normalize(raw)
	if result.Err == nil {
		t.Error("expected error for missing cameraId")
	}
	if !strings.Contains(result.Err.Error(), "cameraId") {
		t.Errorf("error should mention cameraId: %v", result.Err)
	}
}

func TestNormalize_MissingEventType(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.EventType = ""

	result := n.Normalize(raw)
	if result.Err == nil {
		t.Error("expected error for missing eventType")
	}
	if !strings.Contains(result.Err.Error(), "eventType") {
		t.Errorf("error should mention eventType: %v", result.Err)
	}
}

func TestNormalize_MissingTimestamp(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.Timestamp = time.Time{}

	result := n.Normalize(raw)
	if result.Err == nil {
		t.Error("expected error for missing timestamp")
	}
	if !strings.Contains(result.Err.Error(), "timestamp") {
		t.Errorf("error should mention timestamp: %v", result.Err)
	}
}

func TestNormalize_MissingZoneIDForDetectionEvent(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.ZoneID = ""

	result := n.Normalize(raw)
	if result.Err == nil {
		t.Error("expected error for missing zoneId on detection event")
	}
	if !strings.Contains(result.Err.Error(), "zoneId") {
		t.Errorf("error should mention zoneId: %v", result.Err)
	}
}

func TestNormalize_OfflineEventAllowsEmptyZoneID(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := camera.RawCameraEvent{
		CameraID:  "cam-001",
		ZoneID:    "",
		EventType: "offline",
		Timestamp: time.Now().UTC(),
	}

	result := n.Normalize(raw)
	if result.Err != nil {
		t.Fatalf("offline event should allow empty zoneId: %v", result.Err)
	}
}

func TestNormalize_EventIDGeneration(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())

	// Generate two events and verify different IDs
	raw1 := validRawEvent()
	raw2 := validRawEvent()
	raw2.Timestamp = raw2.Timestamp.Add(1 * time.Second)

	result1 := n.Normalize(raw1)
	result2 := n.Normalize(raw2)

	if result1.Err != nil || result2.Err != nil {
		t.Fatal("unexpected errors")
	}

	if result1.Envelope.EventID == result2.Envelope.EventID {
		t.Error("expected different event IDs for different events")
	}

	// Verify UUID-like format (8-4-4-4-12)
	parts := strings.Split(result1.Envelope.EventID, "-")
	if len(parts) != 5 {
		t.Errorf("eventId should be UUID format (5 parts), got %d parts: %q", len(parts), result1.Envelope.EventID)
	}
}

func TestNormalize_ReceivedAtIsSet(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	before := time.Now().UTC()

	raw := validRawEvent()
	result := n.Normalize(raw)

	after := time.Now().UTC()

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	receivedAt := result.Envelope.ReceivedAt
	if receivedAt.Before(before) || receivedAt.After(after) {
		t.Errorf("receivedAt %v not between %v and %v", receivedAt, before, after)
	}
}

func TestNormalize_HighConfidenceGoodQuality(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.Confidence = 0.90

	result := n.Normalize(raw)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Envelope.Quality != events.QualityGood {
		t.Errorf("quality = %q, want GOOD for confidence 0.90", result.Envelope.Quality)
	}
}

func TestNormalize_LowConfidenceUncertainQuality(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.Confidence = 0.5

	result := n.Normalize(raw)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Envelope.Quality != events.QualityUncertain {
		t.Errorf("quality = %q, want UNCERTAIN for confidence 0.5", result.Envelope.Quality)
	}
}

func TestNormalize_OfflineEventUncertainQuality(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := camera.RawCameraEvent{
		CameraID:  "cam-001",
		ZoneID:    "",
		EventType: "offline",
		Timestamp: time.Now().UTC(),
	}

	result := n.Normalize(raw)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	// Offline must map to UNCERTAIN quality (which maps to UNKNOWN state, never VACANT)
	if result.Envelope.Quality != events.QualityUncertain {
		t.Errorf("quality = %q, want UNCERTAIN for offline event", result.Envelope.Quality)
	}
}

func TestNormalize_PayloadContainsEventData(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())
	raw := validRawEvent()
	raw.SnapshotURL = "http://sim/snapshot.jpg"

	result := n.Normalize(raw)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(result.Envelope.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if payload["eventType"] != "person_detected" {
		t.Errorf("payload eventType = %v, want person_detected", payload["eventType"])
	}
	if payload["personCount"] != float64(1) {
		t.Errorf("payload personCount = %v, want 1", payload["personCount"])
	}
	if payload["snapshotUrl"] != "http://sim/snapshot.jpg" {
		t.Errorf("payload snapshotUrl = %v, want http://sim/snapshot.jpg", payload["snapshotUrl"])
	}
}

func TestNormalize_SequenceNoIncreases(t *testing.T) {
	n := NewNormalizer(DefaultNormalizerConfig())

	raw := validRawEvent()
	r1 := n.Normalize(raw)
	r2 := n.Normalize(raw)
	r3 := n.Normalize(raw)

	if r1.Err != nil || r2.Err != nil || r3.Err != nil {
		t.Fatal("unexpected errors")
	}

	if r2.Envelope.SequenceNo <= r1.Envelope.SequenceNo {
		t.Error("sequence numbers should be monotonically increasing")
	}
	if r3.Envelope.SequenceNo <= r2.Envelope.SequenceNo {
		t.Error("sequence numbers should be monotonically increasing")
	}
}

func TestNormalize_NeverProducesVacantConfirmed(t *testing.T) {
	// Safety rule: Camera adapter must NEVER produce VACANT_CONFIRMED.
	// Only the Zone State Engine determines vacancy after a timeout period.
	n := NewNormalizer(DefaultNormalizerConfig())

	testCases := []camera.RawCameraEvent{
		{CameraID: "cam-001", ZoneID: "zone-A", EventType: "person_not_detected", Timestamp: time.Now()},
		{CameraID: "cam-001", ZoneID: "", EventType: "offline", Timestamp: time.Now()},
		{CameraID: "cam-001", ZoneID: "zone-A", EventType: "person_detected", PersonCount: 0, Timestamp: time.Now()},
	}

	for _, raw := range testCases {
		result := n.Normalize(raw)
		if result.Err != nil {
			continue // Skip invalid events
		}
		// Check that the payload never contains VACANT_CONFIRMED
		if strings.Contains(string(result.Envelope.Payload), "VACANT_CONFIRMED") {
			t.Errorf("normalizer must NEVER produce VACANT_CONFIRMED in payload for event type %q", raw.EventType)
		}
	}
}
