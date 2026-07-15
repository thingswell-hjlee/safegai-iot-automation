// Package main implements the output device simulator for SafeGAI.
// It simulates 5 output types: WARNING_LIGHT, WARNING_SIREN,
// VOICE_ANNOUNCE, STOP_REQUEST, DIGITAL_OUTPUT_TEST.
// Accepts commands via HTTP POST and tracks execution history.
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
	defaultAddr = ":9004"
	version     = "0.1.0"
)

// OutputCommand mirrors ports.OutputCommand.
type OutputCommand struct {
	CommandID     string            `json:"commandId"`
	CorrelationID string            `json:"correlationId"`
	CommandType   string            `json:"commandType"`
	Target        string            `json:"target"`
	Parameters    map[string]string `json:"parameters,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	Timeout       string            `json:"timeout"`
}

// OutputResult mirrors ports.OutputResult.
type OutputResult struct {
	CommandID  string    `json:"commandId"`
	Success    bool      `json:"success"`
	ExecutedAt time.Time `json:"executedAt"`
	ErrorMsg   string    `json:"errorMsg,omitempty"`
	Latency    string    `json:"latency"`
}

var supportedTypes = []string{
	"WARNING_LIGHT",
	"WARNING_SIREN",
	"VOICE_ANNOUNCE",
	"STOP_REQUEST",
	"DIGITAL_OUTPUT_TEST",
}

// SimState tracks simulator state.
type SimState struct {
	mu        sync.RWMutex
	running   bool
	results   []OutputResult
	totalExec int64
	startTime time.Time
	// Simulate fault injection rate (0.0 = never fail)
	faultRate float64
}

var state = &SimState{}

func main() {
	addr := envOrDefault("OUTPUT_SIM_ADDR", defaultAddr)

	logJSON("info", "SafeGAI output simulator starting", map[string]string{
		"addr":  addr,
		"types": fmt.Sprintf("%d", len(supportedTypes)),
	})

	state.startTime = time.Now()
	state.running = true
	state.faultRate = 0.05 // 5% failure rate for testing

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/execute", handleExecute)
	mux.HandleFunc("/capabilities", handleCapabilities)
	mux.HandleFunc("/history", handleHistory)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logJSON("error", fmt.Sprintf("HTTP server error: %v", err), nil)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logJSON("info", "Shutting down output simulator", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	state.mu.Lock()
	state.running = false
	state.mu.Unlock()

	logJSON("info", fmt.Sprintf("Output simulator stopped. Total executions: %d", state.totalExec), nil)
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cmd OutputCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate command type
	valid := false
	for _, t := range supportedTypes {
		if cmd.CommandType == t {
			valid = true
			break
		}
	}
	if !valid {
		http.Error(w, fmt.Sprintf("Unsupported command type: %s", cmd.CommandType), http.StatusBadRequest)
		return
	}

	// Simulate execution latency
	latency := time.Duration(50+rand.Intn(200)) * time.Millisecond
	time.Sleep(latency)

	// Simulate possible failure
	success := true
	errorMsg := ""
	state.mu.RLock()
	faultRate := state.faultRate
	state.mu.RUnlock()

	if rand.Float64() < faultRate {
		success = false
		errorMsg = "simulated output device timeout"
	}

	result := OutputResult{
		CommandID:  cmd.CommandID,
		Success:    success,
		ExecutedAt: time.Now().UTC(),
		ErrorMsg:   errorMsg,
		Latency:    latency.String(),
	}

	state.mu.Lock()
	state.results = append(state.results, result)
	if len(state.results) > 200 {
		state.results = state.results[len(state.results)-200:]
	}
	state.totalExec++
	state.mu.Unlock()

	logJSON("info", fmt.Sprintf("Executed: type=%s target=%s success=%v latency=%s",
		cmd.CommandType, cmd.Target, success, latency), nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleCapabilities(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"supportedTypes": supportedTypes,
	})
}

func handleHistory(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	results := make([]OutputResult, len(state.results))
	copy(results, state.results)
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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
		"types":   len(supportedTypes),
	})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	totalExec := state.totalExec
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalExecutions": totalExec,
		"uptime":          time.Since(state.startTime).Truncate(time.Second).String(),
	})
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
		"component": "output-sim",
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
