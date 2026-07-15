// Package main implements the sensor simulator for SafeGAI.
// It simulates 6 sensor types (temperature, humidity, co2, gas, vibration, current)
// generating periodic readings for integration testing.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	defaultAddr     = ":9002"
	defaultInterval = 3 * time.Second
	version         = "0.1.0"
)

// SensorType defines a sensor category with its normal ranges.
type SensorType struct {
	Type    string
	Unit    string
	Min     float64
	Max     float64
	Sensors int
}

var sensorTypes = []SensorType{
	{Type: "temperature", Unit: "celsius", Min: 15.0, Max: 45.0, Sensors: 4},
	{Type: "humidity", Unit: "percent", Min: 20.0, Max: 90.0, Sensors: 4},
	{Type: "co2", Unit: "ppm", Min: 300.0, Max: 2000.0, Sensors: 2},
	{Type: "gas", Unit: "ppm", Min: 0.0, Max: 50.0, Sensors: 2},
	{Type: "vibration", Unit: "mm/s", Min: 0.0, Max: 15.0, Sensors: 3},
	{Type: "current", Unit: "ampere", Min: 0.0, Max: 100.0, Sensors: 3},
}

// SensorReading mirrors ports.SensorReading.
type SensorReading struct {
	SensorID   string    `json:"sensorId"`
	SensorType string    `json:"sensorType"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Quality    string    `json:"quality"`
	Timestamp  time.Time `json:"timestamp"`
}

// SimState tracks simulator state.
type SimState struct {
	mu        sync.RWMutex
	running   bool
	readings  []SensorReading
	totalSent int64
	startTime time.Time
}

var state = &SimState{}

func main() {
	addr := envOrDefault("SENSOR_SIM_ADDR", defaultAddr)
	interval := parseDuration(envOrDefault("SENSOR_SIM_INTERVAL", "3s"), defaultInterval)

	logJSON("info", "SafeGAI sensor simulator starting", map[string]string{
		"addr":     addr,
		"types":    "6",
		"interval": interval.String(),
	})

	state.startTime = time.Now()
	state.running = true

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go generateReadings(ctx, interval)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/readings", handleReadings)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logJSON("error", fmt.Sprintf("HTTP server error: %v", err), nil)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logJSON("info", "Shutting down sensor simulator", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	state.mu.Lock()
	state.running = false
	state.mu.Unlock()

	logJSON("info", fmt.Sprintf("Sensor simulator stopped. Total readings: %d", state.totalSent), nil)
}

func generateReadings(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track base values for smooth drift
	baseValues := make(map[string]float64)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, st := range sensorTypes {
				for i := 1; i <= st.Sensors; i++ {
					sensorID := fmt.Sprintf("%s-%02d", st.Type, i)

					// Initialize or drift base value
					base, exists := baseValues[sensorID]
					if !exists {
						base = st.Min + (st.Max-st.Min)*0.5
					}
					// Random walk with mean reversion
					drift := (rand.Float64() - 0.5) * (st.Max - st.Min) * 0.05
					meanRevert := (st.Min + (st.Max-st.Min)*0.5 - base) * 0.01
					base = base + drift + meanRevert
					base = math.Max(st.Min, math.Min(st.Max, base))
					baseValues[sensorID] = base

					quality := "GOOD"
					if rand.Float64() < 0.02 {
						quality = "UNCERTAIN"
					}

					reading := SensorReading{
						SensorID:   sensorID,
						SensorType: st.Type,
						Value:      math.Round(base*100) / 100,
						Unit:       st.Unit,
						Quality:    quality,
						Timestamp:  time.Now().UTC(),
					}

					state.mu.Lock()
					state.readings = append(state.readings, reading)
					if len(state.readings) > 200 {
						state.readings = state.readings[len(state.readings)-200:]
					}
					state.totalSent++
					state.mu.Unlock()
				}
			}
		}
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	running := state.running
	state.mu.RUnlock()

	status := "healthy"
	if !running {
		status = "stopping"
	}

	totalSensors := 0
	for _, st := range sensorTypes {
		totalSensors += st.Sensors
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"version": version,
		"uptime":  time.Since(state.startTime).Truncate(time.Second).String(),
		"sensors": totalSensors,
		"types":   len(sensorTypes),
	})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	totalSent := state.totalSent
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalReadingsSent": totalSent,
		"uptime":            time.Since(state.startTime).Truncate(time.Second).String(),
	})
}

func handleReadings(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	readings := make([]SensorReading, len(state.readings))
	copy(readings, state.readings)
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readings)
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func logJSON(level, message string, fields map[string]string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"component": "sensor-sim",
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
