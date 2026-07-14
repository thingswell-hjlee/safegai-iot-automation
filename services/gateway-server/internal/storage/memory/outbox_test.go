package memory

import (
	"context"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

func makeTestOutboxItem(id, eventID string) *events.OutboxItem {
	return &events.OutboxItem{
		ID:        id,
		EventID:   eventID,
		Payload:   []byte(`{"test":"data"}`),
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
}

func TestEnqueue_AddsToPending(t *testing.T) {
	store := NewOutboxMemoryStore()
	ctx := context.Background()

	item := makeTestOutboxItem("out-1", "evt-1")
	err := store.Enqueue(ctx, item)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	depth, err := store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("GetDepth failed: %v", err)
	}
	if depth != 1 {
		t.Errorf("expected depth=1, got %d", depth)
	}
}

func TestDequeue_RetrievesInOrder(t *testing.T) {
	store := NewOutboxMemoryStore()
	ctx := context.Background()

	// Enqueue 3 items
	for i := 1; i <= 3; i++ {
		item := makeTestOutboxItem("out-"+itoa(i), "evt-"+itoa(i))
		if err := store.Enqueue(ctx, item); err != nil {
			t.Fatalf("Enqueue %d failed: %v", i, err)
		}
	}

	// Dequeue should return first item
	first, err := store.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if first == nil {
		t.Fatal("expected item, got nil")
	}
	if first.ID != "out-1" {
		t.Errorf("expected ID=out-1, got %s", first.ID)
	}
}

func TestMarkSent_RemovesFromPending(t *testing.T) {
	store := NewOutboxMemoryStore()
	ctx := context.Background()

	item := makeTestOutboxItem("out-sent", "evt-sent")
	if err := store.Enqueue(ctx, item); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	sentAt := time.Now().UTC()
	err := store.MarkSent(ctx, "out-sent", sentAt)
	if err != nil {
		t.Fatalf("MarkSent failed: %v", err)
	}

	// Depth should be 0
	depth, err := store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("GetDepth failed: %v", err)
	}
	if depth != 0 {
		t.Errorf("expected depth=0 after MarkSent, got %d", depth)
	}

	// Dequeue should return nil
	next, err := store.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if next != nil {
		t.Error("expected nil after all items sent")
	}
}

func TestGetDepth_CorrectCount(t *testing.T) {
	store := NewOutboxMemoryStore()
	ctx := context.Background()

	// Empty store
	depth, err := store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("GetDepth failed: %v", err)
	}
	if depth != 0 {
		t.Errorf("expected depth=0, got %d", depth)
	}

	// Add 5 items
	for i := 1; i <= 5; i++ {
		item := makeTestOutboxItem("out-"+itoa(i), "evt-"+itoa(i))
		if err := store.Enqueue(ctx, item); err != nil {
			t.Fatalf("Enqueue %d failed: %v", i, err)
		}
	}

	depth, err = store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("GetDepth failed: %v", err)
	}
	if depth != 5 {
		t.Errorf("expected depth=5, got %d", depth)
	}

	// Mark 2 as sent
	store.MarkSent(ctx, "out-1", time.Now().UTC())
	store.MarkSent(ctx, "out-2", time.Now().UTC())

	depth, err = store.GetDepth(ctx)
	if err != nil {
		t.Fatalf("GetDepth failed: %v", err)
	}
	if depth != 3 {
		t.Errorf("expected depth=3, got %d", depth)
	}
}

func TestGetPending_ReturnsOnlyPending(t *testing.T) {
	store := NewOutboxMemoryStore()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		item := makeTestOutboxItem("out-"+itoa(i), "evt-"+itoa(i))
		if err := store.Enqueue(ctx, item); err != nil {
			t.Fatalf("Enqueue %d failed: %v", i, err)
		}
	}

	// Mark first as sent
	store.MarkSent(ctx, "out-1", time.Now().UTC())

	pending, err := store.GetPending(ctx)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending items, got %d", len(pending))
	}
	if pending[0].ID != "out-2" {
		t.Errorf("expected first pending ID=out-2, got %s", pending[0].ID)
	}
}

func TestMarkSent_NotFound(t *testing.T) {
	store := NewOutboxMemoryStore()
	ctx := context.Background()

	err := store.MarkSent(ctx, "nonexistent", time.Now())
	if err == nil {
		t.Fatal("expected error for nonexistent item")
	}
}
