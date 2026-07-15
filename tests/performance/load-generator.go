// Package main implements a load generator for SafeGAI gateway performance testing.
// It generates concurrent camera events, sensor readings, and output commands
// at configurable rates to measure gateway throughput and latency.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	target      = flag.String("target", "http://localhost:8080", "Gateway target URL")
	duration    = flag.Duration("duration", 5*time.Minute, "Test duration")
	concurrency = flag.Int("concurrency", 10, "Number of concurrent workers")
	ratePerSec  = flag.Int("rate", 100, "Events per second target")
	outputFile  = flag.String("output", "/tmp/safegai-load-report.json", "Report output path")
)

// Metrics tracks performance metrics.
type Metrics struct {
	TotalRequests  int64
	SuccessCount   int64
	ErrorCount     int64
	TotalLatencyUs int64
	MaxLatencyUs   int64
	MinLatencyUs   int64
}

func main() {
	flag.Parse()

	fmt.Printf("SafeGAI Load Generator\n")
	fmt.Printf("  Target: %s\n", *target)
	fmt.Printf("  Duration: %s\n", *duration)
	fmt.Printf("  Concurrency: %d\n", *concurrency)
	fmt.Printf("  Target Rate: %d events/sec\n", *ratePerSec)
	fmt.Printf("  Output: %s\n", *outputFile)
	fmt.Println()

	metrics := &Metrics{
		MinLatencyUs: int64(^uint64(0) >> 1), // MaxInt64
	}

	// Pre-check: verify gateway is reachable
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(*target + "/health/live")
	if err != nil {
		fmt.Printf("ERROR: Gateway unreachable at %s: %v\n", *target, err)
		os.Exit(1)
	}
	resp.Body.Close()
	fmt.Println("Gateway reachable. Starting load test...")

	// Start workers
	var wg sync.WaitGroup
	stop := make(chan struct{})
	interval := time.Second / time.Duration(*ratePerSec / *concurrency)

	startTime := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(workerID, client, metrics, stop, interval)
		}(i)
	}

	// Run for duration
	time.Sleep(*duration)
	close(stop)
	wg.Wait()

	elapsed := time.Since(startTime)

	// Calculate results
	total := atomic.LoadInt64(&metrics.TotalRequests)
	success := atomic.LoadInt64(&metrics.SuccessCount)
	errors := atomic.LoadInt64(&metrics.ErrorCount)
	totalLatency := atomic.LoadInt64(&metrics.TotalLatencyUs)
	maxLatency := atomic.LoadInt64(&metrics.MaxLatencyUs)
	minLatency := atomic.LoadInt64(&metrics.MinLatencyUs)

	avgLatencyUs := int64(0)
	if total > 0 {
		avgLatencyUs = totalLatency / total
	}

	throughput := float64(total) / elapsed.Seconds()

	// Print results
	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Duration: %s\n", elapsed.Truncate(time.Second))
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Success: %d (%.1f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Errors: %d (%.1f%%)\n", errors, float64(errors)/float64(total)*100)
	fmt.Printf("Throughput: %.1f req/sec\n", throughput)
	fmt.Printf("Avg Latency: %d us (%.2f ms)\n", avgLatencyUs, float64(avgLatencyUs)/1000)
	fmt.Printf("Max Latency: %d us (%.2f ms)\n", maxLatency, float64(maxLatency)/1000)
	fmt.Printf("Min Latency: %d us (%.2f ms)\n", minLatency, float64(minLatency)/1000)

	// Write JSON report
	report := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"target":         *target,
		"duration":       elapsed.String(),
		"concurrency":    *concurrency,
		"targetRate":     *ratePerSec,
		"totalRequests":  total,
		"successCount":   success,
		"errorCount":     errors,
		"throughput":     throughput,
		"avgLatencyUs":   avgLatencyUs,
		"maxLatencyUs":   maxLatency,
		"minLatencyUs":   minLatency,
		"successRate":    float64(success) / float64(total) * 100,
	}

	data, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		fmt.Printf("Failed to write report: %v\n", err)
	} else {
		fmt.Printf("\nReport written to: %s\n", *outputFile)
	}
}

func worker(id int, client *http.Client, metrics *Metrics, stop <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	endpoints := []string{
		"/health/live",
		"/health/ready",
	}

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			endpoint := endpoints[rand.Intn(len(endpoints))]
			start := time.Now()

			resp, err := client.Get(*target + endpoint)
			latencyUs := time.Since(start).Microseconds()

			atomic.AddInt64(&metrics.TotalRequests, 1)
			atomic.AddInt64(&metrics.TotalLatencyUs, latencyUs)

			// Update max latency
			for {
				old := atomic.LoadInt64(&metrics.MaxLatencyUs)
				if latencyUs <= old || atomic.CompareAndSwapInt64(&metrics.MaxLatencyUs, old, latencyUs) {
					break
				}
			}

			// Update min latency
			for {
				old := atomic.LoadInt64(&metrics.MinLatencyUs)
				if latencyUs >= old || atomic.CompareAndSwapInt64(&metrics.MinLatencyUs, old, latencyUs) {
					break
				}
			}

			if err != nil {
				atomic.AddInt64(&metrics.ErrorCount, 1)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&metrics.SuccessCount, 1)
			} else {
				atomic.AddInt64(&metrics.ErrorCount, 1)
			}
		}
	}
}

// generateCameraEvent creates a simulated camera event payload.
func generateCameraEvent() string {
	cameraID := fmt.Sprintf("CAM-%02d", rand.Intn(4)+1)
	zoneID := fmt.Sprintf("ZONE-%02d-%02d", rand.Intn(4)+1, rand.Intn(2)+1)
	types := []string{"PERSON_DETECTED", "PERSON_LEFT", "ZONE_OCCUPIED", "ZONE_VACANT"}
	eventType := types[rand.Intn(len(types))]

	return fmt.Sprintf(`{"cameraId":"%s","zoneId":"%s","eventType":"%s","timestamp":"%s","personCount":%d}`,
		cameraID, zoneID, eventType, time.Now().UTC().Format(time.RFC3339), rand.Intn(3))
}

// postEvent sends a POST request with a JSON body (unused when targeting health endpoints).
func postEvent(client *http.Client, url string, body string) (*http.Response, error) {
	return client.Post(url, "application/json", strings.NewReader(body))
}
