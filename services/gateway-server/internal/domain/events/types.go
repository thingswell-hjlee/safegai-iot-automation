package events

import "time"

// Severity represents the severity level of a safety event.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
	SeverityAlarm    Severity = "ALARM"
)

// OccupancyState represents the occupancy state of a zone.
type OccupancyState string

const (
	OccupancyUnknown         OccupancyState = "UNKNOWN"
	OccupancyOccupied        OccupancyState = "OCCUPIED"
	OccupancyVacantPending   OccupancyState = "VACANT_PENDING"
	OccupancyVacantConfirmed OccupancyState = "VACANT_CONFIRMED"
	OccupancyStale           OccupancyState = "STALE"
)

// EquipmentState represents the operational state of equipment.
type EquipmentState string

const (
	EquipmentRunning EquipmentState = "RUNNING"
	EquipmentStopped EquipmentState = "STOPPED"
	EquipmentFault   EquipmentState = "FAULT"
	EquipmentUnknown EquipmentState = "UNKNOWN"
)

// SafetyEvent represents a full safety event record in the system.
type SafetyEvent struct {
	EventEnvelope

	// Severity is the event severity level.
	Severity Severity `json:"severity"`

	// OccupancyState is the current zone occupancy state.
	OccupancyState OccupancyState `json:"occupancyState"`

	// EquipmentState is the current equipment state.
	EquipmentState EquipmentState `json:"equipmentState"`

	// Actions contains actuation actions taken.
	Actions []string `json:"actions,omitempty"`

	// DetectedAt is the original detection time.
	DetectedAt time.Time `json:"detectedAt"`

	// AckBy is the user who acknowledged the event.
	AckBy string `json:"ackBy,omitempty"`

	// AckAt is when the event was acknowledged.
	AckAt *time.Time `json:"ackAt,omitempty"`

	// ResolvedBy is the user who resolved the event.
	ResolvedBy string `json:"resolvedBy,omitempty"`

	// ResolvedAt is when the event was resolved.
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`

	// Classification is the event classification label.
	Classification string `json:"classification,omitempty"`

	// ImageKey is the S3 key for any associated image.
	ImageKey string `json:"imageKey,omitempty"`

	// CameraID identifies the source camera.
	CameraID string `json:"cameraId,omitempty"`
}

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Role      string    `json:"role"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail,omitempty"`
	IP        string    `json:"ip,omitempty"`
}

// OutboxItem represents a message queued for cloud delivery.
type OutboxItem struct {
	ID         string     `json:"id"`
	EventID    string     `json:"eventId"`
	Payload    []byte     `json:"payload"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	SentAt     *time.Time `json:"sentAt,omitempty"`
	RetryCount int        `json:"retryCount"`
	LastError  string     `json:"lastError,omitempty"`
}

// ConfigVersion represents a versioned configuration snapshot.
type ConfigVersion struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	Active    bool      `json:"active"`
}

// User represents a local user account for the gateway.
type User struct {
	ID                  string     `json:"id"`
	Username            string     `json:"username"`
	PasswordHash        string     `json:"-"`
	Role                string     `json:"role"`
	CreatedAt           time.Time  `json:"createdAt"`
	LastLogin           *time.Time `json:"lastLogin,omitempty"`
	ForcePasswordChange bool       `json:"forcePasswordChange"`
}
