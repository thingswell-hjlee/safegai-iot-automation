// Package ports defines the primary port interfaces for the SafeGAI gateway.
// These interfaces decouple the domain/application layer from external systems.
// Each port has one or more adapter implementations:
//   - Simulator adapters for testing and AWS simulation
//   - Physical adapters for real hardware
//   - Disabled adapters for offline operation
//
// ARCHITECTURE: The gateway binary is identical across all environments.
// Only the adapter selection (via configuration profile) changes behavior.
// Domain and application layers MUST NOT import any adapter or AWS SDK directly.
package ports

import (
	"context"
	"time"
)

// CameraEvent represents a raw event from a camera system.
type CameraEvent struct {
	CameraID    string    `json:"cameraId"`
	ZoneID      string    `json:"zoneId"`
	EventType   string    `json:"eventType"`
	Timestamp   time.Time `json:"timestamp"`
	PersonCount int       `json:"personCount,omitempty"`
	Confidence  float64   `json:"confidence,omitempty"`
	FrameID     string    `json:"frameId,omitempty"`
	SequenceNo  int64     `json:"sequenceNo"`
}

// CameraHealth represents camera device health status.
type CameraHealth struct {
	CameraID string    `json:"cameraId"`
	Online   bool      `json:"online"`
	FPS      int       `json:"fps,omitempty"`
	LastSeen time.Time `json:"lastSeen"`
	ErrorMsg string    `json:"errorMsg,omitempty"`
}

// CameraPort is the interface for camera device integration.
// Adapters: SimulatedCameraAdapter, GenericHttpCameraAdapter, VendorCameraAdapter
type CameraPort interface {
	// Start begins the camera adapter lifecycle.
	Start(ctx context.Context) error

	// SubscribeEvents returns a channel of camera events.
	SubscribeEvents(ctx context.Context) (<-chan CameraEvent, error)

	// GetHealth returns health status for all managed cameras.
	GetHealth() []CameraHealth

	// GetSnapshot captures a snapshot from the specified camera/zone.
	GetSnapshot(ctx context.Context, cameraID string, zoneID string) ([]byte, error)

	// Stop gracefully shuts down the camera adapter.
	Stop() error
}

// SensorReading represents a reading from an environmental sensor.
type SensorReading struct {
	SensorID   string    `json:"sensorId"`
	SensorType string    `json:"sensorType"` // temperature, humidity, co2, gas, vibration, current
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Quality    string    `json:"quality"`
	Timestamp  time.Time `json:"timestamp"`
}

// SensorHealth represents sensor device health status.
type SensorHealth struct {
	SensorID string    `json:"sensorId"`
	Online   bool      `json:"online"`
	LastSeen time.Time `json:"lastSeen"`
}

// SensorPort is the interface for environmental sensor integration.
// Adapters: SimulatedSensorAdapter, ModbusSensorAdapter
type SensorPort interface {
	// Start begins polling or subscribing to sensor data.
	Start(ctx context.Context) error

	// GetReadings returns a channel of sensor readings.
	GetReadings(ctx context.Context) (<-chan SensorReading, error)

	// GetHealth returns health status for all sensors.
	GetHealth() []SensorHealth

	// Stop gracefully shuts down the sensor adapter.
	Stop() error
}

// EquipmentState represents the current state of industrial equipment.
type EquipmentStatus struct {
	EquipmentID string    `json:"equipmentId"`
	State       string    `json:"state"` // RUNNING, STOPPED, STARTING, STOPPING, FAULT, OFFLINE, UNKNOWN
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
}

// EquipmentInputPort is the interface for equipment state monitoring.
// Adapters: SimulatedEquipmentAdapter, ModbusEquipmentAdapter
type EquipmentInputPort interface {
	// Start begins polling equipment state.
	Start(ctx context.Context) error

	// GetStatus returns a channel of equipment state changes.
	GetStatus(ctx context.Context) (<-chan EquipmentStatus, error)

	// GetCurrentState returns current state of specified equipment.
	GetCurrentState(equipmentID string) (EquipmentStatus, error)

	// Stop gracefully shuts down the equipment adapter.
	Stop() error
}

