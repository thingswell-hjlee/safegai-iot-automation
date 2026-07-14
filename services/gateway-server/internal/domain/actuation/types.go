// Package actuation implements the output and alarm actuation service for the
// SafeGAI edge gateway. This is an R3 safety-critical module.
//
// SAFETY: All output commands target PLC/Safety Relay inputs ONLY.
// General-purpose digital output (DO) does NOT switch machine power directly.
// Stop request pulses are sent to the PLC/Safety Relay which controls the
// actual equipment power circuit. This gateway never has direct control over
// machine power.
//
// REPLAY GUARD: After restart, past pulse commands are NOT replayed.
// Commands created before boot time are rejected.
//
// RETRY: Retry is bounded (default max 3 attempts). No infinite retry on
// missing ACK. ACK timeout is NOT permission to retry forever.
//
// DEDUP: Duplicate output suppression for the same correlationId+commandType
// within the suppression window.
package actuation

import "time"

// CommandType identifies the type of actuation command.
// These commands target PLC/Safety Relay inputs, never machine power directly.
type CommandType string

const (
	// CommandWarningLight activates a visual warning indicator.
	// Output goes to PLC/Safety Relay input for warning light control.
	CommandWarningLight CommandType = "WARNING_LIGHT"

	// CommandSiren activates an audible alarm.
	// Output goes to PLC/Safety Relay input for siren control.
	CommandSiren CommandType = "SIREN"

	// CommandStopRequestPulse sends a stop request pulse to PLC/Safety Relay.
	// SAFETY CRITICAL: This goes to PLC/Safety Relay input ONLY.
	// The PLC/Safety Relay decides whether to actually stop the equipment.
	// This gateway does NOT directly switch machine power.
	CommandStopRequestPulse CommandType = "STOP_REQUEST_PULSE"

	// CommandAudioAnnouncement triggers an audio announcement.
	// Output goes to PLC/Safety Relay input for PA system control.
	CommandAudioAnnouncement CommandType = "AUDIO_ANNOUNCEMENT"
)

// ValidCommandTypes contains all valid CommandType values.
var ValidCommandTypes = []CommandType{
	CommandWarningLight,
	CommandSiren,
	CommandStopRequestPulse,
	CommandAudioAnnouncement,
}

// IsValid returns true if the CommandType is a recognized constant.
func (ct CommandType) IsValid() bool {
	for _, v := range ValidCommandTypes {
		if ct == v {
			return true
		}
	}
	return false
}

// CommandStatus represents the lifecycle state of an actuation command.
type CommandStatus string

const (
	StatusPending   CommandStatus = "PENDING"
	StatusExecuting CommandStatus = "EXECUTING"
	StatusCompleted CommandStatus = "COMPLETED"
	StatusFailed    CommandStatus = "FAILED"
	StatusTimeout   CommandStatus = "TIMEOUT"
)

// ValidCommandStatuses contains all valid CommandStatus values.
var ValidCommandStatuses = []CommandStatus{
	StatusPending,
	StatusExecuting,
	StatusCompleted,
	StatusFailed,
	StatusTimeout,
}

// IsValid returns true if the CommandStatus is a recognized constant.
func (cs CommandStatus) IsValid() bool {
	for _, v := range ValidCommandStatuses {
		if cs == v {
			return true
		}
	}
	return false
}

// DefaultMaxRetries is the maximum number of retry attempts for a command.
// Retry is bounded; no infinite retry on missing ACK.
const DefaultMaxRetries = 3

// DefaultTimeoutDuration is the default timeout for command execution.
const DefaultTimeoutDuration = 5 * time.Second

// DefaultSuppressionWindow is the default deduplication suppression window.
const DefaultSuppressionWindow = 10 * time.Second

// ActuationCommand represents a single output actuation command.
// Each command targets a PLC/Safety Relay input address, never machine power.
//
// SAFETY: targetAddress refers to a PLC/Safety Relay input channel.
// This gateway does NOT have direct control over machine power circuits.
type ActuationCommand struct {
	// ID is the unique identifier for this command instance.
	ID string

	// CorrelationID links this command to the originating safety event.
	CorrelationID string

	// CommandType identifies what type of actuation to perform.
	CommandType CommandType

	// TargetAddress is the PLC/Safety Relay input address.
	// SAFETY: This is a PLC/Safety Relay input, NOT machine power.
	TargetAddress string

	// Value is the command value (e.g., ON/OFF, intensity level).
	Value string

	// PulseDurationMs is the pulse duration in milliseconds (for pulse commands).
	// Only applicable to STOP_REQUEST_PULSE commands.
	PulseDurationMs int

	// Timeout is the maximum duration to wait for command completion.
	Timeout time.Duration

	// CreatedAt is when this command was created.
	// Used by ReplayGuard to reject commands from before boot.
	CreatedAt time.Time

	// Status is the current lifecycle state of the command.
	Status CommandStatus

	// Result holds the execution result once completed.
	Result *CommandResult

	// RetryCount is the current number of retry attempts.
	RetryCount int

	// MaxRetries is the maximum allowed retry attempts (default 3).
	// Retry is bounded; ACK timeout is NOT permission to retry forever.
	MaxRetries int
}

// CommandResult records the outcome of executing an actuation command.
type CommandResult struct {
	// CommandID references the ActuationCommand.ID this result belongs to.
	CommandID string

	// Success indicates whether the command executed successfully.
	Success bool

	// FeedbackReceived indicates whether feedback/ACK was received from hardware.
	FeedbackReceived bool

	// ExecutedAt is when the command was executed.
	ExecutedAt time.Time

	// LatencyMs is the execution latency in milliseconds.
	LatencyMs int64

	// Error describes any error that occurred during execution.
	Error string
}
