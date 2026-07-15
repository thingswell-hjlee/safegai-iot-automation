// Package normalizer converts raw camera events into domain EventEnvelope.
// It validates required fields, generates event IDs, and applies timestamp normalization.
package normalizer

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// NormalizerConfig holds configuration for the event normalizer.
type NormalizerConfig struct {
	// GatewayID identifies this gateway instance.
	GatewayID string

	// TenantID identifies the tenant.
	TenantID string

	// SiteID identifies the site.
	SiteID string

	// SchemaVersion for normalized events.
	SchemaVersion string
}

// DefaultNormalizerConfig returns a default normalizer configuration.
func DefaultNormalizerConfig() NormalizerConfig {
	return NormalizerConfig{
		GatewayID:     "gw-default",
		TenantID:      "tenant-default",
		SiteID:        "site-default",
		SchemaVersion: "1.0.0",
	}
}

// Normalizer converts RawCameraEvent to domain EventEnvelope.
type Normalizer struct {
	config NormalizerConfig
	seqNo  int64
}

// NewNormalizer creates a new event normalizer with the given configuration.
func NewNormalizer(cfg NormalizerConfig) *Normalizer {
	return &Normalizer{
		config: cfg,
		seqNo:  0,
	}
}

// NormalizeResult holds the result of normalizing a raw camera event.
type NormalizeResult struct {
	Envelope *events.EventEnvelope
	Err      error
}

// Normalize converts a RawCameraEvent into a domain EventEnvelope.
// It validates required fields, generates an eventId, sets receivedAt,
// and determines quality based on the event type.
//
// Safety rule: Camera offline maps to UNKNOWN quality, never VACANT.
// Safety rule: Camera adapter NEVER produces VACANT_CONFIRMED.
func (n *Normalizer) Normalize(raw camera.RawCameraEvent) NormalizeResult {
	// Validate required fields
	if err := n.validate(raw); err != nil {
		return NormalizeResult{Err: err}
	}

	now := time.Now().UTC()
	n.seqNo++

	// Determine quality based on event type
	quality := n.determineQuality(raw)

	// Build payload from raw event data
	payload, err := n.buildPayload(raw)
	if err != nil {
		return NormalizeResult{Err: fmt.Errorf("normalizer: failed to build payload: %w", err)}
	}

	// Generate correlation ID (same as event ID for camera-originated events)
	eventID := generateUUID()

	envelope := &events.EventEnvelope{
		SchemaVersion: n.config.SchemaVersion,
		EventID:       eventID,
		CorrelationID: eventID,
		TenantID:      n.config.TenantID,
		SiteID:        n.config.SiteID,
		GatewayID:     n.config.GatewayID,
		DeviceID:      raw.CameraID,
		ZoneID:        raw.ZoneID,
		ObservedAt:    raw.Timestamp,
		ReceivedAt:    now,
		SequenceNo:    n.seqNo,
		Source:        "camera-adapter",
		Quality:       quality,
		Payload:       payload,
	}

	return NormalizeResult{Envelope: envelope}
}

// validate checks that required fields are present in the raw event.
func (n *Normalizer) validate(raw camera.RawCameraEvent) error {
	if raw.CameraID == "" {
		return fmt.Errorf("normalizer: cameraId is required")
	}
	if raw.EventType == "" {
		return fmt.Errorf("normalizer: eventType is required")
	}
	if raw.Timestamp.IsZero() {
		return fmt.Errorf("normalizer: timestamp is required")
	}
	// ZoneID is required for detection events, optional for offline
	if raw.ZoneID == "" && raw.EventType != "offline" {
		return fmt.Errorf("normalizer: zoneId is required for non-offline events")
	}
	return nil
}

// determineQuality maps event type to data quality.
// Camera offline = UNCERTAIN quality (maps to UNKNOWN occupancy state at zone engine level).
// Camera offline must NEVER map to any form of vacancy.
func (n *Normalizer) determineQuality(raw camera.RawCameraEvent) events.Quality {
	switch raw.EventType {
	case "offline":
		// Offline camera = UNCERTAIN quality.
		// The zone state engine interprets UNCERTAIN as UNKNOWN occupancy state.
		// This ensures offline camera NEVER implies vacancy.
		return events.QualityUncertain
	case "person_detected", "person_not_detected":
		if raw.Confidence >= 0.8 {
			return events.QualityGood
		}
		return events.QualityUncertain
	default:
		return events.QualityUncertain
	}
}

// cameraEventPayload is the JSON structure stored in the envelope payload.
type cameraEventPayload struct {
	EventType   string  `json:"eventType"`
	PersonCount int     `json:"personCount"`
	Confidence  float64 `json:"confidence"`
	SnapshotURL string  `json:"snapshotUrl,omitempty"`
}

// buildPayload constructs the JSON payload for the envelope.
func (n *Normalizer) buildPayload(raw camera.RawCameraEvent) ([]byte, error) {
	p := cameraEventPayload{
		EventType:   raw.EventType,
		PersonCount: raw.PersonCount,
		Confidence:  raw.Confidence,
		SnapshotURL: raw.SnapshotURL,
	}
	return json.Marshal(p)
}

// generateUUID produces a version 4 UUID string using crypto/rand.
func generateUUID() string {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		// Fallback: use timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}

	// Set version (4) and variant (RFC 4122)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