// OutputCommand represents a command to an output device.
type OutputCommand struct {
	CommandID     string            `json:"commandId"`
	CorrelationID string            `json:"correlationId"`
	CommandType   string            `json:"commandType"` // WARNING_LIGHT, WARNING_SIREN, VOICE_ANNOUNCE, STOP_REQUEST, DIGITAL_OUTPUT_TEST
	Target        string            `json:"target"`
	Parameters    map[string]string `json:"parameters,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	Timeout       time.Duration     `json:"timeout"`
}

// OutputResult represents the result of an output command execution.
type OutputResult struct {
	CommandID  string    `json:"commandId"`
	Success    bool      `json:"success"`
	ExecutedAt time.Time `json:"executedAt"`
	ErrorMsg   string    `json:"errorMsg,omitempty"`
}

// OutputPort is the interface for output device control.
// SAFETY: STOP_REQUEST is sent to PLC or Safety Relay only, never direct machine power.
// Adapters: SimulatedOutputAdapter, ModbusTcpOutputAdapter, ModbusRtuOutputAdapter, CameraDioOutputAdapter
type OutputPort interface {
	// Start initializes the output adapter.
	Start(ctx context.Context) error

	// Execute sends an output command and returns the result.
	// SAFETY: Commands are NOT replayed after restart (ReplayGuard).
	Execute(ctx context.Context, cmd OutputCommand) (OutputResult, error)

	// GetCapabilities returns supported command types.
	GetCapabilities() []string

	// Stop gracefully shuts down the output adapter.
	Stop() error
}

// MediaStreamConfig represents configuration for a single media stream.
type MediaStreamConfig struct {
	StreamID  string `json:"streamId"`
	CameraID  string `json:"cameraId"`
	SourceURL string `json:"sourceUrl,omitempty"`
	Protocol  string `json:"protocol"` // rtsp, synthetic
}

// MediaPort is the interface for media stream management.
// Adapters: SimulatedMediaAdapter, MediaMTXAdapter
type MediaPort interface {
	// Start initializes media stream management.
	Start(ctx context.Context) error

	// AddStream adds a stream to the proxy.
	AddStream(config MediaStreamConfig) error

	// RemoveStream removes a stream by ID.
	RemoveStream(streamID string) error

	// GetStreamStatus returns status of all streams.
	GetStreamStatus() map[string]string

	// Stop gracefully shuts down media management.
	Stop() error
}

// CloudMessage represents a message to sync to cloud.
type CloudMessage struct {
	ID        string    `json:"id"`
	Topic     string    `json:"topic"`
	Payload   []byte    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// CloudSyncPort is the interface for cloud connectivity.
// Adapters: DisabledCloudAdapter, AwsIoTCloudAdapter
type CloudSyncPort interface {
	// Start initializes cloud connectivity.
	Start(ctx context.Context) error

	// Publish sends a message to the cloud.
	Publish(ctx context.Context, msg CloudMessage) error

	// IsConnected returns current cloud connectivity status.
	IsConnected() bool

	// Stop gracefully disconnects from cloud.
	Stop() error
}

// Notification represents an alert or notification.
type Notification struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"` // info, warning, critical
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

// NotificationPort is the interface for sending notifications.
// Adapters: LogNotificationAdapter, SNSNotificationAdapter
type NotificationPort interface {
	// Send delivers a notification.
	Send(ctx context.Context, n Notification) error
}

// ClockPort is the interface for time operations.
// Allows testing with deterministic time.
// Adapters: SystemClockAdapter, MockClockAdapter
type ClockPort interface {
	// Now returns the current time.
	Now() time.Time

	// Since returns the duration since the given time.
	Since(t time.Time) time.Duration

	// NewTimer creates a timer that fires after duration d.
	NewTimer(d time.Duration) Timer
}

// Timer is a minimal timer interface for testability.
type Timer interface {
	// C returns the timer channel.
	C() <-chan time.Time

	// Stop prevents the timer from firing.
	Stop() bool
}

// StoragePort is the interface for persistent storage.
// Adapters: MemoryStorageAdapter (testing), SQLiteStorageAdapter (production)
type StoragePort interface {
	// Open initializes the storage backend.
	Open(ctx context.Context, dsn string) error

	// Close gracefully closes storage connections.
	Close() error

	// Migrate runs schema migrations to the latest version.
	Migrate(ctx context.Context) error

	// SchemaVersion returns the current schema version.
	SchemaVersion(ctx context.Context) (int, error)
}
