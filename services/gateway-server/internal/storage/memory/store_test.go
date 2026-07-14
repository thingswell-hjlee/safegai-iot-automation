package memory

import (
	"context"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

func TestStore_AllInterfaces(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Test EventStore via Store
	event := makeTestEvent("store-evt-1")
	if err := store.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent via Store failed: %v", err)
	}
	got, err := store.GetEvent(ctx, "store-evt-1")
	if err != nil {
		t.Fatalf("GetEvent via Store failed: %v", err)
	}
	if got.EventID != "store-evt-1" {
		t.Errorf("expected eventId store-evt-1, got %s", got.EventID)
	}

	// Test AuditStore via Store
	audit := makeTestAuditEntry("store-aud-1")
	if err := store.InsertAudit(ctx, audit); err != nil {
		t.Fatalf("InsertAudit via Store failed: %v", err)
	}
	audits, err := store.ListAudits(ctx, storage.ListOptions{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("ListAudits via Store failed: %v", err)
	}
	if len(audits) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(audits))
	}

	// Test OutboxStore via Store
	item := makeTestOutboxItem("store-out-1", "store-evt-1")
	if err := store.Enqueue(ctx, item); err != nil {
		t.Fatalf("Enqueue via Store failed: %v", err)
	}
	depth, err := store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("GetDepth via Store failed: %v", err)
	}
	if depth != 1 {
		t.Errorf("expected depth=1, got %d", depth)
	}

	// Test ConfigStore via Store
	cfg := &events.ConfigVersion{
		ID:        "cfg-1",
		Version:   1,
		Content:   `{"key":"value"}`,
		CreatedAt: time.Now().UTC(),
		CreatedBy: "admin",
		Active:    true,
	}
	if err := store.SaveVersion(ctx, cfg); err != nil {
		t.Fatalf("SaveVersion via Store failed: %v", err)
	}
	current, err := store.GetCurrent(ctx)
	if err != nil {
		t.Fatalf("GetCurrent via Store failed: %v", err)
	}
	if current.Version != 1 {
		t.Errorf("expected version=1, got %d", current.Version)
	}

	// Test UserStore via Store
	user := &events.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: "hash123",
		Role:         "OPERATOR",
		CreatedAt:    time.Now().UTC(),
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser via Store failed: %v", err)
	}
	gotUser, err := store.GetUser(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetUser via Store failed: %v", err)
	}
	if gotUser.Role != "OPERATOR" {
		t.Errorf("expected role=OPERATOR, got %s", gotUser.Role)
	}
}
