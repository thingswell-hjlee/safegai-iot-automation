package memory

import (
	"context"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

func makeTestAuditEntry(id string) *events.AuditEntry {
	return &events.AuditEntry{
		ID:        id,
		Timestamp: time.Now().UTC(),
		Actor:     "operator1",
		Role:      "OPERATOR",
		Action:    "ACK_EVENT",
		Target:    "evt-001",
		Detail:    "acknowledged event",
		IP:        "192.168.1.100",
	}
}

func TestInsertAudit_Success(t *testing.T) {
	store := NewAuditMemoryStore()
	ctx := context.Background()

	entry := makeTestAuditEntry("aud-001")
	err := store.InsertAudit(ctx, entry)
	if err != nil {
		t.Fatalf("InsertAudit failed: %v", err)
	}

	entries, err := store.ListAudits(ctx, storage.ListOptions{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("ListAudits failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "aud-001" {
		t.Errorf("expected ID=aud-001, got %s", entries[0].ID)
	}
	if entries[0].Actor != "operator1" {
		t.Errorf("expected actor=operator1, got %s", entries[0].Actor)
	}
}

func TestListAudits_Pagination(t *testing.T) {
	store := NewAuditMemoryStore()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		entry := makeTestAuditEntry("aud-" + itoa(i))
		if err := store.InsertAudit(ctx, entry); err != nil {
			t.Fatalf("InsertAudit %d failed: %v", i, err)
		}
	}

	// Page 1
	page1, err := store.ListAudits(ctx, storage.ListOptions{Offset: 0, Limit: 3})
	if err != nil {
		t.Fatalf("ListAudits failed: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(page1))
	}

	// Page past end
	empty, err := store.ListAudits(ctx, storage.ListOptions{Offset: 100, Limit: 10})
	if err != nil {
		t.Fatalf("ListAudits failed: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(empty))
	}
}

func TestInsertAudit_TimestampUTC(t *testing.T) {
	store := NewAuditMemoryStore()
	ctx := context.Background()

	entry := makeTestAuditEntry("aud-utc")
	loc := time.FixedZone("KST", 9*60*60)
	entry.Timestamp = time.Now().In(loc)

	if err := store.InsertAudit(ctx, entry); err != nil {
		t.Fatalf("InsertAudit failed: %v", err)
	}

	entries, err := store.ListAudits(ctx, storage.ListOptions{Offset: 0, Limit: 1})
	if err != nil {
		t.Fatalf("ListAudits failed: %v", err)
	}
	if entries[0].Timestamp.Location() != time.UTC {
		t.Errorf("expected UTC timestamp, got %s", entries[0].Timestamp.Location())
	}
}
