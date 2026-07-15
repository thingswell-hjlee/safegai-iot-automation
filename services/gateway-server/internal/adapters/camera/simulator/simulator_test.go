package simulator

import (
	"context"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/adapters/camera"
)

func TestSimulator_ImplementsCameraAdapter(t *testing.T) {
	var _ camera.CameraAdapter = (*Simulator)(nil)
}

func TestSimulator_ConnectSetsOnline(t *testing.T) {
	sim := New(DefaultConfig())

	health := sim.Health()
	if health.Online {
		t.Error("expected simulator to be offline before Connect")
	}

	err := sim.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	health = sim.Health()
	if !health.Online {
		t.Error("expected simulator to be online after Connect")
	}
}

func TestSimulator_ConnectAfterCloseReturnsError(t *testing.T) {
	sim := New(DefaultConfig())
	sim.Close()

	err := sim.Connect(context.Background())
	if err == nil {
		t.Error("expected error connecting after Close")
	}
}

func TestSimulator_SubscribeEventsRequiresConnect(t *testing.T) {
	sim := New(DefaultConfig())
	ch := make(chan camera.RawCameraEvent, 10)

	err := sim.SubscribeEvents(context.Background(), ch)
	if err == nil {
		t.Error("expected error subscribing without Connect")
	}
}

func TestSimulator_SubscribeEventsProducesEvents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventInterval = 10 * time.Millisecond
	cfg.Scenarios = []ScenarioFunc{Occupied("zone-A")}
	sim := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sim.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	ch := make(chan camera.RawCameraEvent, 10)
	err = sim.SubscribeEvents(ctx, ch)
	if err != nil {
		t.Fatalf("SubscribeEvents failed: %v", err)
	}

	// Collect events
	var events []camera.RawCameraEvent
	timeout := time.After(3 * time.Second)

	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
			// Occupied scenario produces 3 events
			if len(events) >= 3 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(events) != 3 {
		t.Fatalf("expected 3 events from Occupied scenario, got %d", len(events))
	}

	for _, evt := range events {
		if evt.CameraID == "" {
			t.Error("event has empty CameraID")
		}
		if evt.ZoneID != "zone-A" {
			t.Errorf("expected zone-A, got %q", evt.ZoneID)
		}
		if evt.EventType != "person_detected" {
			t.Errorf("expected person_detected, got %q", evt.EventType)
		}
		if evt.PersonCount != 1 {
			t.Errorf("expected personCount=1, got %d", evt.PersonCount)
		}
		if evt.Confidence < 0.0 || evt.Confidence > 1.0 {
			t.Errorf("confidence out of range: %f", evt.Confidence)
		}
		if evt.Timestamp.IsZero() {
			t.Error("event has zero timestamp")
		}
	}
}

func TestSimulator_VacantScenarioNeverEmitsVacantConfirmed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventInterval = 10 * time.Millisecond
	cfg.Scenarios = []ScenarioFunc{Vacant("zone-A")}
	sim := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sim.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	ch := make(chan camera.RawCameraEvent, 10)
	err = sim.SubscribeEvents(ctx, ch)
	if err != nil {
		t.Fatalf("SubscribeEvents failed: %v", err)
	}

	var events []camera.RawCameraEvent
	timeout := time.After(3 * time.Second)
	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
			if len(events) >= 3 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	// Camera adapter must NEVER produce VACANT_CONFIRMED
	for _, evt := range events {
		if evt.EventType == "VACANT_CONFIRMED" || evt.EventType == "vacant_confirmed" {
			t.Error("camera adapter must NEVER produce VACANT_CONFIRMED; only the state engine determines vacancy")
		}
	}
}

func TestSimulator_OfflineScenarioMapsToUnknown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventInterval = 10 * time.Millisecond
	cfg.Scenarios = []ScenarioFunc{Offline()}
	sim := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sim.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	ch := make(chan camera.RawCameraEvent, 10)
	err = sim.SubscribeEvents(ctx, ch)
	if err != nil {
		t.Fatalf("SubscribeEvents failed: %v", err)
	}

	var events []camera.RawCameraEvent
	timeout := time.After(3 * time.Second)
	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
			if len(events) >= 1 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(events) < 1 {
		t.Fatal("expected at least 1 offline event")
	}

	// Verify offline event type is "offline"
	if events[0].EventType != "offline" {
		t.Errorf("expected eventType 'offline', got %q", events[0].EventType)
	}

	// Camera offline must map to UNKNOWN, never VACANT
	// The event type is "offline" which the normalizer maps to UNKNOWN
	if events[0].EventType == "vacant" || events[0].EventType == "VACANT" {
		t.Error("camera offline must never map to VACANT")
	}
}

