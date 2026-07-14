package memory

import (
	"context"
	"sync"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

// AuditMemoryStore implements storage.AuditStore using in-memory slices.
type AuditMemoryStore struct {
	mu      sync.RWMutex
	entries []*events.AuditEntry
}

// NewAuditMemoryStore creates an initialized AuditMemoryStore.
func NewAuditMemoryStore() *AuditMemoryStore {
	return &AuditMemoryStore{
		entries: make([]*events.AuditEntry, 0),
	}
}

// InsertAudit stores a new audit log entry with timestamp in UTC.
func (s *AuditMemoryStore) InsertAudit(_ context.Context, entry *events.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure timestamp is UTC
	entry.Timestamp = entry.Timestamp.UTC()

	copy := *entry
	s.entries = append(s.entries, &copy)
	return nil
}

// ListAudits retrieves audit entries with pagination.
func (s *AuditMemoryStore) ListAudits(_ context.Context, opts storage.ListOptions) ([]*events.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.entries)
	if opts.Offset >= total {
		return []*events.AuditEntry{}, nil
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	end := opts.Offset + limit
	if end > total {
		end = total
	}

	result := make([]*events.AuditEntry, 0, end-opts.Offset)
	for _, entry := range s.entries[opts.Offset:end] {
		copy := *entry
		result = append(result, &copy)
	}
	return result, nil
}
