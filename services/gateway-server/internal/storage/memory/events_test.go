package memory

import (
	"context"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

func makeTestEvent(id string) *events.SafetyEvent {
	now := time.Now().UTC()
	return &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       id,
			CorrelationID: "corr-" + id,
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-1",
			ZoneID:        "zone-1",
			ObservedAt:    now,
			ReceivedAt:    now,
			SequenceNo:    1,
			Source:        "camera-adapter",
			Quality:       events.QualityGood,
		},
		Severity:       events.SeverityWarning,
		OccupancyState: events.OccupancyOccupied,
		EquipmentState: events.EquipmentRunning,
		DetectedAt:     now,
		CameraID:       "cam-1",
	}
}

func TestInsertEvent_Success(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	event := makeTestEvent("evt-001")
	err := store.InsertEvent(ctx, event)
	if err != nil {
		t.Fatalf("InsertEvent failed: %v", err)
	}

	got, err := store.GetEvent(ctx, "evt-001")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
	if got.EventID != "evt-001" {
		t.Errorf("expected eventId evt-001, got %s", got.EventID)
	}
	if got.Severity != events.SeverityWarning {
		t.Errorf("expected severity WARNING, got %s", got.Severity)
	}
}

func TestInsertEvent_DuplicateRejected(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	event := makeTestEvent("evt-dup")
	err := store.InsertEvent(ctx, event)
	if err != nil {
		t.Fatalf("first InsertEvent failed: %v", err)
	}

	event2 := makeTestEvent("evt-dup")
	err = store.InsertEvent(ctx, event2)
	if err == nil {
		t.Fatal("expected error for duplicate eventId, got nil")
	}

	var conflictErr *errors.ConflictError
	domErr, ok := err.(*errors.ConflictError)
	_ = conflictErr
	if !ok {
		t.Fatalf("expected ConflictError, got %T: %v", err, err)
	}
	if domErr.Code() != errors.CodeConflict {
		t.Errorf("expected CONFLICT code, got %s", domErr.Code())
	}
}

func TestListEvents_Pagination(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	// Insert 10 events
	for i := 0; i < 10; i++ {
		event := makeTestEvent("evt-" + itoa(i))
		event.SequenceNo = int64(i)
		if err := store.InsertEvent(ctx, event); err != nil {
			t.Fatalf("InsertEvent failed for event %d: %v", i, err)
		}
	}

	// Page 1: offset 0, limit 3
	page1, err := store.ListEvents(ctx, storage.ListOptions{Offset: 0, Limit: 3})
	if err != nil {
		t.Fatalf("ListEvents page 1 failed: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 events, got %d", len(page1))
	}
	if page1[0].EventID != "evt-0" {
		t.Errorf("expected first event evt-0, got %s", page1[0].EventID)
	}

	// Page 2: offset 3, limit 3
	page2, err := store.ListEvents(ctx, storage.ListOptions{Offset: 3, Limit: 3})
	if err != nil {
		t.Fatalf("ListEvents page 2 failed: %v", err)
	}
	if len(page2) != 3 {
		t.Fatalf("expected 3 events, got %d", len(page2))
	}
	if page2[0].EventID != "evt-3" {
		t.Errorf("expected first event evt-3, got %s", page2[0].EventID)
	}

	// Page past end: offset 9, limit 5
	pageLast, err := store.ListEvents(ctx, storage.ListOptions{Offset: 9, Limit: 5})
	if err != nil {
		t.Fatalf("ListEvents last page failed: %v", err)
	}
	if len(pageLast) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pageLast))
	}

	// Offset past total
	empty, err := store.ListEvents(ctx, storage.ListOptions{Offset: 100, Limit: 5})
	if err != nil {
		t.Fatalf("ListEvents empty page failed: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 events, got %d", len(empty))
	}
}

func TestAckEvent_UpdatesStatus(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	event := makeTestEvent("evt-ack")
	if err := store.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent failed: %v", err)
	}

	ackTime := time.Now().UTC()
	err := store.AckEvent(ctx, "evt-ack", "operator1", ackTime)
	if err != nil {
		t.Fatalf("AckEvent failed: %v", err)
	}

	got, _ := store.GetEvent(ctx, "evt-ack")
	if got.AckBy != "operator1" {
		t.Errorf("expected ackBy=operator1, got %s", got.AckBy)
	}
	if got.AckAt == nil {
		t.Fatal("expected ackAt to be set")
	}
	if !got.AckAt.Equal(ackTime) {
		t.Errorf("expected ackAt=%v, got %v", ackTime, *got.AckAt)
	}
}

func TestAckEvent_NotFound(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	err := store.AckEvent(ctx, "nonexistent", "user", time.Now())
	if err == nil {
		t.Fatal("expected error for nonexistent event")
	}
	if _, ok := err.(*errors.NotFoundError); !ok {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
}

func TestResolveEvent_UpdatesStatus(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	event := makeTestEvent("evt-resolve")
	if err := store.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent failed: %v", err)
	}

	resolveTime := time.Now().UTC()
	err := store.ResolveEvent(ctx, "evt-resolve", "maintainer1", resolveTime)
	if err != nil {
		t.Fatalf("ResolveEvent failed: %v", err)
	}

	got, _ := store.GetEvent(ctx, "evt-resolve")
	if got.ResolvedBy != "maintainer1" {
		t.Errorf("expected resolvedBy=maintainer1, got %s", got.ResolvedBy)
	}
	if got.ResolvedAt == nil {
		t.Fatal("expected resolvedAt to be set")
	}
	if !got.ResolvedAt.Equal(resolveTime) {
		t.Errorf("expected resolvedAt=%v, got %v", resolveTime, *got.ResolvedAt)
	}
}

func TestClassifyEvent(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	event := makeTestEvent("evt-classify")
	if err := store.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent failed: %v", err)
	}

	err := store.ClassifyEvent(ctx, "evt-classify", "intrusion")
	if err != nil {
		t.Fatalf("ClassifyEvent failed: %v", err)
	}

	got, _ := store.GetEvent(ctx, "evt-classify")
	if got.Classification != "intrusion" {
		t.Errorf("expected classification=intrusion, got %s", got.Classification)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	_, err := store.GetEvent(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent event")
	}
	if _, ok := err.(*errors.NotFoundError); !ok {
		t.Fatalf("expected NotFoundError, got %T", err)
	}
}

func TestInsertEvent_TimestampsUTC(t *testing.T) {
	store := NewEventMemoryStore()
	ctx := context.Background()

	// Create event with non-UTC timestamp
	event := makeTestEvent("evt-utc")
	loc := time.FixedZone("KST", 9*60*60)
	event.ObservedAt = time.Now().In(loc)
	event.ReceivedAt = time.Now().In(loc)
	event.DetectedAt = time.Now().In(loc)

	if err := store.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent failed: %v", err)
	}

	got, _ := store.GetEvent(ctx, "evt-utc")
	if got.ObservedAt.Location() != time.UTC {
		t.Errorf("expected ObservedAt in UTC, got %s", got.ObservedAt.Location())
	}
	if got.ReceivedAt.Location() != time.UTC {
		t.Errorf("expected ReceivedAt in UTC, got %s", got.ReceivedAt.Location())
	}
	if got.DetectedAt.Location() != time.UTC {
		t.Errorf("expected DetectedAt in UTC, got %s", got.DetectedAt.Location())
	}
}

// itoa converts int to string without importing strconv (test helper).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
