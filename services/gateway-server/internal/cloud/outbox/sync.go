// Package outbox provides the cloud sync service with exponential backoff.
package outbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

const (
	// DefaultMaxBackoff is the maximum backoff duration between retries.
	DefaultMaxBackoff = 60 * time.Second

	// DefaultBaseBackoff is the initial backoff duration.
	DefaultBaseBackoff = 1 * time.Second

	// DefaultSyncInterval is the polling interval for the sync loop.
	DefaultSyncInterval = 5 * time.Second

	// DefaultMaxQueueSize triggers an alert when exceeded.
	DefaultMaxQueueSize = 1000
)

// CloudPublisher is the interface for publishing events to the cloud.
// Implementations wrap MQTT, HTTP, or other transport mechanisms.
type CloudPublisher interface {
	// Publish sends a payload to the cloud with an idempotency key.
	// Returns nil on success, error on failure.
	Publish(ctx context.Context, idempotencyKey string, payload []byte) error
}

// AlertFunc is called when the queue depth exceeds the threshold.
type AlertFunc func(depth int)

// SyncService manages background synchronization of the outbox to the cloud.
type SyncService struct {
	store        storage.OutboxStore
	publisher    CloudPublisher
	onAlert      AlertFunc
	maxQueueSize int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
	syncInterval time.Duration

	mu              sync.Mutex
	currentBackoff  time.Duration
	consecutiveFail int
}

// SyncConfig provides configuration options for SyncService.
type SyncConfig struct {
	MaxQueueSize int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
	SyncInterval time.Duration
}

// NewSyncService creates a new SyncService with the given dependencies.
func NewSyncService(store storage.OutboxStore, publisher CloudPublisher, alertFn AlertFunc, cfg *SyncConfig) *SyncService {
	s := &SyncService{
		store:        store,
		publisher:    publisher,
		onAlert:      alertFn,
		maxQueueSize: DefaultMaxQueueSize,
		baseBackoff:  DefaultBaseBackoff,
		maxBackoff:   DefaultMaxBackoff,
		syncInterval: DefaultSyncInterval,
	}
	if cfg != nil {
		if cfg.MaxQueueSize > 0 {
			s.maxQueueSize = cfg.MaxQueueSize
		}
		if cfg.BaseBackoff > 0 {
			s.baseBackoff = cfg.BaseBackoff
		}
		if cfg.MaxBackoff > 0 {
			s.maxBackoff = cfg.MaxBackoff
		}
		if cfg.SyncInterval > 0 {
			s.syncInterval = cfg.SyncInterval
		}
	}
	return s
}

// Start begins the background sync loop. It blocks until ctx is cancelled.
// Uses exponential backoff between sync attempts on failure.
func (s *SyncService) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, err := s.SyncOnce(ctx)

		// Determine wait time based on success/failure
		wait := s.syncInterval
		backoff := s.getBackoff()
		if err != nil && backoff > wait {
			wait = backoff
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// SyncOnce attempts to send all pending outbox items immediately (no waiting).
// Backoff tracking is updated but the actual delay is managed by Start().
// Returns the number of items successfully sent and any error from the last failure.
func (s *SyncService) SyncOnce(ctx context.Context) (sent int, err error) {
	// Check queue depth for alerting
	depth, depthErr := s.store.GetDepth(ctx)
	if depthErr == nil && depth > s.maxQueueSize && s.onAlert != nil {
		s.onAlert(depth)
	}

	pending, err := s.store.GetPending(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending items: %w", err)
	}

	if len(pending) == 0 {
		s.resetBackoff()
		return 0, nil
	}

	var lastErr error
	for _, item := range pending {
		select {
		case <-ctx.Done():
			return sent, ctx.Err()
		default:
		}

		key := IdempotencyKey(item.EventID, item.ID)
		pubErr := s.publisher.Publish(ctx, key, item.Payload)
		if pubErr != nil {
			s.increaseBackoff()
			lastErr = pubErr
			// Stop trying more items after a failure
			return sent, lastErr
		}

		// Mark as sent
		now := time.Now().UTC()
		if markErr := s.store.MarkSent(ctx, item.ID, now); markErr != nil {
			lastErr = markErr
			continue
		}

		sent++
		s.resetBackoff()
	}

	return sent, lastErr
}

// getBackoff returns the current backoff duration.
func (s *SyncService) getBackoff() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentBackoff
}

// increaseBackoff doubles the current backoff, capped at maxBackoff.
func (s *SyncService) increaseBackoff() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consecutiveFail++
	if s.currentBackoff == 0 {
		s.currentBackoff = s.baseBackoff
	} else {
		s.currentBackoff *= 2
	}
	if s.currentBackoff > s.maxBackoff {
		s.currentBackoff = s.maxBackoff
	}
}

// resetBackoff resets backoff to zero after a successful send.
func (s *SyncService) resetBackoff() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentBackoff = 0
	s.consecutiveFail = 0
}

// GetCurrentBackoff returns the current backoff duration (for testing).
func (s *SyncService) GetCurrentBackoff() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentBackoff
}

// IdempotencyKey generates a deterministic idempotency key from event and item IDs.
func IdempotencyKey(eventID, itemID string) string {
	h := sha256.New()
	h.Write([]byte(eventID))
	h.Write([]byte(":"))
	h.Write([]byte(itemID))
	return hex.EncodeToString(h.Sum(nil))[:32]
}
