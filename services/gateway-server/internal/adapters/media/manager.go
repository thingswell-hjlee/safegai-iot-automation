package media

import (
	"sync"
	"time"

	domainerrors "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
)

// MediaManager implements the MediaProxy interface.
// It manages stream lifecycle for up to MaxStreams (4) simultaneous camera streams.
//
// Key design:
//   - Thread-safe via sync.RWMutex
//   - Streams reconnect independently (one failure does not affect others)
//   - No transcoding or recording
//   - On-demand connection supported
//   - Memory budget: MediaMTX + 4 streams within 1GB
type MediaManager struct {
	mu      sync.RWMutex
	streams map[string]*managedStream
	config  *MediaMTXConfig
}

// managedStream wraps a StreamConfig with runtime state.
type managedStream struct {
	config        StreamConfig
	state         StreamState
	viewers       int
	bytesReceived int64
	lastFrameAt   time.Time
	errorMsg      string
}

// NewMediaManager creates a new MediaManager instance.
func NewMediaManager() *MediaManager {
	return &MediaManager{
		streams: make(map[string]*managedStream),
		config:  NewMediaMTXConfig(),
	}
}

// AddStream adds a new camera stream to the proxy configuration.
// Returns CapacityLimitError if MaxStreams (4) would be exceeded.
// Returns ConflictError if a stream with the same ID already exists.
// Returns ValidationError if the config is invalid.
func (m *MediaManager) AddStream(cfg StreamConfig) error {
	if err := validateStreamConfig(cfg); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streams[cfg.ID]; exists {
		return domainerrors.NewConflictError("stream", "stream already exists: "+cfg.ID)
	}

	if len(m.streams) >= MaxStreams {
		return domainerrors.NewCapacityLimitError("streams", MaxStreams,
			"cannot add more streams; maximum capacity reached")
	}

	if err := m.config.AddStream(cfg); err != nil {
		return domainerrors.NewInternalError("failed to add stream to config", err)
	}

	m.streams[cfg.ID] = &managedStream{
		config: cfg,
		state:  StreamStateInactive,
	}

	return nil
}

// RemoveStream removes a camera stream by its ID.
// Returns NotFoundError if the stream does not exist.
func (m *MediaManager) RemoveStream(streamID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streams[streamID]; !exists {
		return domainerrors.NewNotFoundError("stream", streamID)
	}

	delete(m.streams, streamID)
	m.config.RemoveStream(streamID)

	return nil
}

// GetStreamStatus returns the current status of a specific stream.
// Returns NotFoundError if the stream does not exist.
func (m *MediaManager) GetStreamStatus(streamID string) (StreamStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms, exists := m.streams[streamID]
	if !exists {
		return StreamStatus{}, domainerrors.NewNotFoundError("stream", streamID)
	}

	return ms.toStatus(), nil
}

// GetAllStreams returns the status of all configured streams.
func (m *MediaManager) GetAllStreams() []StreamStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]StreamStatus, 0, len(m.streams))
	for _, ms := range m.streams {
		result = append(result, ms.toStatus())
	}
	return result
}

// Health returns the overall health of the media proxy subsystem.
func (m *MediaManager) Health() MediaHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := MediaHealth{
		ProcessRunning: true, // Simulated: in production, check MediaMTX process
		TotalStreams:   len(m.streams),
	}

	var errs []string
	for _, ms := range m.streams {
		if ms.state == StreamStateActive {
			health.ActiveStreams++
		}
		if ms.state == StreamStateError && ms.errorMsg != "" {
			errs = append(errs, ms.errorMsg)
		}
	}
	health.Errors = errs

	// Estimate memory: ~50MB base + ~200MB per active stream
	health.MemoryUsageMB = 50 + (health.ActiveStreams * 200)

	return health
}

// GenerateConfig generates MediaMTX YAML configuration.
// The generated config contains stream path definitions but
// NEVER includes actual camera credentials.
func (m *MediaManager) GenerateConfig() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config.GenerateYAML()
}

// AddCamera adds a camera stream (convenience method using camera ID and stream URL).
func (m *MediaManager) AddCamera(cameraID, streamURL string) error {
	return m.AddStream(StreamConfig{
		ID:             cameraID,
		Name:           "Camera " + cameraID,
		SourceURL:      streamURL,
		Protocol:       ProtocolRTSP,
		Codec:          CodecH264,
		Resolution:     Resolution{Width: 640, Height: 360},
		FPS:            10,
		MaxBitrateKbps: 768,
		OnDemand:       true,
	})
}

// RemoveCamera removes a camera stream by camera ID.
func (m *MediaManager) RemoveCamera(cameraID string) error {
	return m.RemoveStream(cameraID)
}

// GetStatus returns the status of a camera stream.
func (m *MediaManager) GetStatus(cameraID string) StreamStatus {
	status, err := m.GetStreamStatus(cameraID)
	if err != nil {
		return StreamStatus{
			ID:    cameraID,
			State: StreamStateInactive,
			Error: "stream not found",
		}
	}
	return status
}

// Reconnect attempts to reconnect a failed stream.
// Streams reconnect independently; one stream failure does not affect others.
func (m *MediaManager) Reconnect(cameraID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ms, exists := m.streams[cameraID]
	if !exists {
		return domainerrors.NewNotFoundError("stream", cameraID)
	}

	// Reset error state and mark as inactive (will become active on successful connection)
	ms.state = StreamStateInactive
	ms.errorMsg = ""

	return nil
}

// MarkStreamActive marks a stream as actively receiving data.
// Used by the stream monitor when frames are detected.
func (m *MediaManager) MarkStreamActive(streamID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ms, exists := m.streams[streamID]; exists {
		ms.state = StreamStateActive
		ms.lastFrameAt = time.Now()
		ms.errorMsg = ""
	}
}

// MarkStreamError marks a stream as having an error.
// Each stream fails independently; other streams are not affected.
func (m *MediaManager) MarkStreamError(streamID string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ms, exists := m.streams[streamID]; exists {
		ms.state = StreamStateError
		ms.errorMsg = errMsg
	}
}

// UpdateBytesReceived updates the bytes received counter for a stream.
func (m *MediaManager) UpdateBytesReceived(streamID string, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ms, exists := m.streams[streamID]; exists {
		ms.bytesReceived += bytes
		ms.lastFrameAt = time.Now()
	}
}

// StreamCount returns the number of configured streams.
func (m *MediaManager) StreamCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.streams)
}

// toStatus converts a managedStream to a StreamStatus.
func (ms *managedStream) toStatus() StreamStatus {
	return StreamStatus{
		ID:            ms.config.ID,
		Name:          ms.config.Name,
		State:         ms.state,
		Viewers:       ms.viewers,
		BytesReceived: ms.bytesReceived,
		LastFrameAt:   ms.lastFrameAt,
		Error:         ms.errorMsg,
	}
}

// validateStreamConfig validates a StreamConfig for required fields.
func validateStreamConfig(cfg StreamConfig) error {
	if cfg.ID == "" {
		return domainerrors.NewValidationError("id", "stream ID is required")
	}
	if cfg.SourceURL == "" {
		return domainerrors.NewValidationError("sourceUrl", "source URL is required")
	}
	if cfg.Protocol == "" {
		return domainerrors.NewValidationError("protocol", "protocol is required")
	}
	if cfg.Codec == "" {
		return domainerrors.NewValidationError("codec", "codec is required")
	}
	return nil
}
