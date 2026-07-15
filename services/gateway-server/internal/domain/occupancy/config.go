package occupancy

import "time"

// Config holds the configuration parameters for the zone occupancy state machine.
//
// R3 SAFETY CRITICAL: These parameters directly affect when vacancy is confirmed.
// Conservative defaults are chosen to prevent false vacancy detection.
//
// Default values per PRODUCT_MVP_SPEC:
//   - VacancyConfirmDuration: 3s (time a zone must remain empty)
//   - VacancyConfirmSamples:  3 (consecutive no-person samples required)
//   - StaleTimeout:           10s (time without data before zone goes STALE)
//   - CameraOfflineTimeout:   30s (time without camera heartbeat before UNKNOWN)
//   - DedupWindow:            2s (duplicate event suppression window)
type Config struct {
	// VacancyConfirmDuration is the minimum time a zone must remain without
	// detected persons before vacancy can be confirmed.
	// R3 SAFETY CRITICAL: Shorter values increase false vacancy risk.
	VacancyConfirmDuration time.Duration

	// VacancyConfirmSamples is the minimum number of consecutive no-person
	// detection events required before vacancy can be confirmed.
	// R3 SAFETY CRITICAL: Lower values increase false vacancy risk.
	VacancyConfirmSamples int

	// StaleTimeout is the maximum time allowed between events before
	// the zone transitions to STALE state.
	// R3 SAFETY CRITICAL: STALE blocks restart and does NOT satisfy vacancy.
	StaleTimeout time.Duration

	// CameraOfflineTimeout is the maximum time allowed without a camera
	// heartbeat before the zone transitions to UNKNOWN.
	// R3 SAFETY CRITICAL: Camera offline MUST produce UNKNOWN, NEVER VACANT.
	CameraOfflineTimeout time.Duration

	// DedupWindow is the time window within which duplicate events from
	// the same source are suppressed.
	DedupWindow time.Duration
}

// DefaultConfig returns the default configuration with safety-conservative values.
//
// R3 SAFETY CRITICAL: These are the production defaults from PRODUCT_MVP_SPEC.
// Do not reduce VacancyConfirmDuration or VacancyConfirmSamples without
// safety review and T1/T2 approval.
func DefaultConfig() Config {
	return Config{
		VacancyConfirmDuration: 3 * time.Second,
		VacancyConfirmSamples:  3,
		StaleTimeout:           10 * time.Second,
		CameraOfflineTimeout:   30 * time.Second,
		DedupWindow:            2 * time.Second,
	}
}
