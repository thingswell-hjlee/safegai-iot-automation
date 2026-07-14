// Package observability provides health monitoring and diagnostics for the gateway.
package observability

import (
	"os"
	"runtime"
	"time"
)

// HealthReport contains system health metrics.
type HealthReport struct {
	CPUPercent  float64 `json:"cpuPercent"`
	MemoryUsed  uint64  `json:"memoryUsed"`
	MemoryTotal uint64  `json:"memoryTotal"`
	DiskUsed    uint64  `json:"diskUsed"`
	DiskTotal   uint64  `json:"diskTotal"`
	DiskHealth  string  `json:"diskHealth"`
	Temperature float64 `json:"temperature"`
	Uptime      int64   `json:"uptimeSeconds"`
	NICStatus   string  `json:"nicStatus"`
}

// HealthCollector gathers system health metrics.
type HealthCollector struct {
	startTime time.Time
}

// NewHealthCollector creates a new HealthCollector.
func NewHealthCollector() *HealthCollector {
	return &HealthCollector{
		startTime: time.Now().UTC(),
	}
}

// Collect gathers current system health metrics.
// Note: Some metrics (CPU%, temperature) require platform-specific APIs.
// This implementation provides what is available from Go runtime and os.
func (c *HealthCollector) Collect() HealthReport {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	report := HealthReport{
		CPUPercent:  estimateCPU(),
		MemoryUsed:  m.Alloc,
		MemoryTotal: m.Sys,
		DiskHealth:  "OK",
		Temperature: 0, // Requires hardware access - not available without external deps
		Uptime:      int64(time.Since(c.startTime).Seconds()),
		NICStatus:   detectNICStatus(),
	}

	// Attempt to get disk usage for the data directory
	used, total := diskUsage("/data")
	report.DiskUsed = used
	report.DiskTotal = total
	if total > 0 && float64(used)/float64(total) > 0.9 {
		report.DiskHealth = "WARNING"
	}

	return report
}

// Uptime returns the duration since the collector was created.
func (c *HealthCollector) Uptime() time.Duration {
	return time.Since(c.startTime)
}

// estimateCPU provides a rough CPU estimate using goroutine count.
// A proper implementation would use /proc/stat or platform APIs.
func estimateCPU() float64 {
	numGoroutine := runtime.NumGoroutine()
	numCPU := runtime.NumCPU()
	// Rough heuristic: ratio of goroutines to available CPUs
	estimate := float64(numGoroutine) / float64(numCPU) * 10.0
	if estimate > 100.0 {
		estimate = 100.0
	}
	return estimate
}

// diskUsage attempts to get disk usage for the given path.
// Returns (0, 0) if unavailable. Real implementation needs syscall.Statfs.
func diskUsage(path string) (used, total uint64) {
	info, err := os.Stat(path)
	if err != nil {
		// Path does not exist or is not accessible
		return 0, 0
	}
	_ = info
	// Without syscall.Statfs (platform-specific), we cannot get real disk stats.
	// Return 0 values - real implementation would use unix.Statfs.
	return 0, 0
}

// detectNICStatus checks basic network interface availability.
func detectNICStatus() string {
	// Check if /sys/class/net exists (Linux-specific)
	if _, err := os.Stat("/sys/class/net"); err == nil {
		return "UP"
	}
	return "UNKNOWN"
}
