package memory

import (
	"context"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

// EventMemoryStore implements storage.EventStore using in-memory maps.
type EventMemoryStore struct {
	mu     sync.RWMutex
	events map[string]*events.SafetyEvent
	order  []string // maintains insertion order for listing
}

// NewEventMemoryStore creates an initialized EventMemoryStore.
func NewEventMemoryStore() *EventMemoryStore {
	return &EventMemoryStore{
		events: make(map[string]*events.SafetyEvent),
		order:  make([]string, 0),
	}
}

// InsertEvent stores a new event. Returns ConflictError if eventId already exists.
func (s *EventMemoryStore) InsertEvent(_ context.Context, event *events.SafetyEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.events[event.EventID]; exists {
		return errors.NewConflictError("event", "duplicate eventId: "+event.EventID)
	}

	// Ensure timestamps are UTC
	event.ObservedAt = event.ObservedAt.UTC()
	event.ReceivedAt = event.ReceivedAt.UTC()
	event.DetectedAt = event.DetectedAt.UTC()

	// Store a copy to prevent external mutation
	copy := *event
	s.events[event.EventID] = &copy
	s.order = append(s.order, event.EventID)
	return nil
}

// GetEvent retrieves an event by ID. Returns NotFoundError if not found.
func (s *EventMemoryStore) GetEvent(_ context.Context, eventID string) (*events.SafetyEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event, exists := s.events[eventID]
	if !exists {
		return nil, errors.NewNotFoundError("event", eventID)
	}

	copy := *event
	return &copy, nil
}

// ListEvents retrieves events with pagination following insertion order.
func (s *EventMemoryStore) ListEvents(_ context.Context, opts storage.ListOptions) ([]*events.SafetyEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.order)
	if opts.Offset >= total {
		return []*events.SafetyEvent{}, nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50 // default page size
	}

	end := opts.Offset + limit
	if end > total {
		end = total
	}

	result := make([]*events.SafetyEvent, 0, end-opts.Offset)
	for _, id := range s.order[opts.Offset:end] {
		event := s.events[id]
		copy := *event
		result = append(result, &copy)
	}
	return result, nil
}

// AckEvent acknowledges an event. Returns NotFoundError if event does not exist.
func (s *EventMemoryStore) AckEvent(_ context.Context, eventID string, actor string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, exists := s.events[eventID]
	if !exists {
		return errors.NewNotFoundError("event", eventID)
	}

	utcAt := at.UTC()
	event.AckBy = actor
	event.AckAt = &utcAt
	return nil
}

// ResolveEvent resolves an event. Returns NotFoundError if event does not exist.
func (s *EventMemoryStore) ResolveEvent(_ context.Context, eventID string, actor string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, exists := s.events[eventID]
	if !exists {
		return errors.NewNotFoundError("event", eventID)
	}

	utcAt := at.UTC()
	event.ResolvedBy = actor
	event.ResolvedAt = &utcAt
	return nil
}

// ClassifyEvent assigns a classification label. Returns NotFoundError if event does not exist.
func (s *EventMemoryStore) ClassifyEvent(_ context.Context, eventID string, classification string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, exists := s.events[eventID]
	if !exists {
		return errors.NewNotFoundError("event", eventID)
	}

	event.Classification = classification
	return nil
}
