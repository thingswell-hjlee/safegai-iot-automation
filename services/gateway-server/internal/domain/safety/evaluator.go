package safety

// This file implements the Activity Engine evaluator that combines all safety rules.
//
// SAFETY CLASSIFICATION: R3 (Risk Level 3 - Safety Critical)
//
// The evaluator applies all rules in strict priority order:
//   Priority: R-03 > R-01 > R-02 > R-04 > SAFE
//
// Key safety invariants enforced by this evaluator:
//   - UNKNOWN and STALE ALWAYS block restart (R-03 has highest priority)
//   - No automatic restart from AI vacancy alone
//   - Rules are deterministic: same input always produces same output
//   - Duplicate suppression prevents duplicate output commands (R-05)
//   - Output is PLC/Safety Relay stop requests only

import (
	"sort"
	"sync"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// EvaluationContext contains all state needed for a safety evaluation cycle.
// [R3] This is the complete input to the deterministic evaluation function.
type EvaluationContext struct {
	// ZoneStates maps zone IDs to their current occupancy state.
	ZoneStates map[string]events.OccupancyState

	// EquipmentStates maps equipment IDs to their current running state.
	EquipmentStates map[string]events.EquipmentState

	// RestartRequested maps equipment IDs to whether restart has been requested.
	RestartRequested map[string]bool

	// ActiveWorkWindows is the set of zone IDs with active approved maintenance windows.
	ActiveWorkWindows map[string]bool

	// ZoneEquipmentMap maps zone IDs to the set of equipment IDs in that zone.
	ZoneEquipmentMap map[string][]string

	// CorrelationID links the evaluation to the triggering event chain.
	CorrelationID string

	// EvaluationTime is the time at which the evaluation is performed.
	// If zero, time.Now() is used for timestamps in results.
	EvaluationTime time.Time
}

// Evaluator is the Activity Engine that applies all safety rules.
// [R3] SAFETY CRITICAL COMPONENT - Thread-safe via internal mutex.
type Evaluator struct {
	mu    sync.Mutex
	dedup *DedupFilter
}

// NewEvaluator creates a new Evaluator with the given dedup suppression window.
// [R3] The evaluator is safe for concurrent use.
func NewEvaluator(suppressionWindow time.Duration) *Evaluator {
	return &Evaluator{
		dedup: NewDedupFilter(suppressionWindow),
	}
}

// Evaluate applies all safety rules to the given context and returns decisions.
//
// [R3] SAFETY CRITICAL FUNCTION
//
// Rule evaluation priority (highest to lowest):
//  1. R-03: UNKNOWN/STALE => SAFETY_CONFIRMATION_UNAVAILABLE
//  2. R-01: OCCUPIED + RUNNING => STOP_REQUEST_REQUIRED
//  3. R-02: restart + zone != VACANT_CONFIRMED => RESTART_INTERLOCK
//  4. R-04: work window + STOPPED => MAINTENANCE_MONITORING
//  5. SAFE (default when no rules trigger)
//
// [R3] UNKNOWN and STALE ALWAYS block restart.
// [R3] No automatic restart from AI vacancy alone.
// [R3] Same input always produces same output (deterministic).
// [R3] Duplicate decisions within the suppression window are filtered (R-05).
func (e *Evaluator) Evaluate(ctx EvaluationContext) []DecisionResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	var results []DecisionResult

	// Process each zone
	for zoneID, zoneState := range ctx.ZoneStates {
		equipmentIDs := ctx.ZoneEquipmentMap[zoneID]
		if len(equipmentIDs) == 0 {
			// Zone with no equipment: still evaluate R-03 for awareness
			equipmentIDs = []string{""}
		}

		for _, equipID := range equipmentIDs {
			zoneResults := e.evaluateZoneEquipment(ctx, zoneID, zoneState, equipID)
			results = append(results, zoneResults...)
		}
	}

	// Sort by severity (highest first) for deterministic output order
	sort.Slice(results, func(i, j int) bool {
		sevI := DecisionSeverity(results[i].Decision)
		sevJ := DecisionSeverity(results[j].Decision)
		if sevI != sevJ {
			return sevI > sevJ
		}
		// Tie-break by zone ID then equipment ID for full determinism
		if results[i].ZoneID != results[j].ZoneID {
			return results[i].ZoneID < results[j].ZoneID
		}
		return results[i].EquipmentID < results[j].EquipmentID
	})

	// Apply R-05 dedup filter
	filtered := make([]DecisionResult, 0, len(results))
	for i := range results {
		if !e.dedup.IsDuplicate(&results[i]) {
			filtered = append(filtered, results[i])
		}
	}

	return filtered
}

