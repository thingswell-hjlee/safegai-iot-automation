// Package simulator provides a mock MediaProxy implementation for testing.
// It simulates MediaMTX stream proxy behavior without requiring the actual
// MediaMTX process or real camera connections.
//
// This simulator:
//   - Replaces real camera hardware in development and testing
//   - Enforces the same constraints as production (max 4 streams, no transcoding)
//   - Allows controlled stream state transitions for testing failure scenarios
//   - Does NOT connect to real cameras, NVRs, or network video sources
package simulator

import (
	"fmt"
	"sync"
	"time"

	media "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/media"
)

// SimulatedMediaProxy is a mock implementation of the MediaProxy interface.
// It provides deterministic behavior for testing without real infrastructure.
type SimulatedMediaProxy struct {
	mu      sync.RWMutex
	streams map[string]*simulatedStream
	running bool
}

type simulatedStream struct {
	config   media.StreamConfig
	state    media.StreamState
	viewers  int
	bytes    int64
	lastTime time.Time
	errMsg   string
}

// New creates a new SimulatedMediaProxy.
func New() *SimulatedMediaProxy {
	return &SimulatedMediaProxy{
		streams: make(map[string]*simulatedStream),
		running: true,
	}
}

// AddStream adds a simulated stream.
func (s *SimulatedMediaProxy) AddStream(config media.StreamConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("validation error on id: stream ID is required")
	}
	if config.SourceURL == "" {
		return fmt.Errorf("validation error on sourceUrl: source URL is required")
	}

	if _, exists := s.streams[config.ID]; exists {
		return fmt.Errorf("conflict on stream: stream already exists: %s", config.ID)
	}

	if len(s.streams) >= media.MaxStreams {
		return fmt.Errorf("capacity limit on streams (max %d): cannot add more streams", media.MaxStreams)
	}

	s.streams[config.ID] = &simulatedStream{
		config: config,
		state:  media.StreamStateInactive,
	}
	return nil
}

// RemoveStream removes a simulated stream.
func (s *SimulatedMediaProxy) RemoveStream(streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[streamID]; !exists {
		return fmt.Errorf("stream not found: %s", streamID)
	}

	delete(s.streams, streamID)
	return nil
}

// GetStreamStatus returns the status of a simulated stream.
func (s *SimulatedMediaProxy) GetStreamStatus(streamID string) (media.StreamStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ss, exists := s.streams[streamID]
	if !exists {
		return media.StreamStatus{}, fmt.Errorf("stream not found: %s", streamID)
	}

	return media.StreamStatus{
		ID:            ss.config.ID,
		Name:          ss.config.Name,
		State:         ss.state,
		Viewers:       ss.viewers,
		BytesReceived: ss.bytes,
		LastFrameAt:   ss.lastTime,
		Error:         ss.errMsg,
	}, nil
}

// GetAllStreams returns the status of all simulated streams.
func (s *SimulatedMediaProxy) GetAllStreams() []media.StreamStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]media.StreamStatus, 0, len(s.streams))
	for _, ss := range s.streams {
		result = append(result, media.StreamStatus{
			ID:            ss.config.ID,
			Name:          ss.config.Name,
			State:         ss.state,
			Viewers:       ss.viewers,
			BytesReceived: ss.bytes,
			LastFrameAt:   ss.lastTime,
			Error:         ss.errMsg,
		})
	}
	return result
}

// Health returns simulated health status.
func (s *SimulatedMediaProxy) Health() media.MediaHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := media.MediaHealth{
		ProcessRunning: s.running,
		TotalStreams:   len(s.streams),
	}

	var errs []string
	for _, ss := range s.streams {
		if ss.state == media.StreamStateActive {
			health.ActiveStreams++
		}
		if ss.state == media.StreamStateError && ss.errMsg != "" {
			errs = append(errs, ss.errMsg)
		}
	}
	health.Errors = errs
	health.MemoryUsageMB = 50 + (health.ActiveStreams * 200)

	return health
}

// GenerateConfig generates a simulated MediaMTX YAML configuration.
func (s *SimulatedMediaProxy) GenerateConfig() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := media.NewMediaMTXConfig()
	for _, ss := range s.streams {
		if err := cfg.AddStream(ss.config); err != nil {
			return nil, err
		}
	}
	return cfg.GenerateYAML()
}

// --- Test helper methods (not part of MediaProxy interface) ---

// SimulateActivate marks a stream as active (simulates successful connection).
func (s *SimulatedMediaProxy) SimulateActivate(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ss, exists := s.streams[streamID]; exists {
		ss.state = media.StreamStateActive
		ss.lastTime = time.Now()
		ss.errMsg = ""
	}
}

// SimulateError marks a stream as having an error (simulates connection failure).
func (s *SimulatedMediaProxy) SimulateError(streamID string, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ss, exists := s.streams[streamID]; exists {
		ss.state = media.StreamStateError
		ss.errMsg = errMsg
	}
}

// SimulateData simulates receiving data on a stream.
func (s *SimulatedMediaProxy) SimulateData(streamID string, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ss, exists := s.streams[streamID]; exists {
		ss.bytes += bytes
		ss.lastTime = time.Now()
	}
}

// SimulateViewerJoin adds a viewer to a stream.
func (s *SimulatedMediaProxy) SimulateViewerJoin(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ss, exists := s.streams[streamID]; exists {
		ss.viewers++
	}
}

// SimulateViewerLeave removes a viewer from a stream.
func (s *SimulatedMediaProxy) SimulateViewerLeave(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ss, exists := s.streams[streamID]; exists {
		if ss.viewers > 0 {
			ss.viewers--
		}
	}
}

// SimulateProcessStop simulates MediaMTX process stopping.
func (s *SimulatedMediaProxy) SimulateProcessStop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
}

// SimulateProcessStart simulates MediaMTX process starting.
func (s *SimulatedMediaProxy) SimulateProcessStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
}
