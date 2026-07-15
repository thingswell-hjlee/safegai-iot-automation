// Package camera defines the adapter interface for camera devices.
// Camera adapters provide event streams, health monitoring, and snapshot capture.
// Implementations include the simulator (for testing) and real camera SDKs.
package camera

import "context"

// CameraAdapter is the primary interface for camera device integration.
// All camera implementations (real hardware, simulator) must satisfy this interface.
type CameraAdapter interface {
	// Connect establishes a connection to the camera device.
	// Returns an error if the connection cannot be established.
	Connect(ctx context.Context) error

	// Health returns the current health status of the camera.
	Health() CameraHealth

	// SubscribeEvents starts streaming camera events to the provided channel.
	// The channel is owned by the caller; the adapter writes events to it.
	// Returns an error if subscription cannot be started.
	// The subscription runs until the context is cancelled or Close is called.
	SubscribeEvents(ctx context.Context, ch chan<- RawCameraEvent) error

	// GetSnapshot captures a snapshot image for the specified zone.
	// Returns the raw image bytes (JPEG) or an error.
	GetSnapshot(ctx context.Context, zoneID string) ([]byte, error)

	// GetCapabilities returns the capabilities of the camera device.
	GetCapabilities() Capabilities

	// Close terminates the connection and releases resources.
	Close() error
}
