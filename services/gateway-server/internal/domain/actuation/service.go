package actuation

import (
	"fmt"
	"sync"
	"time"

	domainerrors "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
)

// IOExecutor is the interface for executing I/O commands.
// Implementations send commands to PLC/Safety Relay inputs ONLY.
// This interface abstracts real hardware I/O for testability.
//
// SAFETY: Implementations must target PLC/Safety Relay inputs.
// General-purpose DO does NOT switch machine power directly.
// The PLC/Safety Relay controls the actual equipment power circuit.
type IOExecutor interface {
	// Execute sends the command to the PLC/Safety Relay input.
	// Returns error if the I/O operation fails.
	// SAFETY: Output goes to PLC/Safety Relay input, never machine power.
	Execute(cmd ActuationCommand) error

	// ReadFeedback checks whether the PLC/Safety Relay acknowledged the command.
	// Returns true if feedback/ACK was received, false otherwise.
	ReadFeedback(commandID string) (bool, error)
}

// ActuationService manages the execution of output/alarm commands.
// All commands target PLC/Safety Relay inputs, never machine power directly.
//
// SAFETY INVARIANTS:
// - Stop request goes to PLC/Safety Relay ONLY (never direct machine power)
// - No general-purpose DO for machine power switching
// - Each command has: commandId, correlationId, timeout, result
// - Retry is bounded (max 3, no infinite retry on missing ACK)
// - Restart does NOT replay past pulse commands (ReplayGuard)
// - Duplicate output suppression for same event (DedupTracker)
type ActuationService struct {
	mu          sync.Mutex
	executor    IOExecutor
	commands    map[string]*ActuationCommand
	dedup       *DedupTracker
	replayGuard *ReplayGuard
}

// NewActuationService creates a new ActuationService.
// The executor must target PLC/Safety Relay inputs only.
// The replayGuard prevents replaying commands from before the current boot.
// The dedupTracker suppresses duplicate output for the same event.
func NewActuationService(executor IOExecutor, replayGuard *ReplayGuard, dedup *DedupTracker) *ActuationService {
	return &ActuationService{
		executor:    executor,
		commands:    make(map[string]*ActuationCommand),
		dedup:       dedup,
		replayGuard: replayGuard,
	}
}

