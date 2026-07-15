package events

import (
	"encoding/json"
	"testing"
	"time"
)

func validEnvelope() EventEnvelope {
	now := time.Now().UTC()
	return EventEnvelope{
		SchemaVersion: "1.0.0",
		EventID:       "550e8400-e29b-41d4-a716-446655440000",
		CorrelationID: "660e8400-e29b-41d4-a716-446655440001",
		TenantID:      "tenant-001",
		SiteID:        "site-001",
		GatewayID:     "gw-001",
		DeviceID:      "cam-001",
		ZoneID:        "zone-A",
		ObservedAt:    now.Add(-100 * time.Millisecond),
		ReceivedAt:    now,
		SequenceNo:    42,
		Source:        "camera-adapter",
		Quality:       QualityGood,
	}
}

func TestEventEnvelope_ValidateAllFieldsPresent(t *testing.T) {
	env := validEnvelope()
	errs := env.Validate()
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got: %v", errs)
	}
}

func TestEventEnvelope_ValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*EventEnvelope)
		wantErr string
	}{
		{"missing schemaVersion", func(e *EventEnvelope) { e.SchemaVersion = "" }, "schemaVersion is required"},
		{"missing eventId", func(e *EventEnvelope) { e.EventID = "" }, "eventId is required"},
		{"missing correlationId", func(e *EventEnvelope) { e.CorrelationID = "" }, "correlationId is required"},
		{"missing tenantId", func(e *EventEnvelope) { e.TenantID = "" }, "tenantId is required"},
		{"missing siteId", func(e *EventEnvelope) { e.SiteID = "" }, "siteId is required"},
		{"missing gatewayId", func(e *EventEnvelope) { e.GatewayID = "" }, "gatewayId is required"},
		{"missing deviceId", func(e *EventEnvelope) { e.DeviceID = "" }, "deviceId is required"},
		{"missing zoneId", func(e *EventEnvelope) { e.ZoneID = "" }, "zoneId is required"},
		{"missing observedAt", func(e *EventEnvelope) { e.ObservedAt = time.Time{} }, "observedAt is required"},
		{"missing receivedAt", func(e *EventEnvelope) { e.ReceivedAt = time.Time{} }, "receivedAt is required"},
		{"negative sequenceNo", func(e *EventEnvelope) { e.SequenceNo = -1 }, "sequenceNo must be non-negative"},
		{"missing source", func(e *EventEnvelope) { e.Source = "" }, "source is required"},
		{"invalid quality", func(e *EventEnvelope) { e.Quality = "INVALID" }, "quality must be one of GOOD, UNCERTAIN, BAD, STALE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validEnvelope()
			tt.modify(&env)
			errs := env.Validate()
			found := false
			for _, err := range errs {
				if err == tt.wantErr {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error %q in %v", tt.wantErr, errs)
			}
		})
	}
}

func TestEventEnvelope_ValidateMultipleErrors(t *testing.T) {
	env := EventEnvelope{}
	errs := env.Validate()
	// Should have at least schemaVersion, eventId, correlationId, tenantId,
	// siteId, gatewayId, deviceId, zoneId, observedAt, receivedAt, source, quality
	if len(errs) < 12 {
		t.Errorf("expected at least 12 validation errors for empty envelope, got %d: %v", len(errs), errs)
	}
}

func TestEventEnvelope_SequenceNoZeroIsValid(t *testing.T) {
	env := validEnvelope()
	env.SequenceNo = 0
	errs := env.Validate()
	if len(errs) != 0 {
		t.Errorf("sequenceNo=0 should be valid, got errors: %v", errs)
	}
}

func TestEventEnvelope_JSONRoundtrip(t *testing.T) {
	env := validEnvelope()
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded EventEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.EventID != env.EventID {
		t.Errorf("eventId mismatch: got %q, want %q", decoded.EventID, env.EventID)
	}
	if decoded.Quality != env.Quality {
		t.Errorf("quality mismatch: got %q, want %q", decoded.Quality, env.Quality)
	}
	if decoded.SequenceNo != env.SequenceNo {
		t.Errorf("sequenceNo mismatch: got %d, want %d", decoded.SequenceNo, env.SequenceNo)
	}
}

func TestQuality_IsValid(t *testing.T) {
	tests := []struct {
		quality Quality
		valid   bool
	}{
		{QualityGood, true},
		{QualityUncertain, true},
		{QualityBad, true},
		{QualityStale, true},
		{"INVALID", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.quality), func(t *testing.T) {
			if got := tt.quality.IsValid(); got != tt.valid {
				t.Errorf("Quality(%q).IsValid() = %v, want %v", tt.quality, got, tt.valid)
			}
		})
	}
}
