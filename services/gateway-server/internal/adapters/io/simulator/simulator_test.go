package simulator

import (
	"context"
	"testing"
	"time"

	ioAdapter "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/io"
	domainErrors "github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/errors"
)

func TestSimulator_Connect_Success(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	err := sim.Connect(ctx, ioAdapter.DefaultIOConfig())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	h := sim.Health()
	if !h.Online {
		t.Fatal("expected health.Online to be true after connect")
	}
}

func TestSimulator_Connect_FailOnConnect(t *testing.T) {
	cfg := Config{FailOnConnect: true}
	sim := New(cfg)
	ctx := context.Background()

	err := sim.Connect(ctx, ioAdapter.DefaultIOConfig())
	if err == nil {
		t.Fatal("expected connection error")
	}

	var connErr *domainErrors.ConnectionError
	if ce, ok := err.(*domainErrors.ConnectionError); !ok {
		t.Fatalf("expected *ConnectionError, got %T: %v", err, err)
	} else {
		connErr = ce
	}
	if !connErr.Retryable() {
		t.Fatal("connection errors should be retryable")
	}

	h := sim.Health()
	if h.Online {
		t.Fatal("health should not be online after failed connect")
	}
}

func TestSimulator_ReadDI_Single(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	// Initially all DI should be false
	val, err := sim.ReadDI(ctx, 0)
	if err != nil {
		t.Fatalf("ReadDI error: %v", err)
	}
	if val {
		t.Fatal("expected DI[0] to be false initially")
	}

	// Set DI[0] to true
	sim.SetDI(0, true)
	val, err = sim.ReadDI(ctx, 0)
	if err != nil {
		t.Fatalf("ReadDI error: %v", err)
	}
	if !val {
		t.Fatal("expected DI[0] to be true after SetDI")
	}
}

func TestSimulator_ReadDI_InvalidAddress(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	_, err := sim.ReadDI(ctx, -1)
	if err == nil {
		t.Fatal("expected error for invalid address -1")
	}

	_, err = sim.ReadDI(ctx, 8)
	if err == nil {
		t.Fatal("expected error for invalid address 8")
	}
}

func TestSimulator_ReadAllDI(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	// Set specific pattern
	sim.SetDI(0, true)
	sim.SetDI(3, true)
	sim.SetDI(7, true)

	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatalf("ReadAllDI error: %v", err)
	}

	if len(states) != ioAdapter.NumDIPoints {
		t.Fatalf("expected %d states, got %d", ioAdapter.NumDIPoints, len(states))
	}

	// Check expected values
	expected := [ioAdapter.NumDIPoints]bool{true, false, false, true, false, false, false, true}
	for i, state := range states {
		if state.Address != i {
			t.Errorf("state[%d].Address = %d, expected %d", i, state.Address, i)
		}
		if state.Value != expected[i] {
			t.Errorf("state[%d].Value = %v, expected %v", i, state.Value, expected[i])
		}
		if state.Quality != ioAdapter.DIQualityGood {
			t.Errorf("state[%d].Quality = %s, expected GOOD", i, state.Quality)
		}
		if state.LastUpdate.IsZero() {
			t.Errorf("state[%d].LastUpdate is zero", i)
		}
	}
}

func TestSimulator_WriteDO(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	cmd := ioAdapter.DOCommand{
		Address:       0,
		Value:         true,
		PulseDuration: 0,
		CommandID:     "cmd-001",
		CorrelationID: "corr-001",
	}

	err := sim.WriteDO(ctx, 0, cmd)
	if err != nil {
		t.Fatalf("WriteDO error: %v", err)
	}

	if !sim.GetDO(0) {
		t.Fatal("expected DO[0] to be true after write")
	}

	log := sim.GetDOLog()
	if len(log) != 1 {
		t.Fatalf("expected 1 DO log entry, got %d", len(log))
	}
	if log[0].CommandID != "cmd-001" {
		t.Errorf("expected CommandID 'cmd-001', got %q", log[0].CommandID)
	}
}

func TestSimulator_WriteDO_InvalidAddress(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	cmd := ioAdapter.DOCommand{Value: true}
	err := sim.WriteDO(ctx, 8, cmd)
	if err == nil {
		t.Fatal("expected error for invalid DO address 8")
	}
}

func TestSimulator_ReadDI_FailOnRead(t *testing.T) {
	cfg := Config{FailOnRead: true, SimulatedLatencyMs: 1}
	sim := New(cfg)
	ctx := context.Background()

	// Override FailOnConnect so we can connect
	sim.SetConfig(Config{FailOnConnect: false, SimulatedLatencyMs: 1})
	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	// Now enable read failure
	sim.SetConfig(Config{FailOnRead: true, SimulatedLatencyMs: 1})

	_, err := sim.ReadDI(ctx, 0)
	if err == nil {
		t.Fatal("expected I/O failure error")
	}

	// I/O failure must not be treated as normal state
	h := sim.Health()
	if h.ErrorCount == 0 {
		t.Fatal("expected error count to increase on I/O failure")
	}
}

func TestSimulator_ReadDI_TimeoutOnRead(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	sim.SetConfig(Config{TimeoutOnRead: true})

	_, err := sim.ReadDI(ctx, 0)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if _, ok := err.(*domainErrors.TimeoutError); !ok {
		t.Fatalf("expected *TimeoutError, got %T", err)
	}
}