// evaluateZoneEquipment applies all rules to a single zone+equipment pair.
// [R3] SAFETY CRITICAL - applies rules in strict priority order.
func (e *Evaluator) evaluateZoneEquipment(
	ctx EvaluationContext,
	zoneID string,
	zoneState events.OccupancyState,
	equipID string,
) []DecisionResult {
	var results []DecisionResult
	correlationID := ctx.CorrelationID

	equipState := events.EquipmentUnknown
	if equipID != "" {
		if s, ok := ctx.EquipmentStates[equipID]; ok {
			equipState = s
		}
	}

	restartRequested := false
	if equipID != "" {
		restartRequested = ctx.RestartRequested[equipID]
	}

	workWindowActive := ctx.ActiveWorkWindows[zoneID]

	// Priority 1 (highest): R-03 - UNKNOWN/STALE blocks everything
	// [R3] This check has highest priority. UNKNOWN and STALE always
	// produce SAFETY_CONFIRMATION_UNAVAILABLE regardless of other state.
	r03 := EvaluateSafetyUnavailable(zoneState, zoneID, equipID, correlationID)
	if r03 != nil {
		results = append(results, *r03)
		// R-03 supersedes R-01 and R-02 for this zone.
		// UNKNOWN/STALE means we cannot confirm safety at all.
		return results
	}

	// Priority 2: R-01 - OCCUPIED + RUNNING = STOP_REQUEST_REQUIRED
	r01 := EvaluateOccupiedRunning(zoneState, equipState, zoneID, equipID, correlationID)
	if r01 != nil {
		results = append(results, *r01)
	}

	// Priority 3: R-02 - Restart interlock
	// [R3] Even if R-01 triggered, we also check R-02 because
	// a restart request during occupied state must be explicitly blocked.
	r02 := EvaluateRestartInterlock(zoneState, restartRequested, zoneID, equipID, correlationID)
	if r02 != nil {
		results = append(results, *r02)
	}

	// Priority 4: R-04 - Maintenance window
	r04 := EvaluateMaintenanceWindow(workWindowActive, equipState, zoneID, equipID, correlationID)
	if r04 != nil {
		results = append(results, *r04)
	}

	// Default: SAFE (only if no other rules triggered)
	if len(results) == 0 {
		results = append(results, DecisionResult{
			Decision:      DecisionSafe,
			Rule:          "DEFAULT",
			ZoneID:        zoneID,
			EquipmentID:   equipID,
			Reason:        "no safety rules triggered",
			Timestamp:     time.Now(),
			CorrelationID: correlationID,
		})
	}

	return results
}

// EvaluateSimple evaluates a single zone+equipment pair without dedup.
// This is a convenience function for simple single-pair evaluations.
//
// [R3] SAFETY CRITICAL FUNCTION
// [R3] Same input always produces same output (deterministic).
func (e *Evaluator) EvaluateSimple(
	zoneState events.OccupancyState,
	equipState events.EquipmentState,
	restartRequested bool,
	workWindowActive bool,
	zoneID, equipmentID, correlationID string,
) []DecisionResult {
	ctx := EvaluationContext{
		ZoneStates:        map[string]events.OccupancyState{zoneID: zoneState},
		EquipmentStates:   map[string]events.EquipmentState{equipmentID: equipState},
		RestartRequested:  map[string]bool{equipmentID: restartRequested},
		ActiveWorkWindows: map[string]bool{zoneID: workWindowActive},
		ZoneEquipmentMap:  map[string][]string{zoneID: {equipmentID}},
		CorrelationID:     correlationID,
	}
	return e.Evaluate(ctx)
}
