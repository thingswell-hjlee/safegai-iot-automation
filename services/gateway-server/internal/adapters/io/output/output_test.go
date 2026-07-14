package output

import (
	"testing"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/actuation"
)

// TestOutputExecutorWarningLight verifies warning light output execution.
// SAFETY: Output goes to PLC/Safety Relay input for warning light control.
func TestOutputExecutorWarningLight(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:            "cmd-out-001",
		CorrelationID: "event-001",
		CommandType:   actuation.CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !adapter.GetOutput("plc-relay://zone1/warning-light-1") {
		t.Error("expected warning light to be ON")
	}

	// Turn off.
	cmdOff := cmd
	cmdOff.ID = "cmd-out-002"
	cmdOff.Value = "OFF"
	err = executor.Execute(cmdOff)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if adapter.GetOutput("plc-relay://zone1/warning-light-1") {
		t.Error("expected warning light to be OFF")
	}
}

// TestOutputExecutorStopRequest verifies stop request pulse execution.
// SAFETY CRITICAL: Stop request goes to PLC/Safety Relay ONLY.
// Never direct machine power switching.
func TestOutputExecutorStopRequest(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:              "cmd-out-003",
		CorrelationID:   "event-003",
		CommandType:     actuation.CommandStopRequestPulse,
		TargetAddress:   "plc-relay://zone1/stop-request-input",
		Value:           "PULSE",
		PulseDurationMs: 500,
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	pulse := adapter.GetPulse("plc-relay://zone1/stop-request-input")
	if pulse != 500 {
		t.Errorf("expected 500ms pulse, got %d", pulse)
	}
}

// TestOutputExecutorStopRequestDefaultDuration verifies default pulse duration.
// SAFETY: Default pulse duration (500ms) is used when not specified.
func TestOutputExecutorStopRequestDefaultDuration(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:              "cmd-out-004",
		CorrelationID:   "event-004",
		CommandType:     actuation.CommandStopRequestPulse,
		TargetAddress:   "plc-relay://zone1/stop-request-input",
		PulseDurationMs: 0, // no duration specified
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	pulse := adapter.GetPulse("plc-relay://zone1/stop-request-input")
	if pulse != 500 {
		t.Errorf("expected default 500ms pulse, got %d", pulse)
	}
}

// TestOutputExecutorSiren verifies siren output execution.
// SAFETY: Output goes to PLC/Safety Relay input for siren control.
func TestOutputExecutorSiren(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:            "cmd-out-005",
		CorrelationID: "event-005",
		CommandType:   actuation.CommandSiren,
		TargetAddress: "plc-relay://zone1/siren-1",
		Value:         "ON",
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !adapter.GetOutput("plc-relay://zone1/siren-1") {
		t.Error("expected siren to be ON")
	}
}

// TestOutputExecutorAudioAnnouncement verifies audio announcement execution.
// SAFETY: Output goes to PLC/Safety Relay input for PA system control.
func TestOutputExecutorAudioAnnouncement(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:            "cmd-out-006",
		CorrelationID: "event-006",
		CommandType:   actuation.CommandAudioAnnouncement,
		TargetAddress: "plc-relay://zone1/pa-system",
		Value:         "ON",
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !adapter.GetOutput("plc-relay://zone1/pa-system") {
		t.Error("expected PA system to be ON")
	}
}

// TestOutputExecutorIOFailure verifies behavior when I/O fails.
// SAFETY: I/O failure is reported as error, never treated as success.
func TestOutputExecutorIOFailure(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	adapter.SetFailNext()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:            "cmd-out-007",
		CorrelationID: "event-007",
		CommandType:   actuation.CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
	}

	err := executor.Execute(cmd)
	if err == nil {
		t.Fatal("expected error on I/O failure, got nil")
	}
}

// TestOutputExecutorReadFeedback verifies feedback reading.
// SAFETY: Feedback confirms PLC/Safety Relay received the signal.
func TestOutputExecutorReadFeedback(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	// Execute a command first.
	cmd := actuation.ActuationCommand{
		ID:            "cmd-out-008",
		CorrelationID: "event-008",
		CommandType:   actuation.CommandWarningLight,
		TargetAddress: "plc-relay://zone1/warning-light-1",
		Value:         "ON",
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// Read feedback for executed command.
	feedback, err := executor.ReadFeedback("cmd-out-008")
	if err != nil {
		t.Fatalf("read feedback failed: %v", err)
	}
	// Initially false since we haven't simulated ACK.
	if feedback {
		t.Error("expected feedback=false initially")
	}

	// Read feedback for non-existent command.
	_, err = executor.ReadFeedback("non-existent")
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

// TestOutputExecutorUnsupportedType verifies rejection of unknown command types.
func TestOutputExecutorUnsupportedType(t *testing.T) {
	adapter := NewSimulatedIOAdapter()
	executor := NewOutputExecutor(adapter)

	cmd := actuation.ActuationCommand{
		ID:            "cmd-out-009",
		CorrelationID: "event-009",
		CommandType:   actuation.CommandType("UNKNOWN_TYPE"),
		TargetAddress: "plc-relay://zone1/unknown",
		Value:         "ON",
	}

	err := executor.Execute(cmd)
	if err == nil {
		t.Fatal("expected error for unsupported command type")
	}
}

// TestSimulatedIOAdapterConcurrency verifies thread safety.
func TestSimulatedIOAdapterConcurrency(t *testing.T) {
	adapter := NewSimulatedIOAdapter()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			adapter.WriteDO("addr-1", true)
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		adapter.ReadDI("addr-1")
	}

	<-done
}
