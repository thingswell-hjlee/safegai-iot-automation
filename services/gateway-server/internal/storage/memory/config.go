package memory

import (
	"context"
	"sync"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// ConfigMemoryStore implements storage.ConfigStore using in-memory slices.
type ConfigMemoryStore struct {
	mu       sync.RWMutex
	versions []*events.ConfigVersion
}

// NewConfigMemoryStore creates an initialized ConfigMemoryStore.
func NewConfigMemoryStore() *ConfigMemoryStore {
	return &ConfigMemoryStore{
		versions: make([]*events.ConfigVersion, 0),
	}
}

// GetCurrent retrieves the currently active configuration.
// Returns NotFoundError if no active configuration exists.
func (s *ConfigMemoryStore) GetCurrent(_ context.Context) (*events.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.versions) - 1; i >= 0; i-- {
		if s.versions[i].Active {
			copy := *s.versions[i]
			return &copy, nil
		}
	}
	return nil, errors.NewNotFoundError("config", "active")
}

// SaveVersion stores a new configuration version.
// If marked active, all other versions are deactivated.
func (s *ConfigMemoryStore) SaveVersion(_ context.Context, cfg *events.ConfigVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg.CreatedAt = cfg.CreatedAt.UTC()

	if cfg.Active {
		for _, v := range s.versions {
			v.Active = false
		}
	}

	copy := *cfg
	s.versions = append(s.versions, &copy)
	return nil
}

// ListVersions retrieves all configuration versions in order.
func (s *ConfigMemoryStore) ListVersions(_ context.Context) ([]*events.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*events.ConfigVersion, 0, len(s.versions))
	for _, v := range s.versions {
		copy := *v
		result = append(result, &copy)
	}
	return result, nil
}
