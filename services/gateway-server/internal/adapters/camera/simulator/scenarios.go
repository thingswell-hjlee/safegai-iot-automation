package simulator

import (
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
)

// ScenarioFunc is a function that generates a sequence of RawCameraEvents for testing.
type ScenarioFunc func(baseTime time.Time) []camera.RawCameraEvent

// Occupied generates events simulating a zone with a person present.
func Occupied(zoneID string) ScenarioFunc {
	return func(baseTime time.Time) []camera.RawCameraEvent {
		return []camera.RawCameraEvent{
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.95,
				Timestamp:   baseTime,
			},
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.92,
				Timestamp:   baseTime.Add(500 * time.Millisecond),
			},
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.97,
				Timestamp:   baseTime.Add(1 * time.Second),
			},
		}
	}
}

// Vacant generates events simulating a zone where a person leaves.
// Note: The camera adapter only reports "person_not_detected". It NEVER emits
// VACANT_CONFIRMED. The Zone State Engine determines vacancy after a timeout.
func Vacant(zoneID string) ScenarioFunc {
	return func(baseTime time.Time) []camera.RawCameraEvent {
		return []camera.RawCameraEvent{
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.90,
				Timestamp:   baseTime,
			},
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_not_detected",
				PersonCount: 0,
				Confidence:  0.88,
				Timestamp:   baseTime.Add(3 * time.Second),
			},
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_not_detected",
				PersonCount: 0,
				Confidence:  0.93,
				Timestamp:   baseTime.Add(6 * time.Second),
			},
		}
	}
}

// Offline generates events simulating a camera going offline.
// An offline camera maps to UNKNOWN state, NEVER VACANT.
func Offline() ScenarioFunc {
	return func(baseTime time.Time) []camera.RawCameraEvent {
		return []camera.RawCameraEvent{
			{
				CameraID:    "cam-sim-001",
				ZoneID:      "",
				EventType:   "offline",
				PersonCount: 0,
				Confidence:  0.0,
				Timestamp:   baseTime,
			},
		}
	}
}

// Duplicate generates events with duplicate entries within a short time window.
// Used to test duplicate suppression logic.
func Duplicate(zoneID string) ScenarioFunc {
	return func(baseTime time.Time) []camera.RawCameraEvent {
		return []camera.RawCameraEvent{
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.95,
				Timestamp:   baseTime,
			},
			// Duplicate: same camera, zone, type, within dedup window
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.95,
				Timestamp:   baseTime.Add(100 * time.Millisecond),
			},
			// Duplicate: same camera, zone, type, within dedup window
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.95,
				Timestamp:   baseTime.Add(200 * time.Millisecond),
			},
			// After window: should pass through
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.95,
				Timestamp:   baseTime.Add(3 * time.Second),
			},
		}
	}
}

// Malformed generates events with missing or invalid fields.
// Used to test event validation and error handling.
func Malformed() ScenarioFunc {
	return func(baseTime time.Time) []camera.RawCameraEvent {
		return []camera.RawCameraEvent{
			// Missing cameraId
			{
				CameraID:    "",
				ZoneID:      "zone-A",
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.90,
				Timestamp:   baseTime,
			},
			// Missing eventType
			{
				CameraID:    "cam-sim-001",
				ZoneID:      "zone-A",
				EventType:   "",
				PersonCount: 0,
				Confidence:  0.0,
				Timestamp:   baseTime.Add(1 * time.Second),
			},
			// Zero timestamp
			{
				CameraID:    "cam-sim-001",
				ZoneID:      "zone-A",
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.80,
				Timestamp:   time.Time{},
			},
		}
	}
}

// OutOfOrder generates events with timestamps that arrive out of sequence.
// Used to test out-of-order handling.
func OutOfOrder(zoneID string) ScenarioFunc {
	return func(baseTime time.Time) []camera.RawCameraEvent {
		return []camera.RawCameraEvent{
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 1,
				Confidence:  0.95,
				Timestamp:   baseTime.Add(2 * time.Second),
			},
			// This event has an earlier timestamp but arrives later
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_detected",
				PersonCount: 2,
				Confidence:  0.90,
				Timestamp:   baseTime,
			},
			{
				CameraID:    "cam-sim-001",
				ZoneID:      zoneID,
				EventType:   "person_not_detected",
				PersonCount: 0,
				Confidence:  0.88,
				Timestamp:   baseTime.Add(5 * time.Second),
			},
		}
	}
}
