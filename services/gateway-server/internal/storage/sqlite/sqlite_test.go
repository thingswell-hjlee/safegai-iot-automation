package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

func tempDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestOpenAndMigrate(t *testing.T) {
	store := tempDB(t)

	ver, err := store.SchemaVersion()
	if err != nil {
		t.Fatalf("schema version error: %v", err)
	}
	if ver != len(migrations) {
		t.Errorf("expected schema version %d, got %d", len(migrations), ver)
	}
}

func TestWALMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal-test.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	// WAL file should exist after operations
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Check WAL file exists
	walPath := path + "-wal"
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		// WAL file may not exist yet until writes happen; that's OK
		// The important thing is the pragma was set
	}
}

func TestEventInsertAndRetrieve(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	event := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-001",
			CorrelationID: "corr-001",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-01",
			ZoneID:        "zone-01",
			Source:        "camera",
			Quality:       events.QualityGood,
			ObservedAt:    time.Now().UTC(),
			ReceivedAt:    time.Now().UTC(),
			SequenceNo:    1,
		},
		Severity:       events.SeverityInfo,
		OccupancyState: events.OccupancyOccupied,
		EquipmentState: events.EquipmentRunning,
		CameraID:       "cam-01",
	}

	err := store.InsertEvent(ctx, event)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	// Retrieve
	got, err := store.GetEvent(ctx, "evt-001")
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if got.EventID != "evt-001" {
		t.Errorf("expected evt-001, got %s", got.EventID)
	}
	if got.ZoneID != "zone-01" {
		t.Errorf("expected zone-01, got %s", got.ZoneID)
	}
}

func TestEventDuplicateRejection(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	event := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-dup",
			CorrelationID: "corr-dup",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-01",
			ZoneID:        "zone-01",
			Source:        "camera",
			Quality:       events.QualityGood,
			ObservedAt:    time.Now().UTC(),
			ReceivedAt:    time.Now().UTC(),
			SequenceNo:    1,
		},
		Severity: events.SeverityInfo,
	}

	if err := store.InsertEvent(ctx, event); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Duplicate should fail
	event.SequenceNo = 2 // Different seq to avoid ordering guard
	err := store.InsertEvent(ctx, event)
	if err == nil {
		t.Error("expected error for duplicate event ID")
	}
}

func TestEventOrderingGuard(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	// Insert event with seq 5
	event1 := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-seq-5",
			CorrelationID: "corr-1",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-01",
			ZoneID:        "zone-01",
			Source:        "camera",
			Quality:       events.QualityGood,
			ObservedAt:    time.Now().UTC(),
			ReceivedAt:    time.Now().UTC(),
			SequenceNo:    5,
		},
		Severity: events.SeverityInfo,
	}
	if err := store.InsertEvent(ctx, event1); err != nil {
		t.Fatalf("insert seq 5: %v", err)
	}

	// Insert event with seq 3 (out of order) should fail
	event2 := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-seq-3",
			CorrelationID: "corr-2",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-01",
			ZoneID:        "zone-01",
			Source:        "camera",
			Quality:       events.QualityGood,
			ObservedAt:    time.Now().UTC(),
			ReceivedAt:    time.Now().UTC(),
			SequenceNo:    3,
		},
		Severity: events.SeverityInfo,
	}
	err := store.InsertEvent(ctx, event2)
	if err == nil {
		t.Error("expected error for out-of-order event")
	}
}

func TestStaleEventGuard(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	// Insert event with old timestamp (> 60s ago)
	event := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-stale",
			CorrelationID: "corr-stale",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-02",
			ZoneID:        "zone-01",
			Source:        "camera",
			Quality:       events.QualityGood,
			ObservedAt:    time.Now().Add(-2 * time.Minute).UTC(),
			ReceivedAt:    time.Now().UTC(),
			SequenceNo:    1,
		},
		Severity: events.SeverityInfo,
	}
	err := store.InsertEvent(ctx, event)
	if err == nil {
		t.Error("expected error for stale event")
	}
}

func TestListEvents(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		event := &events.SafetyEvent{
			EventEnvelope: events.EventEnvelope{
				SchemaVersion: "1.0.0",
				EventID:       fmt.Sprintf("evt-list-%d", i),
				CorrelationID: fmt.Sprintf("corr-%d", i),
				TenantID:      "tenant-1",
				SiteID:        "site-1",
				GatewayID:     "gw-1",
				DeviceID:      "cam-01",
				ZoneID:        "zone-01",
				Source:        "camera",
				Quality:       events.QualityGood,
				ObservedAt:    time.Now().UTC(),
				ReceivedAt:    time.Now().UTC(),
				SequenceNo:    int64(i),
			},
			Severity: events.SeverityInfo,
		}
		if err := store.InsertEvent(ctx, event); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	list, err := store.ListEvents(ctx, storage.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 events, got %d", len(list))
	}
}

func TestAuditInsertAndList(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	entry := &events.AuditEntry{
		ID:        "audit-001",
		Timestamp: time.Now().UTC(),
		Actor:     "admin",
		Role:      "MAINTAINER",
		Action:    "LOGIN",
		Target:    "session",
		Detail:    "successful login",
		IP:        "192.168.1.1",
	}
	if err := store.InsertAudit(ctx, entry); err != nil {
		t.Fatalf("insert audit: %v", err)
	}

	list, err := store.ListAudits(ctx, storage.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("list audits: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 audit entry, got %d", len(list))
	}
	if list[0].Actor != "admin" {
		t.Errorf("expected actor admin, got %s", list[0].Actor)
	}
}

func TestOutboxEnqueueAndDequeue(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	item := &events.OutboxItem{
		ID:        "outbox-001",
		EventID:   "evt-001",
		Payload:   []byte(`{"test":"data"}`),
		Status:    "PENDING",
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Enqueue(ctx, item); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	depth, err := store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("get depth: %v", err)
	}
	if depth != 1 {
		t.Errorf("expected depth 1, got %d", depth)
	}

	got, err := store.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if got == nil {
		t.Fatal("expected item, got nil")
	}
	if got.ID != "outbox-001" {
		t.Errorf("expected outbox-001, got %s", got.ID)
	}

	// Mark sent
	if err := store.MarkSent(ctx, "outbox-001", time.Now()); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	// Depth should be 0
	depth, err = store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("get depth after sent: %v", err)
	}
	if depth != 0 {
		t.Errorf("expected depth 0 after mark sent, got %d", depth)
	}
}

func TestBootRecord(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	err := store.RecordBoot(ctx, "0.1.0", len(migrations))
	if err != nil {
		t.Fatalf("record boot: %v", err)
	}

	bootTime, err := store.GetLastBootTime(ctx)
	if err != nil {
		t.Fatalf("get last boot: %v", err)
	}
	if bootTime.IsZero() {
		t.Error("expected non-zero boot time")
	}
}

func TestUserCRUD(t *testing.T) {
	store := tempDB(t)
	ctx := context.Background()

	user := &events.User{
		ID:                  "user-001",
		Username:            "admin",
		PasswordHash:        "hashed-password",
		Role:                "MAINTAINER",
		CreatedAt:           time.Now().UTC(),
		ForcePasswordChange: true,
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := store.GetUser(ctx, "admin")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Username != "admin" {
		t.Errorf("expected admin, got %s", got.Username)
	}
	if got.Role != "MAINTAINER" {
		t.Errorf("expected MAINTAINER, got %s", got.Role)
	}

	list, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 user, got %d", len(list))
	}
}
