package observability

import (
	"testing"
	"time"
)

func TestHealthCollector_Collect(t *testing.T) {
	collector := NewHealthCollector()

	// Wait a bit for uptime to be non-zero
	time.Sleep(10 * time.Millisecond)

	report := collector.Collect()

	if report.Uptime < 0 {
		t.Errorf("expected non-negative uptime, got %d", report.Uptime)
	}

	if report.CPUPercent < 0 || report.CPUPercent > 100 {
		t.Errorf("expected CPU percent 0-100, got %f", report.CPUPercent)
	}

	// MemoryTotal should be non-zero (Go runtime always uses some memory)
	if report.MemoryTotal == 0 {
		t.Error("expected non-zero MemoryTotal")
	}

	// MemoryUsed should be non-zero
	if report.MemoryUsed == 0 {
		t.Error("expected non-zero MemoryUsed")
	}

	// DiskHealth should be a valid value
	if report.DiskHealth != "OK" && report.DiskHealth != "WARNING" {
		t.Errorf("unexpected DiskHealth: %s", report.DiskHealth)
	}

	// NICStatus should be a valid value
	if report.NICStatus != "UP" && report.NICStatus != "UNKNOWN" {
		t.Errorf("unexpected NICStatus: %s", report.NICStatus)
	}
}

func TestHealthCollector_Uptime(t *testing.T) {
	collector := NewHealthCollector()

	time.Sleep(50 * time.Millisecond)

	uptime := collector.Uptime()
	if uptime < 50*time.Millisecond {
		t.Errorf("expected uptime >= 50ms, got %v", uptime)
	}
}

func TestHealthCollector_MultipleCollects(t *testing.T) {
	collector := NewHealthCollector()

	r1 := collector.Collect()
	time.Sleep(10 * time.Millisecond)
	r2 := collector.Collect()

	if r2.Uptime < r1.Uptime {
		t.Error("expected uptime to increase between collects")
	}
}
