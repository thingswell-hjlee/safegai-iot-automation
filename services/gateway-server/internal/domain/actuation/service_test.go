package actuation

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockExecutor is a test implementation of IOExecutor.
// SAFETY: In production, the real executor targets PLC/Safety Relay inputs only.
type mockExecutor struct {
	mu           sync.Mutex
	executeCalls int
	executeErr   error
	feedbackMap  map[string]bool
	feedbackErr  error
	executeDelay time.Duration
	failUntil    int // fail the first N calls
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		feedbackMap: make(map[string]bool),
	}
}

func (m *mockExecutor) Execute(cmd ActuationCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalls++
	if m.executeDelay > 0 {
		m.mu.Unlock()
		time.Sleep(m.executeDelay)
		m.mu.Lock()
	}
	if m.failUntil > 0 && m.executeCalls <= m.failUntil {
		return m.executeErr
	}
	if m.executeErr != nil && m.failUntil == 0 {
		return m.executeErr
	}
	return nil
}

func (m *mockExecutor) ReadFeedback(commandID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.feedbackErr != nil {
		return false, m.feedbackErr
	}
	return m.feedbackMap[commandID], nil
}

func (m *mockExecutor) getExecuteCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executeCalls
}

// TestExecuteWarningLight verifies successful warning light command execution.
// SAFETY: Output goes to PLC/Safety Relay input for warning light control.
func TestExecuteWarningLight(t *testing.T) {
	executor := newMockExecutor()
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	cmd := ActuationCommand{
		ID:            "cmd-001",
		CorrelationID: "event-001",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
		CreatedAt:     time.Now(),
	}

	result, err := svc.ExecuteCommand(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Error)
	}
	if result.CommandID != "cmd-001" {
		t.Errorf("expected commandId cmd-001, got %s", result.CommandID)
	}
	if result.LatencyMs < 0 {
		t.Errorf("expected non-negative latency, got %d", result.LatencyMs)
	}
	if executor.getExecuteCalls() != 1 {
		t.Errorf("expected 1 execute call, got %d", executor.getExecuteCalls())
	}
}

// TestExecuteStopRequest verifies stop request pulse execution.
// SAFETY: Stop request goes to PLC/Safety Relay ONLY. Never direct machine power.
func TestExecuteStopRequest(t *testing.T) {
	executor := newMockExecutor()
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	cmd := ActuationCommand{
		ID:              "cmd-002",
		CorrelationID:   "event-002",
		CommandType:     CommandStopRequestPulse,
		TargetAddress:   "plc-relay://zone1/stop-request-input",
		Value:           "PULSE",
		PulseDurationMs: 500,
		CreatedAt:       time.Now(),
	}

	result, err := svc.ExecuteCommand(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.Error)
	}
	if result.CommandID != "cmd-002" {
		t.Errorf("expected commandId cmd-002, got %s", result.CommandID)
	}
}

// TestCommandTimeout verifies that a command times out after configured duration.
// SAFETY: Timeout prevents infinite wait. ACK timeout is NOT permission to retry forever.
func TestCommandTimeout(t *testing.T) {
	executor := newMockExecutor()
	executor.executeErr = fmt.Errorf("I/O busy")
	executor.executeDelay = 100 * time.Millisecond
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	cmd := ActuationCommand{
		ID:            "cmd-003",
		CorrelationID: "event-003",
		CommandType:   CommandSiren,
		TargetAddress: "plc-relay://zone1/siren-1",
		Value:         "ON",
		Timeout:       150 * time.Millisecond, // short timeout to trigger
		MaxRetries:    10,                     // high retry, but timeout will stop it
		CreatedAt:     time.Now(),
	}

	result, err := svc.ExecuteCommand(cmd)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if result.Success {
		t.Fatal("expected failure on timeout")
	}
	// The command should not have retried 10 times (timeout stops it)
	calls := executor.getExecuteCalls()
	if calls >= 10 {
		t.Errorf("expected fewer than 10 calls due to timeout, got %d", calls)
	}
}

// TestMaxRetries verifies that retry stops after max attempts (default 3).
// SAFETY: Retry is bounded. No infinite retry on missing ACK.
func TestMaxRetries(t *testing.T) {
	executor := newMockExecutor()
	executor.executeErr = fmt.Errorf("PLC not responding")
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	cmd := ActuationCommand{
		ID:            "cmd-004",
		CorrelationID: "event-004",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone2/warning-light-1",
		Value:         "ON",
		MaxRetries:    3,
		CreatedAt:     time.Now(),
	}

	result, err := svc.ExecuteCommand(cmd)
	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	if result.Success {
		t.Fatal("expected failure after max retries")
	}
	// Should have called execute exactly MaxRetries+1 times (initial + 3 retries)
	calls := executor.getExecuteCalls()
	if calls != 4 {
		t.Errorf("expected 4 execute calls (1 initial + 3 retries), got %d", calls)
	}
}

