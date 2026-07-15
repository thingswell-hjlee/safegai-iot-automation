package camera

import "time"

// RawCameraEvent represents an unprocessed event from a camera device.
// This is the raw data before normalization into a domain EventEnvelope.
type RawCameraEvent struct {
	// CameraID uniquely identifies the camera device.
	CameraID string `json:"cameraId"`

	// ZoneID identifies the zone the camera is monitoring.
	ZoneID string `json:"zoneId"`

	// EventType classifies the camera event (e.g., "person_detected", "person_not_detected", "offline").
	EventType string `json:"eventType"`

	// PersonCount is the number of persons detected in the zone.
	PersonCount int `json:"personCount"`

	// Confidence is the detection confidence score (0.0 to 1.0).
	Confidence float64 `json:"confidence"`

	// Timestamp is when the event was observed at the camera.
	Timestamp time.Time `json:"timestamp"`

	// SnapshotURL is the optional URL to a snapshot image.
	SnapshotURL string `json:"snapshotUrl,omitempty"`

	// RawPayload holds any additional camera-specific data as raw JSON.
	RawPayload []byte `json:"rawPayload,omitempty"`
}

// CameraHealth represents the health status of a camera device.
type CameraHealth struct {
	// Online indicates whether the camera is currently connected and responsive.
	Online bool `json:"online"`

	// LastEventAt is the timestamp of the most recent event received.
	LastEventAt time.Time `json:"lastEventAt"`

	// ErrorCount is the number of errors since last reset.
	ErrorCount int `json:"errorCount"`

	// LatencyMs is the last measured round-trip latency in milliseconds.
	LatencyMs int64 `json:"latencyMs"`
}

// Capabilities describes what a camera device supports.
type Capabilities struct {
	// Zones lists the zone identifiers this camera monitors.
	Zones []string `json:"zones"`

	// MaxPersons is the maximum number of persons the camera can detect simultaneously.
	MaxPersons int `json:"maxPersons"`

	// SupportsSnapshot indicates whether the camera supports snapshot capture.
	SupportsSnapshot bool `json:"supportsSnapshot"`

	// StreamURL is the optional URL for a live video stream.
	StreamURL string `json:"streamUrl,omitempty"`
}