func TestSimulator_ReadAllDI_Timeout(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	sim.SetConfig(Config{TimeoutOnRead: true})

	_, err := sim.ReadAllDI(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSimulator_WriteDO_FailOnWrite(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	sim.SetConfig(Config{FailOnWrite: true})

	cmd := ioAdapter.DOCommand{Value: true, CommandID: "cmd-fail"}
	err := sim.WriteDO(ctx, 0, cmd)
	if err == nil {
		t.Fatal("expected I/O failure error on write")
	}
}

func TestSimulator_OperationsAfterClose(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	if err := sim.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	_, err := sim.ReadDI(ctx, 0)
	if err == nil {
		t.Fatal("expected error after close")
	}

	_, err = sim.ReadAllDI(ctx)
	if err == nil {
		t.Fatal("expected error after close")
	}

	cmd := ioAdapter.DOCommand{Value: true}
	err = sim.WriteDO(ctx, 0, cmd)
	if err == nil {
		t.Fatal("expected error after close")
	}

	h := sim.Health()
	if h.Online {
		t.Fatal("health should be offline after close")
	}
}

func TestSimulator_OperationsBeforeConnect(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	_, err := sim.ReadDI(ctx, 0)
	if err == nil {
		t.Fatal("expected error before connect")
	}

	_, err = sim.ReadAllDI(ctx)
	if err == nil {
		t.Fatal("expected error before connect")
	}

	cmd := ioAdapter.DOCommand{Value: true}
	err = sim.WriteDO(ctx, 0, cmd)
	if err == nil {
		t.Fatal("expected error before connect")
	}
}

func TestSimulator_Health_Updates(t *testing.T) {
	sim := New(Config{SimulatedLatencyMs: 5})
	ctx := context.Background()

	h := sim.Health()
	if h.Online {
		t.Fatal("expected offline before connect")
	}

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	h = sim.Health()
	if !h.Online {
		t.Fatal("expected online after connect")
	}
	if h.LatencyMs != 5 {
		t.Errorf("expected latency 5ms, got %d", h.LatencyMs)
	}

	// After read
	_, _ = sim.ReadAllDI(ctx)
	h = sim.Health()
	if h.LastPollAt.IsZero() {
		t.Fatal("expected LastPollAt to be set after read")
	}
}

// TestScenario_EquipmentRunning verifies the equipment running scenario DI pattern.
func TestScenario_EquipmentRunning(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	scenario := EquipmentRunning()
	ApplyScenario(sim, scenario)

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !states[0].Value {
		t.Fatal("EquipmentRunning: DI[0] should be true (running signal)")
	}
}

// TestScenario_EquipmentStopped verifies the equipment stopped scenario DI pattern.
func TestScenario_EquipmentStopped(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	scenario := EquipmentStopped()
	ApplyScenario(sim, scenario)

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if states[0].Value {
		t.Fatal("EquipmentStopped: DI[0] should be false")
	}
}

// TestScenario_RestartRequested verifies the restart requested scenario DI pattern.
func TestScenario_RestartRequested(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	scenario := RestartRequested()
	ApplyScenario(sim, scenario)

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !states[1].Value {
		t.Fatal("RestartRequested: DI[1] should be true (restart request)")
	}
}

// TestScenario_ModbusOffline verifies the Modbus offline scenario.
// I/O failure must not be treated as normal state.
func TestScenario_ModbusOffline(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	scenario := ModbusOffline()
	ApplyScenario(sim, scenario)

	err := sim.Connect(ctx, ioAdapter.DefaultIOConfig())
	if err == nil {
		t.Fatal("ModbusOffline: connect should fail")
	}
}

// TestScenario_OutputFeedback verifies the output feedback scenario DI pattern.
func TestScenario_OutputFeedback(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	scenario := OutputFeedback()
	ApplyScenario(sim, scenario)

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !states[0].Value {
		t.Fatal("OutputFeedback: DI[0] should be true (equipment running)")
	}
	if !states[2].Value {
		t.Fatal("OutputFeedback: DI[2] should be true (feedback confirmation)")
	}
}

// TestScenario_Timeout verifies the timeout scenario.
func TestScenario_Timeout(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	scenario := Timeout()
	// Don't apply config before connect (connect would succeed with default)
	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	// Now apply timeout scenario (after connect succeeds)
	sim.SetConfig(scenario.Config)

	_, err := sim.ReadDI(ctx, 0)
	if err == nil {
		t.Fatal("Timeout: ReadDI should fail with timeout")
	}

	if _, ok := err.(*domainErrors.TimeoutError); !ok {
		t.Fatalf("expected *TimeoutError, got %T", err)
	}
}

// TestSimulator_SetAllDI verifies bulk DI state setting.
func TestSimulator_SetAllDI(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	values := [ioAdapter.NumDIPoints]bool{true, false, true, false, true, false, true, false}
	sim.SetAllDI(values)

	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for i, state := range states {
		if state.Value != values[i] {
			t.Errorf("DI[%d] = %v, expected %v", i, state.Value, values[i])
		}
	}
}

// TestSimulator_LastUpdate_IsFresh verifies timestamps are recent.
func TestSimulator_LastUpdate_IsFresh(t *testing.T) {
	sim := New(DefaultConfig())
	ctx := context.Background()

	if err := sim.Connect(ctx, ioAdapter.DefaultIOConfig()); err != nil {
		t.Fatal(err)
	}

	before := time.Now().UTC()
	states, err := sim.ReadAllDI(ctx)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC()

	for i, state := range states {
		if state.LastUpdate.Before(before) || state.LastUpdate.After(after) {
			t.Errorf("state[%d].LastUpdate not in expected range", i)
		}
	}
}
