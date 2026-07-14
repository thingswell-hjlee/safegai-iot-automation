package memory

import (
	"context"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// OutboxMemoryStore implements storage.OutboxStore using in-memory slices.
type OutboxMemoryStore struct {
	mu    sync.RWMutex
	items []*events.OutboxItem
}

// NewOutboxMemoryStore creates an initialized OutboxMemoryStore.
func NewOutboxMemoryStore() *OutboxMemoryStore {
	return &OutboxMemoryStore{
		items: make([]*events.OutboxItem, 0),
	}
}

// Enqueue adds a new item to the outbox with status "pending".
func (s *OutboxMemoryStore) Enqueue(_ context.Context, item *events.OutboxItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item.Status = "pending"
	item.CreatedAt = item.CreatedAt.UTC()

	copy := *item
	copy.Payload = make([]byte, len(item.Payload))
	_ = __copy(copy.Payload, item.Payload)
	s.items = append(s.items, &copy)
	return nil
}

// __copy is a wrapper to allow the use of builtin copy.
func __copy(dst, src []byte) int {
	return copy(dst, src)
}

// Dequeue retrieves the next pending item without removing it (peek).
// Returns nil, nil if no pending items exist.
func (s *OutboxMemoryStore) Dequeue(_ context.Context) (*events.OutboxItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items {
		if item.Status == "pending" {
			copy := *item
			copy.Payload = make([]byte, len(item.Payload))
			_ = __copy(copy.Payload, item.Payload)
			return &copy, nil
		}
	}
	return nil, nil
}

// MarkSent marks an outbox item as sent with the given timestamp.
func (s *OutboxMemoryStore) MarkSent(_ context.Context, itemID string, sentAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range s.items {
		if item.ID == itemID {
			utcSent := sentAt.UTC()
			item.Status = "sent"
			item.SentAt = &utcSent
			return nil
		}
	}
	return errors.NewNotFoundError("outbox_item", itemID)
}

// GetPending retrieves all pending (unsent) outbox items in order.
func (s *OutboxMemoryStore) GetPending(_ context.Context) ([]*events.OutboxItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*events.OutboxItem, 0)
	for _, item := range s.items {
		if item.Status == "pending" {
			copy := *item
			copy.Payload = make([]byte, len(item.Payload))
			_ = __copy(copy.Payload, item.Payload)
			result = append(result, &copy)
		}
	}
	return result, nil
}

// GetDepth returns the number of pending items in the outbox.
func (s *OutboxMemoryStore) GetDepth(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, item := range s.items {
		if item.Status == "pending" {
			count++
		}
	}
	return count, nil
}
