package outbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage/memory"
)

// mockPublisher implements CloudPublisher for testing.
type mockPublisher struct {
	published []publishedItem
	failCount int // number of calls to fail before succeeding
	callCount int
}

type publishedItem struct {
	key     string
	payload []byte
}

func (m *mockPublisher) Publish(_ context.Context, key string, payload []byte) error {
	m.callCount++
	if m.callCount <= m.failCount {
		return fmt.Errorf("simulated failure %d", m.callCount)
	}
	m.published = append(m.published, publishedItem{key: key, payload: payload})
	return nil
}

func newTestSyncService(store *memory.OutboxMemoryStore, pub *mockPublisher, alertFn AlertFunc) *SyncService {
	return NewSyncService(store, pub, alertFn, &SyncConfig{
		BaseBackoff:  1 * time.Millisecond, // use short backoffs for tests
		MaxBackoff:   60 * time.Second,
		MaxQueueSize: 5,
		SyncInterval: 100 * time.Millisecond,
	})
}

func makeOutboxItem(id, eventID string) *events.OutboxItem {
	return &events.OutboxItem{
		ID:        id,
		EventID:   eventID,
		Payload:   []byte(`{"eventId":"` + eventID + `"}`),
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}
}

func TestSyncOnce_SuccessfulSync(t *testing.T) {
	store := memory.NewOutboxMemoryStore()
	ctx := context.Background()

	// Enqueue 3 items
	for i := 1; i <= 3; i++ {
		item := makeOutboxItem(fmt.Sprintf("item-%d", i), fmt.Sprintf("evt-%d", i))
		store.Enqueue(ctx, item)
	}

	pub := &mockPublisher{}
	svc := newTestSyncService(store, pub, nil)

	sent, err := svc.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}
	if sent != 3 {
		t.Errorf("expected 3 sent, got %d", sent)
	}

	// Verify outbox is empty
	depth, _ := store.GetDepth(ctx)
	if depth != 0 {
		t.Errorf("expected depth=0 after sync, got %d", depth)
	}

	// Verify publisher received items
	if len(pub.published) != 3 {
		t.Errorf("expected 3 published, got %d", len(pub.published))
	}
}

func TestSyncOnce_FailedSyncIncreasesBackoff(t *testing.T) {
	store := memory.NewOutboxMemoryStore()
	ctx := context.Background()

	item := makeOutboxItem("item-fail", "evt-fail")
	store.Enqueue(ctx, item)

	pub := &mockPublisher{failCount: 10} // always fail
	svc := newTestSyncService(store, pub, nil)

	_, err := svc.SyncOnce(ctx)
	if err == nil {
		t.Fatal("expected error from failed publish")
	}

	backoff1 := svc.GetCurrentBackoff()
	if backoff1 == 0 {
		t.Fatal("expected non-zero backoff after failure")
	}

	// Call again - backoff should increase
	_, _ = svc.SyncOnce(ctx)
	backoff2 := svc.GetCurrentBackoff()
	if backoff2 <= backoff1 {
		t.Errorf("expected backoff to increase: %v -> %v", backoff1, backoff2)
	}
}

func TestSyncOnce_MaxBackoffCapped(t *testing.T) {
	store := memory.NewOutboxMemoryStore()
	ctx := context.Background()

	item := makeOutboxItem("item-cap", "evt-cap")
	store.Enqueue(ctx, item)

	pub := &mockPublisher{failCount: 100} // always fail
	maxBackoff := 60 * time.Second
	svc := NewSyncService(store, pub, nil, &SyncConfig{
		BaseBackoff:  1 * time.Millisecond,
		MaxBackoff:   maxBackoff,
		MaxQueueSize: 1000,
		SyncInterval: 100 * time.Millisecond,
	})

	// Run many failures to exceed max
	for i := 0; i < 30; i++ {
		svc.SyncOnce(ctx)
	}

	backoff := svc.GetCurrentBackoff()
	if backoff > maxBackoff {
		t.Errorf("backoff %v exceeds max %v", backoff, maxBackoff)
	}
	if backoff != maxBackoff {
		t.Errorf("expected backoff to reach max %v, got %v", maxBackoff, backoff)
	}
}

func TestSyncOnce_QueueDepthAlert(t *testing.T) {
	store := memory.NewOutboxMemoryStore()
	ctx := context.Background()

	// Enqueue more than threshold
	for i := 1; i <= 10; i++ {
		item := makeOutboxItem(fmt.Sprintf("item-%d", i), fmt.Sprintf("evt-%d", i))
		store.Enqueue(ctx, item)
	}

	alertCalled := false
	var alertDepth int
	alertFn := func(depth int) {
		alertCalled = true
		alertDepth = depth
	}

	pub := &mockPublisher{}
	svc := newTestSyncService(store, pub, alertFn) // maxQueueSize=5

	svc.SyncOnce(ctx)

	if !alertCalled {
		t.Fatal("expected alert to be called when queue exceeds threshold")
	}
	if alertDepth != 10 {
		t.Errorf("expected alert depth=10, got %d", alertDepth)
	}
}

func TestIdempotencyKey_Deterministic(t *testing.T) {
	key1 := IdempotencyKey("evt-1", "item-1")
	key2 := IdempotencyKey("evt-1", "item-1")

	if key1 != key2 {
		t.Errorf("expected deterministic keys, got %s and %s", key1, key2)
	}

	// Different inputs should produce different keys
	key3 := IdempotencyKey("evt-1", "item-2")
	if key1 == key3 {
		t.Error("expected different keys for different inputs")
	}
}

func TestIdempotencyKey_Length(t *testing.T) {
	key := IdempotencyKey("evt-1", "item-1")
	if len(key) != 32 {
		t.Errorf("expected key length=32, got %d", len(key))
	}
}

func TestSyncOnce_EmptyOutbox(t *testing.T) {
	store := memory.NewOutboxMemoryStore()
	ctx := context.Background()

	pub := &mockPublisher{}
	svc := newTestSyncService(store, pub, nil)

	sent, err := svc.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("SyncOnce on empty outbox failed: %v", err)
	}
	if sent != 0 {
		t.Errorf("expected 0 sent for empty outbox, got %d", sent)
	}
}
