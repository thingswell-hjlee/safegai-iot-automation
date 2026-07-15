// Package main implements the equipment state simulator for SafeGAI.
// It simulates industrial equipment with state transitions
// (RUNNING, STOPPED, STARTING, STOPPING, FAULT, OFFLINE, UNKNOWN).
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
	defaultAddr     = ":9003"
	defaultInterval = 5 * time.Second
	version         = "0.1.0"
)

// EquipmentState represents equipment running states.
type EquipmentState string

const (
	StateRunning  EquipmentState = "RUNNING"
	StateStopped  EquipmentState = "STOPPED"
	StateStarting EquipmentState = "STARTING"
	StateStopping EquipmentState = "STOPPING"
	StateFault    EquipmentState = "FAULT"
	StateOffline  EquipmentState = "OFFLINE"
	StateUnknown  EquipmentState = "UNKNOWN"
)

// Equipment tracks a simulated equipment instance.
type Equipment struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	CurrentState EquipmentState `json:"currentState"`
	LastChanged  time.Time      `json:"lastChanged"`
}

// EquipmentEvent represents a state change event.
type EquipmentEvent struct {
	EquipmentID string         `json:"equipmentId"`
	State       EquipmentState `json:"state"`
	PrevState   EquipmentState `json:"prevState"`
	Timestamp   time.Time      `json:"timestamp"`
	Source      string         `json:"source"`
}

// Valid state transitions
var transitions = map[EquipmentState][]EquipmentState{
	StateRunning:  {StateRunning, StateRunning, StateRunning, StateStopping, StateFault},
	StateStopped:  {StateStopped, StateStopped, StateStarting, StateOffline},
	StateStarting: {StateRunning, StateRunning, StateFault},
	StateStopping: {StateStopped, StateStopped, StateFault},
	StateFault:    {StateFault, StateStopped, StateOffline},
	StateOffline:  {StateOffline, StateStopped},
	StateUnknown:  {StateStopped, StateRunning, StateFault},
}

var equipmentList = []Equipment{
	{ID: "EQ-PRESS-01", Name: "Hydraulic Press #1"},
	{ID: "EQ-PRESS-02", Name: "Hydraulic Press #2"},
	{ID: "EQ-CONV-01", Name: "Conveyor Belt #1"},
	{ID: "EQ-CONV-02", Name: "Conveyor Belt #2"},
	{ID: "EQ-ROBOT-01", Name: "Robot Arm #1"},
	{ID: "EQ-WELD-01", Name: "Welding Station #1"},
}

// SimState tracks simulator state.
type SimState struct {
	mu        sync.RWMutex
	running   bool
	equipment []Equipment
	events    []EquipmentEvent
	totalSent int64
	startTime time.Time
}

var state = &SimState{}

func main() {
	addr := envOrDefault("EQUIPMENT_SIM_ADDR", defaultAddr)
	interval := parseDuration(envOrDefault("EQUIPMENT_SIM_INTERVAL", "5s"), defaultInterval)

	logJSON("info", "SafeGAI equipment simulator starting", map[string]string{
		"addr":      addr,
		"equipment": fmt.Sprintf("%d", len(equipmentList)),
		"interval":  interval.String(),
	})

	// Initialize equipment with initial states
	state.startTime = time.Now()
	state.running = true
	state.equipment = make([]Equipment, len(equipmentList))
	for i, eq := range equipmentList {
		eq.CurrentState = StateStopped
		eq.LastChanged = time.Now()
		state.equipment[i] = eq
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go simulateTransitions(ctx, interval)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/status", handleStatus)
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
	logJSON("info", "Shutting down equipment simulator", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	state.mu.Lock()
	state.running = false
	state.mu.Unlock()

	logJSON("info", fmt.Sprintf("Equipment simulator stopped. Total events: %d", state.totalSent), nil)
}

func simulateTransitions(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Pick a random equipment to potentially transition
			idx := rand.Intn(len(state.equipment))

			state.mu.Lock()
			eq := &state.equipment[idx]
			prevState := eq.CurrentState

			possibleStates := transitions[prevState]
			if len(possibleStates) > 0 {
				newState := possibleStates[rand.Intn(len(possibleStates))]
				if newState != prevState {
					eq.CurrentState = newState
					eq.LastChanged = time.Now()

					event := EquipmentEvent{
						EquipmentID: eq.ID,
						State:       newState,
						PrevState:   prevState,
						Timestamp:   time.Now().UTC(),
						Source:      "equipment-sim",
					}

					state.events = append(state.events, event)
					if len(state.events) > 100 {
						state.events = state.events[len(state.events)-100:]
					}
					state.totalSent++

					logJSON("debug", fmt.Sprintf("Transition: %s %s -> %s", eq.ID, prevState, newState), nil)
				}
			}
			state.mu.Unlock()
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"version":   version,
		"uptime":    time.Since(state.startTime).Truncate(time.Second).String(),
		"equipment": len(equipmentList),
	})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	totalSent := state.totalSent
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalTransitions": totalSent,
		"uptime":           time.Since(state.startTime).Truncate(time.Second).String(),
	})
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	equipment := make([]Equipment, len(state.equipment))
	copy(equipment, state.equipment)
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(equipment)
}

func handleEvents(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	events := make([]EquipmentEvent, len(state.events))
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
		"component": "equipment-sim",
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
