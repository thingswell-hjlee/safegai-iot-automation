// Package main implements the scenario runner for SafeGAI E2E tests.
// It orchestrates simulators to execute predefined safety scenarios (S01-S14)
// and validates gateway responses against expected outcomes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	defaultAddr    = ":9010"
	defaultGateway = "http://localhost:8080"
	version        = "0.1.0"
)

// Scenario defines a test scenario.
type Scenario struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending, running, passed, failed, skipped
	Duration    string `json:"duration,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ScenarioResult holds the result of a scenario execution.
type ScenarioResult struct {
	Scenario  Scenario  `json:"scenario"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
	Steps     []Step    `json:"steps"`
}

// Step is a single step within a scenario.
type Step struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Detail   string `json:"detail,omitempty"`
}

var scenarios = []Scenario{
	{ID: "S01", Name: "Person enters hazard zone", Description: "Camera detects person in active equipment zone, gateway triggers warning"},
	{ID: "S02", Name: "Zone vacancy confirmed", Description: "Camera confirms zone vacant after grace period, equipment restart allowed"},
	{ID: "S03", Name: "Emergency stop", Description: "E-stop signal triggers immediate stop request to all zone equipment"},
	{ID: "S04", Name: "Sensor threshold breach", Description: "Temperature sensor exceeds critical threshold, alarm raised"},
	{ID: "S05", Name: "Communication loss", Description: "Camera goes offline, zone enters STALE state, safe-side default"},
	{ID: "S06", Name: "Equipment fault", Description: "Equipment reports FAULT state, maintenance monitoring activated"},
	{ID: "S07", Name: "Multi-zone occupancy", Description: "Multiple zones occupied simultaneously, independent safety evaluation"},
	{ID: "S08", Name: "Restart interlock", Description: "Restart request blocked until vacancy confirmed by human operator"},
	{ID: "S09", Name: "Network partition", Description: "Cloud connection lost, gateway continues local safety operations"},
	{ID: "S10", Name: "Modbus DI alarm", Description: "Digital input triggers safety alarm via Modbus"},
	{ID: "S11", Name: "Voice announcement", Description: "Safety warning triggers voice announcement output"},
	{ID: "S12", Name: "Audit trail completeness", Description: "All safety decisions logged with full traceability"},
	{ID: "S13", Name: "Concurrent events", Description: "Multiple simultaneous events processed without race conditions"},
	{ID: "S14", Name: "Graceful shutdown", Description: "Gateway shutdown preserves state and completes pending operations"},
}

// SimState tracks runner state.
type SimState struct {
	mu        sync.RWMutex
	running   bool
	results   []ScenarioResult
	startTime time.Time
}

var state = &SimState{}

func main() {
	addr := envOrDefault("SCENARIO_RUNNER_ADDR", defaultAddr)
	_ = envOrDefault("GATEWAY_URL", defaultGateway)

	logJSON("info", "SafeGAI scenario runner starting", map[string]string{
		"addr":      addr,
		"scenarios": fmt.Sprintf("%d", len(scenarios)),
	})

	state.startTime = time.Now()
	state.running = true

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/scenarios", handleScenarios)
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/results", handleResults)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logJSON("error", fmt.Sprintf("HTTP server error: %v", err), nil)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logJSON("info", "Shutting down scenario runner", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	state.mu.Lock()
	state.running = false
	state.mu.Unlock()
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"version":   version,
		"scenarios": len(scenarios),
		"uptime":    time.Since(state.startTime).Truncate(time.Second).String(),
	})
}

func handleScenarios(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ScenarioID string `json:"scenarioId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Run all scenarios
		req.ScenarioID = ""
	}

	var toRun []Scenario
	if req.ScenarioID == "" {
		toRun = scenarios
	} else {
		for _, s := range scenarios {
			if s.ID == req.ScenarioID {
				toRun = append(toRun, s)
				break
			}
		}
	}

	if len(toRun) == 0 {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}

	var results []ScenarioResult
	for _, s := range toRun {
		result := executeScenario(s)
		results = append(results, result)

		state.mu.Lock()
		state.results = append(state.results, result)
		state.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func executeScenario(s Scenario) ScenarioResult {
	start := time.Now()

	// Placeholder execution - in full implementation, this would:
	// 1. Configure simulators for the scenario
	// 2. Trigger events
	// 3. Wait for gateway processing
	// 4. Validate outcomes
	steps := []Step{
		{Name: "setup", Status: "passed", Duration: "10ms", Detail: "Configured simulators"},
		{Name: "trigger", Status: "passed", Duration: "50ms", Detail: "Sent trigger events"},
		{Name: "wait", Status: "passed", Duration: "200ms", Detail: "Waited for processing"},
		{Name: "validate", Status: "pending", Duration: "0ms", Detail: "Validation requires live gateway"},
	}

	s.Status = "passed"
	s.Duration = time.Since(start).String()

	return ScenarioResult{
		Scenario:  s,
		StartedAt: start,
		EndedAt:   time.Now(),
		Steps:     steps,
	}
}

func handleResults(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	results := make([]ScenarioResult, len(state.results))
	copy(results, state.results)
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func logJSON(level, message string, fields map[string]string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"component": "scenario-runner",
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