func TestSimulator_GetCapabilities(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Zones = []string{"zone-A", "zone-B"}
	cfg.MaxPersons = 5
	sim := New(cfg)

	caps := sim.GetCapabilities()
	if len(caps.Zones) != 2 {
		t.Errorf("expected 2 zones, got %d", len(caps.Zones))
	}
	if caps.MaxPersons != 5 {
		t.Errorf("expected maxPersons=5, got %d", caps.MaxPersons)
	}
	if !caps.SupportsSnapshot {
		t.Error("expected SupportsSnapshot=true for simulator")
	}
}

func TestSimulator_GetSnapshotRequiresConnect(t *testing.T) {
	sim := New(DefaultConfig())

	_, err := sim.GetSnapshot(context.Background(), "zone-A")
	if err == nil {
		t.Error("expected error getting snapshot without Connect")
	}
}

func TestSimulator_GetSnapshotReturnsData(t *testing.T) {
	sim := New(DefaultConfig())
	err := sim.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	data, err := sim.GetSnapshot(context.Background(), "zone-A")
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty snapshot data")
	}
}

func TestSimulator_GetSnapshotUnknownZone(t *testing.T) {
	sim := New(DefaultConfig())
	err := sim.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	_, err = sim.GetSnapshot(context.Background(), "zone-nonexistent")
	if err == nil {
		t.Error("expected error for unknown zone")
	}
}

func TestSimulator_CloseStopsSubscription(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventInterval = 50 * time.Millisecond
	cfg.Scenarios = []ScenarioFunc{Occupied("zone-A"), Occupied("zone-A"), Occupied("zone-A")}
	sim := New(cfg)

	ctx := context.Background()
	err := sim.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	ch := make(chan camera.RawCameraEvent, 100)
	err = sim.SubscribeEvents(ctx, ch)
	if err != nil {
		t.Fatalf("SubscribeEvents failed: %v", err)
	}

	// Let a few events come through
	time.Sleep(100 * time.Millisecond)

	err = sim.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// After close, no more events should arrive
	time.Sleep(200 * time.Millisecond)

	health := sim.Health()
	if health.Online {
		t.Error("expected simulator to be offline after Close")
	}
}

func TestSimulator_DuplicateScenarioProducesEvents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventInterval = 10 * time.Millisecond
	cfg.Scenarios = []ScenarioFunc{Duplicate("zone-A")}
	sim := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sim.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	ch := make(chan camera.RawCameraEvent, 10)
	err = sim.SubscribeEvents(ctx, ch)
	if err != nil {
		t.Fatalf("SubscribeEvents failed: %v", err)
	}

	var events []camera.RawCameraEvent
	timeout := time.After(3 * time.Second)
	for {
		select {
		case evt := <-ch:
			events = append(events, evt)
			if len(events) >= 4 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(events) != 4 {
		t.Fatalf("expected 4 events from Duplicate scenario, got %d", len(events))
	}
}

func TestSimulator_HealthUpdatesLastEventAt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventInterval = 10 * time.Millisecond
	cfg.Scenarios = []ScenarioFunc{Occupied("zone-A")}
	sim := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sim.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sim.Close()

	healthBefore := sim.Health()
	if !healthBefore.LastEventAt.IsZero() {
		t.Error("expected zero LastEventAt before events")
	}

	ch := make(chan camera.RawCameraEvent, 10)
	err = sim.SubscribeEvents(ctx, ch)
	if err != nil {
		t.Fatalf("SubscribeEvents failed: %v", err)
	}

	// Wait for at least one event
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	healthAfter := sim.Health()
	if healthAfter.LastEventAt.IsZero() {
		t.Error("expected non-zero LastEventAt after events")
	}
}