// TestFeedbackVerification verifies optional feedback confirmation.
// SAFETY: Feedback comes from PLC/Safety Relay confirming it received the signal.
func TestFeedbackVerification(t *testing.T) {
	executor := newMockExecutor()
	executor.feedbackMap["cmd-005"] = true
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	// First execute the command.
	cmd := ActuationCommand{
		ID:            "cmd-005",
		CorrelationID: "event-005",
		CommandType:   CommandStopRequestPulse,
		TargetAddress: "plc-relay://zone1/stop-request-input",
		Value:         "PULSE",
		CreatedAt:     time.Now(),
	}

	_, err := svc.ExecuteCommand(cmd)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// Check feedback.
	feedback, err := svc.CheckFeedback("cmd-005")
	if err != nil {
		t.Fatalf("check feedback failed: %v", err)
	}
	if !feedback {
		t.Error("expected feedback=true, got false")
	}

	// Check non-existent command.
	_, err = svc.CheckFeedback("non-existent")
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

// TestConcurrentCommands verifies that multiple simultaneous commands execute safely.
// SAFETY: Thread safety is critical for safety-critical actuation.
func TestConcurrentCommands(t *testing.T) {
	executor := newMockExecutor()
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	var wg sync.WaitGroup
	var successCount atomic.Int32
	numCommands := 10

	for i := 0; i < numCommands; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd := ActuationCommand{
				ID:            fmt.Sprintf("cmd-concurrent-%d", idx),
				CorrelationID: fmt.Sprintf("event-concurrent-%d", idx),
				CommandType:   CommandWarningLight,
				TargetAddress: fmt.Sprintf("plc-relay://zone%d/warning-light-1", idx),
				Value:         "ON",
				CreatedAt:     time.Now(),
			}
			result, err := svc.ExecuteCommand(cmd)
			if err == nil && result.Success {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if int(successCount.Load()) != numCommands {
		t.Errorf("expected %d successful commands, got %d", numCommands, successCount.Load())
	}
}

// TestCancelCommand verifies command cancellation.
func TestCancelCommand(t *testing.T) {
	executor := newMockExecutor()
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	// Execute a command first.
	cmd := ActuationCommand{
		ID:            "cmd-cancel-001",
		CorrelationID: "event-cancel-001",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
		CreatedAt:     time.Now(),
	}

	_, err := svc.ExecuteCommand(cmd)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// Try to cancel a completed command (should fail).
	err = svc.CancelCommand("cmd-cancel-001")
	if err == nil {
		t.Error("expected error cancelling completed command")
	}

	// Try to cancel non-existent command.
	err = svc.CancelCommand("non-existent")
	if err == nil {
		t.Error("expected error cancelling non-existent command")
	}
}

// TestGetPendingCommands verifies retrieval of pending commands.
func TestGetPendingCommands(t *testing.T) {
	executor := newMockExecutor()
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	// Execute a command (will complete immediately with mock).
	cmd := ActuationCommand{
		ID:            "cmd-pending-001",
		CorrelationID: "event-pending-001",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
		CreatedAt:     time.Now(),
	}

	_, err := svc.ExecuteCommand(cmd)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// No commands should be pending after successful execution.
	pending := svc.GetPendingCommands()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending commands, got %d", len(pending))
	}
}

// TestValidationErrors verifies that invalid commands are rejected.
func TestValidationErrors(t *testing.T) {
	executor := newMockExecutor()
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	tests := []struct {
		name string
		cmd  ActuationCommand
	}{
		{
			name: "missing ID",
			cmd: ActuationCommand{
				CorrelationID: "event-001",
				CommandType:   CommandWarningLight,
				TargetAddress: "plc-relay://zone1/light",
				CreatedAt:     time.Now(),
			},
		},
		{
			name: "missing correlationId",
			cmd: ActuationCommand{
				ID:            "cmd-001",
				CommandType:   CommandWarningLight,
				TargetAddress: "plc-relay://zone1/light",
				CreatedAt:     time.Now(),
			},
		},
		{
			name: "invalid command type",
			cmd: ActuationCommand{
				ID:            "cmd-001",
				CorrelationID: "event-001",
				CommandType:   CommandType("INVALID"),
				TargetAddress: "plc-relay://zone1/light",
				CreatedAt:     time.Now(),
			},
		},
		{
			name: "missing target address",
			cmd: ActuationCommand{
				ID:            "cmd-001",
				CorrelationID: "event-001",
				CommandType:   CommandWarningLight,
				CreatedAt:     time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ExecuteCommand(tt.cmd)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

// TestRetrySucceedsOnLaterAttempt verifies bounded retry with eventual success.
func TestRetrySucceedsOnLaterAttempt(t *testing.T) {
	executor := newMockExecutor()
	executor.executeErr = fmt.Errorf("temporary failure")
	executor.failUntil = 2 // fail first 2, succeed on 3rd
	rg := NewReplayGuard()
	dedup := NewDedupTracker(DefaultSuppressionWindow)
	svc := NewActuationService(executor, rg, dedup)

	cmd := ActuationCommand{
		ID:            "cmd-retry-001",
		CorrelationID: "event-retry-001",
		CommandType:   CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
		MaxRetries:    3,
		CreatedAt:     time.Now(),
	}

	result, err := svc.ExecuteCommand(cmd)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success after retry")
	}
	calls := executor.getExecuteCalls()
	if calls != 3 {
		t.Errorf("expected 3 execute calls (2 fails + 1 success), got %d", calls)
	}
}
