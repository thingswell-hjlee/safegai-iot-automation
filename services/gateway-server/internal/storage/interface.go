// Package storage defines repository interfaces for the SafeGAI gateway persistence layer.
// Implementations include in-memory (for testing) and SQLite (when driver is available).
package storage

import (
	"context"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// ListOptions provides pagination parameters for list queries.
type ListOptions struct {
	Offset int
	Limit  int
}

// EventStore defines the interface for safety event persistence.
type EventStore interface {
	// InsertEvent stores a new event. Returns error if eventId already exists.
	InsertEvent(ctx context.Context, event *events.SafetyEvent) error

	// GetEvent retrieves an event by its ID.
	GetEvent(ctx context.Context, eventID string) (*events.SafetyEvent, error)

	// ListEvents retrieves events with pagination.
	ListEvents(ctx context.Context, opts ListOptions) ([]*events.SafetyEvent, error)

	// AckEvent acknowledges an event by ID with the given actor and timestamp.
	AckEvent(ctx context.Context, eventID string, actor string, at time.Time) error

	// ResolveEvent resolves an event by ID with the given actor and timestamp.
	ResolveEvent(ctx context.Context, eventID string, actor string, at time.Time) error

	// ClassifyEvent assigns a classification label to an event.
	ClassifyEvent(ctx context.Context, eventID string, classification string) error
}

// AuditStore defines the interface for audit log persistence.
type AuditStore interface {
	// InsertAudit stores a new audit log entry.
	InsertAudit(ctx context.Context, entry *events.AuditEntry) error

	// ListAudits retrieves audit entries with pagination.
	ListAudits(ctx context.Context, opts ListOptions) ([]*events.AuditEntry, error)
}

// OutboxStore defines the interface for the cloud sync outbox queue.
type OutboxStore interface {
	// Enqueue adds a new item to the outbox.
	Enqueue(ctx context.Context, item *events.OutboxItem) error

	// Dequeue retrieves the next pending item without removing it.
	Dequeue(ctx context.Context) (*events.OutboxItem, error)

	// MarkSent marks an outbox item as sent with a timestamp.
	MarkSent(ctx context.Context, itemID string, sentAt time.Time) error

	// GetPending retrieves all pending (unsent) outbox items.
	GetPending(ctx context.Context) ([]*events.OutboxItem, error)

	// GetDepth returns the number of pending items in the outbox.
	GetDepth(ctx context.Context) (int, error)
}

// ConfigStore defines the interface for versioned configuration management.
type ConfigStore interface {
	// GetCurrent retrieves the currently active configuration.
	GetCurrent(ctx context.Context) (*events.ConfigVersion, error)

	// SaveVersion stores a new configuration version.
	SaveVersion(ctx context.Context, cfg *events.ConfigVersion) error

	// ListVersions retrieves all configuration versions.
	ListVersions(ctx context.Context) ([]*events.ConfigVersion, error)
}

// UserStore defines the interface for local user account management.
type UserStore interface {
	// GetUser retrieves a user by username.
	GetUser(ctx context.Context, username string) (*events.User, error)

	// ListUsers retrieves all users.
	ListUsers(ctx context.Context) ([]*events.User, error)

	// CreateUser stores a new user. Returns error if username already exists.
	CreateUser(ctx context.Context, user *events.User) error

	// UpdateUser updates an existing user record.
	UpdateUser(ctx context.Context, user *events.User) error
}

// Store aggregates all repository interfaces into a single access point.
type Store interface {
	EventStore
	AuditStore
	OutboxStore
	ConfigStore
	UserStore
}
