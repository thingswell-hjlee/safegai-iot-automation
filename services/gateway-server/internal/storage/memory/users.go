package memory

import (
	"context"
	"sync"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// UserMemoryStore implements storage.UserStore using in-memory maps.
type UserMemoryStore struct {
	mu    sync.RWMutex
	users map[string]*events.User // keyed by username
}

// NewUserMemoryStore creates an initialized UserMemoryStore.
func NewUserMemoryStore() *UserMemoryStore {
	return &UserMemoryStore{
		users: make(map[string]*events.User),
	}
}

// GetUser retrieves a user by username. Returns NotFoundError if not found.
func (s *UserMemoryStore) GetUser(_ context.Context, username string) (*events.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, errors.NewNotFoundError("user", username)
	}

	copy := *user
	return &copy, nil
}

// ListUsers retrieves all users.
func (s *UserMemoryStore) ListUsers(_ context.Context) ([]*events.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*events.User, 0, len(s.users))
	for _, u := range s.users {
		copy := *u
		result = append(result, &copy)
	}
	return result, nil
}

// CreateUser stores a new user. Returns ConflictError if username already exists.
func (s *UserMemoryStore) CreateUser(_ context.Context, user *events.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Username]; exists {
		return errors.NewConflictError("user", "username already exists: "+user.Username)
	}

	user.CreatedAt = user.CreatedAt.UTC()
	copy := *user
	s.users[user.Username] = &copy
	return nil
}

// UpdateUser updates an existing user record. Returns NotFoundError if not found.
func (s *UserMemoryStore) UpdateUser(_ context.Context, user *events.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Username]; !exists {
		return errors.NewNotFoundError("user", user.Username)
	}

	copy := *user
	s.users[user.Username] = &copy
	return nil
}
