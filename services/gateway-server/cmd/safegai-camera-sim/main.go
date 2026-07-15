// Package main implements the camera simulator for SafeGAI.
// It simulates 4 cameras with 8 zones, generating occupancy events
// at configurable rates for integration testing.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	defaultAddr      = ":9001"
	defaultGateway   = "http://localhost:8080"
	defaultInterval  = 2 * time.Second
	numCameras       = 4
	zonesPerCamera   = 2
	totalZones       = numCameras * zonesPerCamera
	version          = "0.1.0"
)

// CameraEvent mirrors ports.CameraEvent for simulator output.
type CameraEvent struct {
	CameraID    string    `json:"cameraId"`
	ZoneID      string    `json:"zoneId"`
	EventType   string    `json:"eventType"`
	Timestamp   time.Time `json:"timestamp"`
	PersonCount int       `json:"personCount"`
	Confidence  float64   `json:"confidence"`
	FrameID     string    `json:"frameId"`
	SequenceNo  int64     `json:"sequenceNo"`
}

// SimState tracks the simulator's current state.
type SimState struct {
	mu         sync.RWMutex
	running    bool
	events     []CameraEvent
	seqNo      int64
	totalSent  int64
	startTime  time.Time
}

var (
	state = &SimState{}
)

func main() {
	addr := envOrDefault("CAMERA_SIM_ADDR", defaultAddr)
	gatewayURL := envOrDefault("GATEWAY_URL", defaultGateway)
	interval := parseDuration(envOrDefault("CAMERA_SIM_INTERVAL", "2s"), defaultInterval)

	logJSON("info", "SafeGAI camera simulator starting", map[string]string{
		"addr":     addr,
		"gateway":  gatewayURL,
		"cameras":  fmt.Sprintf("%d", numCameras),
		"zones":    fmt.Sprintf("%d", totalZones),
		"interval": interval.String(),
	})

	state.startTime = time.Now()
	state.running = true

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start event generation
	go generateEvents(ctx, interval, gatewayURL)

	// Start health/metrics HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/events", handleEvents)

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
	logJSON("info", "Shutting down camera simulator", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	state.mu.Lock()
	state.running = false
	state.mu.Unlock()

	logJSON("info", fmt.Sprintf("Camera simulator stopped. Total events sent: %d", state.totalSent), nil)
}

func generateEvents(ctx context.Context, interval time.Duration, _ string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	eventTypes := []string{"PERSON_DETECTED", "PERSON_LEFT", "ZONE_OCCUPIED", "ZONE_VACANT"}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cameraIdx := rand.Intn(numCameras) + 1
			zoneIdx := rand.Intn(zonesPerCamera) + 1
			evtType := eventTypes[rand.Intn(len(eventTypes))]

			state.mu.Lock()
			state.seqNo++
			seq := state.seqNo

			event := CameraEvent{
				CameraID:    fmt.Sprintf("CAM-%02d", cameraIdx),
				ZoneID:      fmt.Sprintf("ZONE-%02d-%02d", cameraIdx, zoneIdx),
				EventType:   evtType,
				Timestamp:   time.Now().UTC(),
				PersonCount: personCount(evtType),
				Confidence:  0.85 + rand.Float64()*0.15,
				FrameID:     fmt.Sprintf("frame-%d", seq),
				SequenceNo:  seq,
			}

			// Keep last 100 events for inspection
			state.events = append(state.events, event)
			if len(state.events) > 100 {
				state.events = state.events[len(state.events)-100:]
			}
			state.totalSent++
			state.mu.Unlock()

			logJSON("debug", fmt.Sprintf("Event: camera=%s zone=%s type=%s count=%d",
				event.CameraID, event.ZoneID, event.EventType, event.PersonCount), nil)
		}
	}
}

func personCount(evtType string) int {
	switch evtType {
	case "PERSON_DETECTED", "ZONE_OCCUPIED":
		return rand.Intn(3) + 1
	case "PERSON_LEFT":
		return rand.Intn(2)
	default:
		return 0
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"version": version,
		"uptime":  time.Since(state.startTime).Truncate(time.Second).String(),
		"cameras": numCameras,
		"zones":   totalZones,
	})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	totalSent := state.totalSent
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalEventsSent": totalSent,
		"uptime":          time.Since(state.startTime).Truncate(time.Second).String(),
	})
}

func handleEvents(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	events := make([]CameraEvent, len(state.events))
	copy(events, state.events)
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
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
		"component": "camera-sim",
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
