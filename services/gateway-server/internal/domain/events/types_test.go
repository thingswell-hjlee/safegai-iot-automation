package events

import "testing"

func TestOccupancyState_AllStatesDefined(t *testing.T) {
	expected := []OccupancyState{
		OccupancyOccupied,
		OccupancyVacantPending,
		OccupancyVacantConfirmed,
		OccupancyUnknown,
		OccupancyStale,
	}

	if len(ValidOccupancyStates) != len(expected) {
		t.Fatalf("ValidOccupancyStates has %d entries, want %d", len(ValidOccupancyStates), len(expected))
	}

	for _, s := range expected {
		if !s.IsValid() {
			t.Errorf("expected %q to be a valid OccupancyState", s)
		}
	}
}

func TestOccupancyState_StringValues(t *testing.T) {
	tests := []struct {
		state OccupancyState
		value string
	}{
		{OccupancyOccupied, "OCCUPIED"},
		{OccupancyVacantPending, "VACANT_PENDING"},
		{OccupancyVacantConfirmed, "VACANT_CONFIRMED"},
		{OccupancyUnknown, "UNKNOWN"},
		{OccupancyStale, "STALE"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if string(tt.state) != tt.value {
				t.Errorf("OccupancyState %q has string value %q, want %q", tt.state, string(tt.state), tt.value)
			}
		})
	}
}

func TestOccupancyState_IsVacant(t *testing.T) {
	tests := []struct {
		state    OccupancyState
		isVacant bool
	}{
		{OccupancyOccupied, false},
		{OccupancyVacantPending, false},
		{OccupancyVacantConfirmed, true},
		{OccupancyUnknown, false},
		{OccupancyStale, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsVacant(); got != tt.isVacant {
				t.Errorf("OccupancyState(%q).IsVacant() = %v, want %v", tt.state, got, tt.isVacant)
			}
		})
	}
}

func TestOccupancyState_UnknownAndStaleAreNotVacant(t *testing.T) {
	// Safety rule: UNKNOWN and STALE must never be treated as vacancy.
	if OccupancyUnknown.IsVacant() {
		t.Error("UNKNOWN must not be treated as vacant")
	}
	if OccupancyStale.IsVacant() {
		t.Error("STALE must not be treated as vacant")
	}
}

func TestOccupancyState_InvalidValue(t *testing.T) {
	invalid := OccupancyState("NONEXISTENT")
	if invalid.IsValid() {
		t.Error("expected NONEXISTENT to be invalid OccupancyState")
	}
}

func TestEquipmentState_AllStatesDefined(t *testing.T) {
	expected := []EquipmentState{
		EquipmentRunning,
		EquipmentStopped,
		EquipmentStarting,
		EquipmentStopping,
		EquipmentFault,
		EquipmentOffline,
		EquipmentUnknown,
	}

	if len(ValidEquipmentStates) != len(expected) {
		t.Fatalf("ValidEquipmentStates has %d entries, want %d", len(ValidEquipmentStates), len(expected))
	}

	for _, s := range expected {
		if !s.IsValid() {
			t.Errorf("expected %q to be a valid EquipmentState", s)
		}
	}
}

func TestEquipmentState_StringValues(t *testing.T) {
	tests := []struct {
		state EquipmentState
		value string
	}{
		{EquipmentRunning, "RUNNING"},
		{EquipmentStopped, "STOPPED"},
		{EquipmentStarting, "STARTING"},
		{EquipmentStopping, "STOPPING"},
		{EquipmentFault, "FAULT"},
		{EquipmentOffline, "OFFLINE"},
		{EquipmentUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if string(tt.state) != tt.value {
				t.Errorf("EquipmentState %q has string value %q, want %q", tt.state, string(tt.state), tt.value)
			}
		})
	}
}

func TestEquipmentState_RestartRequestedNotAState(t *testing.T) {
	invalid := EquipmentState("RESTART_REQUESTED")
	if invalid.IsValid() {
		t.Error("RESTART_REQUESTED must not be a valid EquipmentState")
	}
}

func TestEquipmentState_InvalidValue(t *testing.T) {
	invalid := EquipmentState("MAINTENANCE")
	if invalid.IsValid() {
		t.Error("expected MAINTENANCE to be invalid EquipmentState")
	}
}

func TestSafetyDecision_AllDecisionsDefined(t *testing.T) {
	expected := []SafetyDecision{
		SafetyDecisionSafe,
		SafetyDecisionWarning,
		SafetyDecisionStopRequestRequired,
		SafetyDecisionRestartInterlock,
		SafetyDecisionSafetyConfirmationUnavailable,
		SafetyDecisionMaintenanceMonitoring,
	}

	if len(ValidSafetyDecisions) != len(expected) {
		t.Fatalf("ValidSafetyDecisions has %d entries, want %d", len(ValidSafetyDecisions), len(expected))
	}

	for _, d := range expected {
		if !d.IsValid() {
			t.Errorf("expected %q to be a valid SafetyDecision", d)
		}
	}
}

func TestSafetyDecision_StringValues(t *testing.T) {
	tests := []struct {
		decision SafetyDecision
		value    string
	}{
		{SafetyDecisionSafe, "SAFE"},
		{SafetyDecisionWarning, "WARNING"},
		{SafetyDecisionStopRequestRequired, "STOP_REQUEST_REQUIRED"},
		{SafetyDecisionRestartInterlock, "RESTART_INTERLOCK"},
		{SafetyDecisionSafetyConfirmationUnavailable, "SAFETY_CONFIRMATION_UNAVAILABLE"},
		{SafetyDecisionMaintenanceMonitoring, "MAINTENANCE_MONITORING"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if string(tt.decision) != tt.value {
				t.Errorf("SafetyDecision %q has string value %q, want %q", tt.decision, string(tt.decision), tt.value)
			}
		})
	}
}

func TestSafetyDecision_InvalidValue(t *testing.T) {
	invalid := SafetyDecision("ABORT")
	if invalid.IsValid() {
		t.Error("expected ABORT to be invalid SafetyDecision")
	}
}