// ExecuteCommand executes an actuation command.
// The command is sent to the PLC/Safety Relay input via the IOExecutor.
//
// SAFETY: Output goes to PLC/Safety Relay ONLY. Never direct machine power.
// Retry is bounded at MaxRetries (default 3). No infinite retry on missing ACK.
// Commands from before boot time are rejected by the ReplayGuard.
// Duplicate commands for the same correlationId+type within the suppression
// window are blocked by the DedupTracker.
func (s *ActuationService) ExecuteCommand(cmd ActuationCommand) (CommandResult, error) {
	// Validate command.
	if cmd.ID == "" {
		return CommandResult{}, domainerrors.NewValidationError("id", "command ID is required")
	}
	if cmd.CorrelationID == "" {
		return CommandResult{}, domainerrors.NewValidationError("correlationId", "correlation ID is required")
	}
	if !cmd.CommandType.IsValid() {
		return CommandResult{}, domainerrors.NewValidationError("commandType", "invalid command type")
	}
	if cmd.TargetAddress == "" {
		return CommandResult{}, domainerrors.NewValidationError("targetAddress", "target address is required")
	}

	// Set defaults.
	if cmd.MaxRetries == 0 {
		cmd.MaxRetries = DefaultMaxRetries
	}
	if cmd.Timeout == 0 {
		cmd.Timeout = DefaultTimeoutDuration
	}
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now()
	}

	// SAFETY: ReplayGuard prevents replaying commands from before boot.
	// After restart, past pulse commands are NOT replayed.
	if !s.replayGuard.ShouldExecute(cmd) {
		return CommandResult{
			CommandID: cmd.ID,
			Success:   false,
			Error:     "command rejected: created before current boot (replay guard)",
		}, domainerrors.NewValidationError("createdAt", "command predates current boot; replay guard blocked execution")
	}

	// SAFETY: DedupTracker prevents duplicate output for same event.
	if s.dedup.IsDuplicate(cmd.CorrelationID, cmd.CommandType) {
		return CommandResult{
			CommandID: cmd.ID,
			Success:   false,
			Error:     "command rejected: duplicate within suppression window",
		}, domainerrors.NewConflictError("actuation_command", "duplicate command for same correlationId+type within suppression window")
	}

	s.mu.Lock()
	cmd.Status = StatusExecuting
	s.commands[cmd.ID] = &cmd
	s.mu.Unlock()

	// Execute with bounded retry. No infinite retry on missing ACK.
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= cmd.MaxRetries; attempt++ {
		cmd.RetryCount = attempt

		// Check timeout before each attempt.
		if time.Since(startTime) > cmd.Timeout {
			s.mu.Lock()
			cmd.Status = StatusTimeout
			result := CommandResult{
				CommandID:  cmd.ID,
				Success:    false,
				ExecutedAt: time.Now(),
				LatencyMs:  time.Since(startTime).Milliseconds(),
				Error:      fmt.Sprintf("command timed out after %v", cmd.Timeout),
			}
			cmd.Result = &result
			s.commands[cmd.ID] = &cmd
			s.mu.Unlock()
			return result, domainerrors.NewTimeoutError(fmt.Sprintf("actuation command %s", cmd.ID))
		}

		// SAFETY: Execute sends to PLC/Safety Relay input, not machine power.
		err := s.executor.Execute(cmd)
		if err == nil {
			// Success: record the command execution in dedup tracker.
			s.dedup.Record(cmd.CorrelationID, cmd.CommandType)

			s.mu.Lock()
			cmd.Status = StatusCompleted
			result := CommandResult{
				CommandID:  cmd.ID,
				Success:    true,
				ExecutedAt: time.Now(),
				LatencyMs:  time.Since(startTime).Milliseconds(),
			}
			cmd.Result = &result
			s.commands[cmd.ID] = &cmd
			s.mu.Unlock()
			return result, nil
		}

		lastErr = err
	}

	// All retries exhausted. Bounded retry: do not continue.
	s.mu.Lock()
	cmd.Status = StatusFailed
	result := CommandResult{
		CommandID:  cmd.ID,
		Success:    false,
		ExecutedAt: time.Now(),
		LatencyMs:  time.Since(startTime).Milliseconds(),
		Error:      fmt.Sprintf("command failed after %d retries: %v", cmd.MaxRetries, lastErr),
	}
	cmd.Result = &result
	s.commands[cmd.ID] = &cmd
	s.mu.Unlock()

	return result, domainerrors.NewIOFailureError("actuation", fmt.Sprintf("execute command %s", cmd.ID), lastErr)
}

// CheckFeedback verifies whether feedback/ACK was received for a command.
// This queries the IOExecutor for hardware acknowledgment.
//
// SAFETY: Feedback comes from PLC/Safety Relay, confirming it received
// the stop request or alarm activation signal.
func (s *ActuationService) CheckFeedback(commandID string) (bool, error) {
	s.mu.Lock()
	cmd, exists := s.commands[commandID]
	s.mu.Unlock()

	if !exists {
		return false, domainerrors.NewNotFoundError("actuation_command", commandID)
	}

	feedback, err := s.executor.ReadFeedback(commandID)
	if err != nil {
		return false, domainerrors.NewIOFailureError("actuation", fmt.Sprintf("read feedback for %s", commandID), err)
	}

	if feedback && cmd.Result != nil {
		s.mu.Lock()
		cmd.Result.FeedbackReceived = true
		s.mu.Unlock()
	}

	return feedback, nil
}

// GetPendingCommands returns all commands currently in PENDING or EXECUTING state.
// Used for monitoring and diagnostics.
func (s *ActuationService) GetPendingCommands() []ActuationCommand {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pending []ActuationCommand
	for _, cmd := range s.commands {
		if cmd.Status == StatusPending || cmd.Status == StatusExecuting {
			pending = append(pending, *cmd)
		}
	}
	return pending
}

// CancelCommand cancels a pending or executing command.
// Commands that have already completed or failed cannot be cancelled.
func (s *ActuationService) CancelCommand(commandID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, exists := s.commands[commandID]
	if !exists {
		return domainerrors.NewNotFoundError("actuation_command", commandID)
	}

	if cmd.Status == StatusCompleted || cmd.Status == StatusFailed || cmd.Status == StatusTimeout {
		return domainerrors.NewConflictError("actuation_command",
			fmt.Sprintf("cannot cancel command in %s state", cmd.Status))
	}

	cmd.Status = StatusFailed
	cmd.Result = &CommandResult{
		CommandID:  commandID,
		Success:    false,
		ExecutedAt: time.Now(),
		Error:      "command cancelled",
	}

	return nil
}
