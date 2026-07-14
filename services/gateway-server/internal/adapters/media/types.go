// Package media provides MediaMTX stream proxy management for the SafeGAI gateway.
// It supports up to 4 simultaneous RTSP camera streams with WebRTC/HLS output.
// NO transcoding, NO recording - codec passthrough only.
package media

import "time"

// MaxStreams is the hard limit on simultaneous camera streams.
// Constrained by memory budget: MediaMTX + 4 streams within 1GB.
const MaxStreams = 4

// StreamState represents the current state of a media stream.
type StreamState string

const (
	StreamStateActive   StreamState = "ACTIVE"
	StreamStateInactive StreamState = "INACTIVE"
	StreamStateError    StreamState = "ERROR"
)

// Protocol defines supported input protocols.
type Protocol string

const (
	ProtocolRTSP Protocol = "RTSP"
)

// Codec defines supported video codecs.
type Codec string

const (
	CodecH264 Codec = "H264"
)

// OutputProtocol defines output protocols for client consumption.
type OutputProtocol string

const (
	OutputProtocolWebRTC OutputProtocol = "WebRTC" // Preferred: low latency
	OutputProtocolHLS    OutputProtocol = "HLS"    // Fallback: wider compatibility
)

// Resolution represents video resolution.
type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// StreamConfig holds the configuration for a single camera stream.
// sourceURL is a placeholder - actual credentials come from encrypted config.
type StreamConfig struct {
	// ID is the unique identifier for this stream (e.g., "cam-01").
	ID string `json:"id"`

	// Name is a human-readable label (e.g., "Zone A Camera 1").
	Name string `json:"name"`

	// SourceURL is the RTSP source URL placeholder.
	// Actual camera credentials are loaded from local encrypted config at runtime.
	// NEVER hardcode credentials in source or config files.
	SourceURL string `json:"sourceUrl"`

	// Protocol is the input protocol (always RTSP for cameras).
	Protocol Protocol `json:"protocol"`

	// Codec is the video codec (always H264, passthrough only).
	Codec Codec `json:"codec"`

	// Resolution is the stream resolution.
	// 4-split: 640x360 to 1280x720
	// Single full: 1920x1080
	Resolution Resolution `json:"resolution"`

	// FPS is the target frame rate (5-10 fps for 4-split).
	FPS int `json:"fps"`

	// MaxBitrateKbps is the maximum bitrate per channel.
	// 4-split: 768 Kbps max per channel
	// Single full: 2048 Kbps max
	MaxBitrateKbps int `json:"maxBitrateKbps"`

	// OnDemand enables on-demand connection (connect only when viewers exist).
	OnDemand bool `json:"onDemand"`
}

// StreamStatus represents the current operational status of a stream.
type StreamStatus struct {
	// ID is the stream identifier.
	ID string `json:"id"`

	// Name is the human-readable stream label.
	Name string `json:"name"`

	// State is the current stream state (ACTIVE, INACTIVE, ERROR).
	State StreamState `json:"state"`

	// Viewers is the number of active viewers on this stream.
	Viewers int `json:"viewers"`

	// BytesReceived is the total bytes received from the source.
	BytesReceived int64 `json:"bytesReceived"`

	// LastFrameAt is the timestamp of the last received video frame.
	LastFrameAt time.Time `json:"lastFrameAt,omitempty"`

	// Error contains the error message if State is ERROR.
	Error string `json:"error,omitempty"`
}

// MediaHealth represents the overall health of the media proxy subsystem.
type MediaHealth struct {
	// ProcessRunning indicates whether MediaMTX process is alive.
	ProcessRunning bool `json:"processRunning"`

	// ActiveStreams is the number of streams currently receiving data.
	ActiveStreams int `json:"activeStreams"`

	// TotalStreams is the total number of configured streams.
	TotalStreams int `json:"totalStreams"`

	// MemoryUsageMB is the estimated memory usage in megabytes.
	// Budget: MediaMTX + 4 streams within 1GB total.
	MemoryUsageMB int `json:"memoryUsageMB"`

	// Errors contains any current error conditions.
	Errors []string `json:"errors,omitempty"`
}
