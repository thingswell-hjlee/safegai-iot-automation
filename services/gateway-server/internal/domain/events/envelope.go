// Package events defines the core event domain types for the SafeGAI gateway.
package events

import "time"

// Quality represents the data quality indicator for an event payload.
type Quality string

const (
	QualityGood      Quality = "GOOD"
	QualityUncertain Quality = "UNCERTAIN"
	QualityBad       Quality = "BAD"
	QualityStale     Quality = "STALE"
)

// ValidQualities contains all valid Quality values.
var ValidQualities = []Quality{
	QualityGood,
	QualityUncertain,
	QualityBad,
	QualityStale,
}

// IsValid returns true if the Quality value is a recognized constant.
func (q Quality) IsValid() bool {
	for _, v := range ValidQualities {
		if q == v {
			return true
		}
	}
	return false
}

// EventEnvelope is the common wrapper for all SafeGAI domain events.
// Every event flowing through the gateway includes these fields.
type EventEnvelope struct {
	// SchemaVersion is the semantic version of the envelope schema.
	SchemaVersion string `json:"schemaVersion"`

	// EventID is the globally unique event identifier (UUIDv4).
	EventID string `json:"eventId"`

	// CorrelationID links related events in a causal chain.
	CorrelationID string `json:"correlationId"`

	// TenantID identifies the tenant for multi-tenant isolation.
	TenantID string `json:"tenantId"`

	// SiteID identifies the physical site or factory.
	SiteID string `json:"siteId"`

	// GatewayID identifies the gateway instance.
	GatewayID string `json:"gatewayId"`

	// DeviceID identifies the source device (camera, I/O module, etc.).
	DeviceID string `json:"deviceId"`

	// ZoneID identifies the logical zone within the site.
	ZoneID string `json:"zoneId"`

	// ObservedAt is when the event was observed at source.
	ObservedAt time.Time `json:"observedAt"`

	// ReceivedAt is when the gateway received the event.
	ReceivedAt time.Time `json:"receivedAt"`

	// SequenceNo is a monotonically increasing sequence number per source.
	SequenceNo int64 `json:"sequenceNo"`

	// Source is the logical source component name.
	Source string `json:"source"`

	// Quality indicates data quality for the event payload.
	Quality Quality `json:"quality"`

	// Payload holds the event-specific data as raw JSON.
	Payload []byte `json:"payload,omitempty"`
}

// Validate checks that all required fields of the EventEnvelope are populated
// and that field values are within their allowed ranges.
func (e *EventEnvelope) Validate() []string {
	var errs []string

	if e.SchemaVersion == "" {
		errs = append(errs, "schemaVersion is required")
	}
	if e.EventID == "" {
		errs = append(errs, "eventId is required")
	}
	if e.CorrelationID == "" {
		errs = append(errs, "correlationId is required")
	}
	if e.TenantID == "" {
		errs = append(errs, "tenantId is required")
	}
	if e.SiteID == "" {
		errs = append(errs, "siteId is required")
	}
	if e.GatewayID == "" {
		errs = append(errs, "gatewayId is required")
	}
	if e.DeviceID == "" {
		errs = append(errs, "deviceId is required")
	}
	if e.ZoneID == "" {
		errs = append(errs, "zoneId is required")
	}
	if e.ObservedAt.IsZero() {
		errs = append(errs, "observedAt is required")
	}
	if e.ReceivedAt.IsZero() {
		errs = append(errs, "receivedAt is required")
	}
	if e.SequenceNo < 0 {
		errs = append(errs, "sequenceNo must be non-negative")
	}
	if e.Source == "" {
		errs = append(errs, "source is required")
	}
	if !e.Quality.IsValid() {
		errs = append(errs, "quality must be one of GOOD, UNCERTAIN, BAD, STALE")
	}

	return errs
}
