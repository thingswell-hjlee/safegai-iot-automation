// Package memory provides an in-memory implementation of all storage interfaces.
// This is used for testing and development until a real SQLite driver is available.
package memory

import "sync"

// Store is the top-level in-memory store that aggregates all sub-stores.
type Store struct {
	*EventMemoryStore
	*AuditMemoryStore
	*OutboxMemoryStore
	*ConfigMemoryStore
	*UserMemoryStore
}

// NewStore creates a new in-memory store with all sub-stores initialized.
func NewStore() *Store {
	return &Store{
		EventMemoryStore:  NewEventMemoryStore(),
		AuditMemoryStore:  NewAuditMemoryStore(),
		OutboxMemoryStore: NewOutboxMemoryStore(),
		ConfigMemoryStore: NewConfigMemoryStore(),
		UserMemoryStore:   NewUserMemoryStore(),
	}
}

// mu is a helper for embedding sync.Mutex with a short name.
type mu struct {
	sync.RWMutex
}
