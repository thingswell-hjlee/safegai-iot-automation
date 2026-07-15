// Package scenarios defines the E2E test scenarios S01-S14 for SafeGAI.
// Each scenario tests a specific safety behavior of the gateway system.
//
// To run: go test -tags=e2e ./tests/e2e/scenarios/ -v
// Requires running gateway and simulators.
package scenarios

import (
	"encoding/json"
	"os"
	"testing"
)

// Scenario defines a test scenario with its metadata.
type Scenario struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Precondition string           `json:"precondition"`
	Steps       []ScenarioStep    `json:"steps"`
	Expected    ExpectedOutcome   `json:"expected"`
	SafetyRule  string            `json:"safetyRule,omitempty"`
}

// ScenarioStep defines a single step in the scenario.
type ScenarioStep struct {
	Order       int               `json:"order"`
	Action      string            `json:"action"`
	Target      string            `json:"target"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	WaitMs      int               `json:"waitMs,omitempty"`
}

// ExpectedOutcome defines what should happen after the scenario.
type ExpectedOutcome struct {
	SafetyDecision string   `json:"safetyDecision"`
	OutputCommands []string `json:"outputCommands,omitempty"`
	ZoneState      string   `json:"zoneState,omitempty"`
	AuditEntries   int      `json:"auditEntries,omitempty"`
}

// AllScenarios returns the complete list of E2E scenarios.
func AllScenarios() []Scenario {
	return []Scenario{
		{
			ID:          "S01",
			Name:        "Person enters hazard zone",
			Description: "Camera detects person in active equipment zone, gateway triggers warning",
			Precondition: "Equipment EQ-PRESS-01 is RUNNING, Zone ZONE-01-01 is VACANT_CONFIRMED",
			Steps: []ScenarioStep{
				{Order: 1, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "PERSON_DETECTED", "zoneId": "ZONE-01-01", "personCount": "1"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
				OutputCommands: []string{"WARNING_LIGHT", "WARNING_SIREN"},
				ZoneState:      "OCCUPIED",
			},
			SafetyRule: "R-01",
		},
		{
			ID:          "S02",
			Name:        "Zone vacancy confirmed",
			Description: "Camera confirms zone vacant after grace period, equipment restart allowed",
			Precondition: "Zone ZONE-01-01 is OCCUPIED, equipment is STOPPED",
			Steps: []ScenarioStep{
				{Order: 1, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "PERSON_LEFT", "zoneId": "ZONE-01-01", "personCount": "0"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 3000},
				{Order: 3, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "ZONE_VACANT", "zoneId": "ZONE-01-01", "personCount": "0"}},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "SAFE",
				ZoneState:      "VACANT_CONFIRMED",
			},
			SafetyRule: "R-02",
		},
		{
			ID:          "S03",
			Name:        "Emergency stop",
			Description: "E-stop signal triggers immediate stop request to all zone equipment",
			Precondition: "Equipment EQ-PRESS-01 is RUNNING",
			Steps: []ScenarioStep{
				{Order: 1, Action: "modbus_di", Target: "DI-01", Parameters: map[string]string{"index": "1", "value": "false"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 100},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "STOP_REQUEST_REQUIRED",
				OutputCommands: []string{"STOP_REQUEST", "WARNING_SIREN"},
				AuditEntries:   2,
			},
			SafetyRule: "R-03",
		},
		{
			ID:          "S04",
			Name:        "Sensor threshold breach",
			Description: "Temperature sensor exceeds critical threshold, alarm raised",
			Precondition: "Normal operating conditions",
			Steps: []ScenarioStep{
				{Order: 1, Action: "sensor_inject", Target: "temperature-01", Parameters: map[string]string{"value": "85.0", "unit": "celsius"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
				OutputCommands: []string{"WARNING_LIGHT", "VOICE_ANNOUNCE"},
			},
			SafetyRule: "R-04",
		},
		{
			ID:          "S05",
			Name:        "Communication loss",
			Description: "Camera goes offline, zone enters STALE state, safe-side default",
			Precondition: "Camera CAM-01 is online, zone is monitored",
			Steps: []ScenarioStep{
				{Order: 1, Action: "camera_offline", Target: "CAM-01", Parameters: nil},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 10000},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "SAFETY_CONFIRMATION_UNAVAILABLE",
				ZoneState:      "STALE",
			},
			SafetyRule: "R-05",
		},
		{
			ID:          "S06",
			Name:        "Equipment fault",
			Description: "Equipment reports FAULT state, maintenance monitoring activated",
			Precondition: "Equipment EQ-PRESS-01 is RUNNING",
			Steps: []ScenarioStep{
				{Order: 1, Action: "equipment_state", Target: "EQ-PRESS-01", Parameters: map[string]string{"state": "FAULT"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "MAINTENANCE_MONITORING",
				OutputCommands: []string{"WARNING_LIGHT"},
			},
		},
		{
			ID:          "S07",
			Name:        "Multi-zone occupancy",
			Description: "Multiple zones occupied simultaneously, independent safety evaluation",
			Precondition: "All zones vacant, equipment running",
			Steps: []ScenarioStep{
				{Order: 1, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "PERSON_DETECTED", "zoneId": "ZONE-01-01", "personCount": "1"}},
				{Order: 2, Action: "camera_event", Target: "CAM-02", Parameters: map[string]string{"eventType": "PERSON_DETECTED", "zoneId": "ZONE-02-01", "personCount": "2"}},
				{Order: 3, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
				AuditEntries:   2,
			},
		},
		{
			ID:          "S08",
			Name:        "Restart interlock",
			Description: "Restart request blocked until vacancy confirmed by human operator",
			Precondition: "Zone ZONE-01-01 is OCCUPIED, equipment is STOPPED",
			Steps: []ScenarioStep{
				{Order: 1, Action: "restart_request", Target: "EQ-PRESS-01", Parameters: map[string]string{"operator": "test-user"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 200},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "RESTART_INTERLOCK",
				AuditEntries:   1,
			},
		},
		{
			ID:          "S09",
			Name:        "Network partition",
			Description: "Cloud connection lost, gateway continues local safety operations",
			Precondition: "Gateway connected to cloud, zone monitoring active",
			Steps: []ScenarioStep{
				{Order: 1, Action: "cloud_disconnect", Target: "", Parameters: nil},
				{Order: 2, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "PERSON_DETECTED", "zoneId": "ZONE-01-01", "personCount": "1"}},
				{Order: 3, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
				OutputCommands: []string{"WARNING_LIGHT"},
			},
		},
		{
			ID:          "S10",
			Name:        "Modbus DI alarm",
			Description: "Digital input triggers safety alarm via Modbus",
			Precondition: "Modbus DI-02 is normal (true)",
			Steps: []ScenarioStep{
				{Order: 1, Action: "modbus_di", Target: "DI-02", Parameters: map[string]string{"index": "2", "value": "true"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
			},
		},
		{
			ID:          "S11",
			Name:        "Voice announcement",
			Description: "Safety warning triggers voice announcement output",
			Precondition: "Safety warning condition met",
			Steps: []ScenarioStep{
				{Order: 1, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "PERSON_DETECTED", "zoneId": "ZONE-01-01", "personCount": "3"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
				OutputCommands: []string{"VOICE_ANNOUNCE"},
			},
		},
		{
			ID:          "S12",
			Name:        "Audit trail completeness",
			Description: "All safety decisions logged with full traceability",
			Precondition: "Gateway running with clean audit log",
			Steps: []ScenarioStep{
				{Order: 1, Action: "camera_event", Target: "CAM-01", Parameters: map[string]string{"eventType": "PERSON_DETECTED", "zoneId": "ZONE-01-01", "personCount": "1"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 500},
				{Order: 3, Action: "verify_audit", Target: "", Parameters: map[string]string{"minEntries": "1"}},
			},
			Expected: ExpectedOutcome{
				AuditEntries: 1,
			},
		},
		{
			ID:          "S13",
			Name:        "Concurrent events",
			Description: "Multiple simultaneous events processed without race conditions",
			Precondition: "Multiple cameras active, equipment running",
			Steps: []ScenarioStep{
				{Order: 1, Action: "concurrent_events", Target: "ALL", Parameters: map[string]string{"count": "10", "interval_ms": "50"}},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 2000},
				{Order: 3, Action: "verify_no_panic", Target: "", Parameters: nil},
			},
			Expected: ExpectedOutcome{
				SafetyDecision: "WARNING",
			},
		},
		{
			ID:          "S14",
			Name:        "Graceful shutdown",
			Description: "Gateway shutdown preserves state and completes pending operations",
			Precondition: "Gateway running with active events",
			Steps: []ScenarioStep{
				{Order: 1, Action: "send_sigterm", Target: "gateway", Parameters: nil},
				{Order: 2, Action: "wait", Target: "", Parameters: nil, WaitMs: 5000},
				{Order: 3, Action: "verify_shutdown", Target: "gateway", Parameters: nil},
			},
			Expected: ExpectedOutcome{
				AuditEntries: 1,
			},
		},
	}
}

func TestScenariosDefinition(t *testing.T) {
	scenarios := AllScenarios()

	if len(scenarios) != 14 {
		t.Errorf("expected 14 scenarios, got %d", len(scenarios))
	}

	seen := make(map[string]bool)
	for _, s := range scenarios {
		if seen[s.ID] {
			t.Errorf("duplicate scenario ID: %s", s.ID)
		}
		seen[s.ID] = true

		if s.Name == "" {
			t.Errorf("scenario %s has empty name", s.ID)
		}
		if s.Description == "" {
			t.Errorf("scenario %s has empty description", s.ID)
		}
		if len(s.Steps) == 0 {
			t.Errorf("scenario %s has no steps", s.ID)
		}
	}
}

func TestScenariosJSON(t *testing.T) {
	scenarios := AllScenarios()
	data, err := json.MarshalIndent(scenarios, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal scenarios: %v", err)
	}

	// Write scenarios JSON for runner consumption
	outPath := os.Getenv("SCENARIO_OUTPUT_PATH")
	if outPath != "" {
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			t.Logf("could not write scenarios JSON: %v", err)
		}
	}

	// Verify round-trip
	var parsed []Scenario
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal scenarios: %v", err)
	}

	if len(parsed) != len(scenarios) {
		t.Errorf("round-trip mismatch: %d vs %d", len(parsed), len(scenarios))
	}
}
