package media

// MediaProxy defines the interface for media stream proxy management.
// Implementation wraps MediaMTX configuration and stream lifecycle.
//
// Key constraints:
//   - Maximum 4 simultaneous streams
//   - No transcoding (codec passthrough only)
//   - No recording
//   - On-demand connection supported
//   - Streams reconnect independently
type MediaProxy interface {
	// AddStream adds a new camera stream to the proxy configuration.
	// Returns CapacityLimitError if MaxStreams (4) would be exceeded.
	// Returns ConflictError if a stream with the same ID already exists.
	// Returns ValidationError if the config is invalid.
	AddStream(config StreamConfig) error

	// RemoveStream removes a camera stream by its ID.
	// Returns NotFoundError if the stream does not exist.
	RemoveStream(streamID string) error

	// GetStreamStatus returns the current status of a specific stream.
	// Returns NotFoundError if the stream does not exist.
	GetStreamStatus(streamID string) (StreamStatus, error)

	// GetAllStreams returns the status of all configured streams.
	GetAllStreams() []StreamStatus

	// Health returns the overall health of the media proxy subsystem.
	Health() MediaHealth

	// GenerateConfig generates MediaMTX YAML configuration.
	// The generated config contains stream path definitions but
	// NEVER includes actual camera credentials.
	// Credentials are resolved at runtime from encrypted local config.
	GenerateConfig() ([]byte, error)
}
