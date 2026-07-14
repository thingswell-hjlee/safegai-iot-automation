// Package main is the entry point for the SafeGAI edge gateway server.
// It provides health endpoints, structured JSON logging, configuration loading,
// local REST API with RBAC, cloud outbox sync, and graceful shutdown.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/auth"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/httpapi"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/observability"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage/memory"
)

const (
	defaultAddr     = ":8080"
	version         = "0.1.0"
	shutdownTimeout = 10 * time.Second
)

// config holds the gateway server configuration.
type config struct {
	Addr      string `json:"addr"`
	GatewayID string `json:"gatewayId"`
	SiteID    string `json:"siteId"`
	TenantID  string `json:"tenantId"`
}

// logEntry represents a structured JSON log line.
type logEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Component string `json:"component,omitempty"`
}

// healthResponse is the JSON body returned by health endpoints.
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

var (
	startTime = time.Now()
	ready     atomic.Bool
)

func main() {
	cfg := loadConfig()

	logJSON("info", "SafeGAI edge gateway starting", "main")
	logJSON("info", fmt.Sprintf("Listening on %s", cfg.Addr), "main")

	// Initialize storage (in-memory for now; SQLite when driver is available)
	store := memory.NewStore()

	// Initialize session store with a secret (from env in production)
	sessionSecret := []byte(os.Getenv("SAFEGAI_SESSION_SECRET"))
	if len(sessionSecret) == 0 {
		sessionSecret = []byte("default-dev-secret-do-not-use-in-prod")
	}
	sessionStore := auth.NewSessionStore(sessionSecret)

	// Initialize health collector
	healthCollector := observability.NewHealthCollector()

	// Initialize HTTP API
	handlers := httpapi.NewHandlers(store, sessionStore, healthCollector)
	router := httpapi.NewRouter(handlers, sessionStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", handleLive)
	mux.HandleFunc("/health/ready", handleReady)
	mux.Handle("/api/", router)

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Mark server as ready once listening starts.
	ready.Store(true)

	// Start server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	logJSON("info", "SafeGAI edge gateway ready", "main")

	// Wait for shutdown signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		logJSON("error", fmt.Sprintf("Server error: %v", err), "main")
		os.Exit(1)
	case <-ctx.Done():
		logJSON("info", "Shutdown signal received", "main")
	}

	// Graceful shutdown.
	ready.Store(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logJSON("error", fmt.Sprintf("Shutdown error: %v", err), "main")
		os.Exit(1)
	}

	logJSON("info", "SafeGAI edge gateway stopped", "main")
}

// loadConfig loads configuration from environment variables with defaults.
func loadConfig() config {
	cfg := config{
		Addr:      defaultAddr,
		GatewayID: "gw-default",
		SiteID:    "site-default",
		TenantID:  "tenant-default",
	}

	if addr := os.Getenv("SAFEGAI_LISTEN_ADDR"); addr != "" {
		cfg.Addr = addr
	}
	if gw := os.Getenv("SAFEGAI_GATEWAY_ID"); gw != "" {
		cfg.GatewayID = gw
	}
	if site := os.Getenv("SAFEGAI_SITE_ID"); site != "" {
		cfg.SiteID = site
	}
	if tenant := os.Getenv("SAFEGAI_TENANT_ID"); tenant != "" {
		cfg.TenantID = tenant
	}

	return cfg
}

// handleLive responds to liveness probes. Always returns 200 if the process is running.
func handleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeHealth(w, "healthy")
}

// handleReady responds to readiness probes. Returns 200 only when the server is ready.
func handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !ready.Load() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		resp := healthResponse{
			Status:  "unhealthy",
			Version: version,
			Uptime:  time.Since(startTime).Truncate(time.Second).String(),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	writeHealth(w, "healthy")
}

// writeHealth writes a JSON health response.
func writeHealth(w http.ResponseWriter, status string) {
	w.Header().Set("Content-Type", "application/json")
	resp := healthResponse{
		Status:  status,
		Version: version,
		Uptime:  time.Since(startTime).Truncate(time.Second).String(),
	}
	json.NewEncoder(w).Encode(resp)
}

// logJSON writes a structured JSON log entry to stderr.
func logJSON(level, message, component string) {
	entry := logEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Component: component,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("log marshal error: %v", err)
		return
	}
	fmt.Fprintln(os.Stderr, string(data))
}
