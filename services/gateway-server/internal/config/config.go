// Package config provides configuration loading for the SafeGAI gateway.
// Configuration is loaded from YAML files in a layered manner:
//  1. common.yaml (base defaults)
//  2. <profile>.yaml (profile-specific overrides)
//  3. Environment variables (final overrides)
//
// The configuration profile is selected via --config flag or SAFEGAI_PROFILE env var.
// Valid profiles: aws-sim, local-sim, local-lab, local-pilot
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Profile identifies a deployment profile.
type Profile string

const (
	ProfileAWSSim     Profile = "aws-sim"
	ProfileLocalSim   Profile = "local-sim"
	ProfileLocalLab   Profile = "local-lab"
	ProfileLocalPilot Profile = "local-pilot"
)

// ValidProfiles contains all valid profile identifiers.
var ValidProfiles = []Profile{
	ProfileAWSSim,
	ProfileLocalSim,
	ProfileLocalLab,
	ProfileLocalPilot,
}

// IsValid returns true if the profile is recognized.
func (p Profile) IsValid() bool {
	for _, v := range ValidProfiles {
		if p == v {
			return true
		}
	}
	return false
}

// Config holds the complete gateway configuration.
type Config struct {
	Profile  Profile        `json:"profile"`
	Gateway  GatewayConfig  `json:"gateway"`
	Adapters AdaptersConfig `json:"adapters"`
	Storage  StorageConfig  `json:"storage"`
	Safety   SafetyConfig   `json:"safety"`
	Logging  LoggingConfig  `json:"logging"`
	Health   HealthConfig   `json:"health"`
	Audit    AuditConfig    `json:"audit"`
	Outbox   OutboxConfig   `json:"outbox"`
}

// GatewayConfig holds core gateway settings.
type GatewayConfig struct {
	ID              string        `json:"id"`
	SiteID          string        `json:"site_id"`
	TenantID        string        `json:"tenant_id"`
	ListenAddr      string        `json:"listen_addr"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	Version         string        `json:"version"`
}

// AdaptersConfig holds adapter selection and configuration.
type AdaptersConfig struct {
	Camera       AdapterEntry `json:"camera"`
	Sensor       AdapterEntry `json:"sensor"`
	Equipment    AdapterEntry `json:"equipment"`
	Output       AdapterEntry `json:"output"`
	Media        AdapterEntry `json:"media"`
	Cloud        AdapterEntry `json:"cloud"`
	Notification AdapterEntry `json:"notification"`
}

// AdapterEntry holds the type and parameters for a single adapter.
type AdapterEntry struct {
	Type       string            `json:"type"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// StorageConfig holds database configuration.
type StorageConfig struct {
	Type          string `json:"type"`
	Path          string `json:"path"`
	WALMode       bool   `json:"wal_mode"`
	BusyTimeoutMs int    `json:"busy_timeout_ms"`
	MaxOpenConns  int    `json:"max_open_conns"`
}

// SafetyConfig holds fixed safety parameters.
// These values are NOT configurable per-profile for safety reasons.
type SafetyConfig struct {
	VacancyConfirmDuration time.Duration `json:"vacancy_confirm_duration"`
	VacancyConfirmSamples  int           `json:"vacancy_confirm_samples"`
	StaleTimeout           time.Duration `json:"stale_timeout"`
	CameraOfflineTimeout   time.Duration `json:"camera_offline_timeout"`
	DedupWindow            time.Duration `json:"dedup_window"`
	// SAFETY: AutoRestart is ALWAYS false. This field exists for documentation only.
	AutoRestart bool `json:"auto_restart"`
}

// LoggingConfig holds structured logging parameters.
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

// HealthConfig holds health check endpoint parameters.
type HealthConfig struct {
	LivenessPath     string        `json:"liveness_path"`
	ReadinessPath    string        `json:"readiness_path"`
	WatchdogInterval time.Duration `json:"watchdog_interval"`
}

// AuditConfig holds audit trail parameters.
type AuditConfig struct {
	Enabled       bool `json:"enabled"`
	RetentionDays int  `json:"retention_days"`
}

// OutboxConfig holds cloud sync outbox parameters.
type OutboxConfig struct {
	MaxQueueSize int           `json:"max_queue_size"`
	SyncInterval time.Duration `json:"sync_interval"`
	BaseBackoff  time.Duration `json:"base_backoff"`
	MaxBackoff   time.Duration `json:"max_backoff"`
}

// Default returns the default configuration matching common.yaml.
func Default() *Config {
	return &Config{
		Profile: ProfileLocalSim,
		Gateway: GatewayConfig{
			ID:              "gw-default",
			SiteID:          "site-default",
			TenantID:        "tenant-default",
			ListenAddr:      ":8080",
			ShutdownTimeout: 10 * time.Second,
			Version:         "0.1.0",
		},
		Adapters: AdaptersConfig{
			Camera:       AdapterEntry{Type: "simulator"},
			Sensor:       AdapterEntry{Type: "simulator"},
			Equipment:    AdapterEntry{Type: "simulator"},
			Output:       AdapterEntry{Type: "simulator"},
			Media:        AdapterEntry{Type: "simulator"},
			Cloud:        AdapterEntry{Type: "disabled"},
			Notification: AdapterEntry{Type: "log"},
		},
		Storage: StorageConfig{
			Type:          "sqlite",
			Path:          "/var/lib/safegai/data/safegai.db",
			WALMode:       true,
			BusyTimeoutMs: 5000,
			MaxOpenConns:  1,
		},
		Safety: SafetyConfig{
			VacancyConfirmDuration: 3 * time.Second,
			VacancyConfirmSamples:  3,
			StaleTimeout:           10 * time.Second,
			CameraOfflineTimeout:   30 * time.Second,
			DedupWindow:            2 * time.Second,
			AutoRestart:            false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stderr",
		},
		Health: HealthConfig{
			LivenessPath:     "/health/live",
			ReadinessPath:    "/health/ready",
			WatchdogInterval: 15 * time.Second,
		},
		Audit: AuditConfig{
			Enabled:       true,
			RetentionDays: 90,
		},
		Outbox: OutboxConfig{
			MaxQueueSize: 1000,
			SyncInterval: 5 * time.Second,
			BaseBackoff:  1 * time.Second,
			MaxBackoff:   60 * time.Second,
		},
	}
}

// Load loads configuration from the specified config file path.
// If configPath is empty, uses SAFEGAI_CONFIG_PATH env var.
// Falls back to defaults if no file is found.
func Load(configPath string) (*Config, error) {
	cfg := Default()

	// Determine config file path
	if configPath == "" {
		configPath = os.Getenv("SAFEGAI_CONFIG_PATH")
	}

	// If a config path is given, determine the profile from the file
	if configPath != "" {
		profile := profileFromPath(configPath)
		if profile != "" && profile.IsValid() {
			cfg.Profile = profile
		}
	}

	// Override profile from environment
	if envProfile := os.Getenv("SAFEGAI_PROFILE"); envProfile != "" {
		p := Profile(envProfile)
		if !p.IsValid() {
			return nil, fmt.Errorf("invalid profile: %s (valid: %s)", envProfile, validProfilesString())
		}
		cfg.Profile = p
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Enforce safety constraints
	enforceSafetyConstraints(cfg)

	return cfg, nil
}

// profileFromPath extracts profile name from a config file path.
// e.g., "/etc/safegai/aws-sim.yaml" -> "aws-sim"
func profileFromPath(path string) Profile {
	base := filepath.Base(path)
	// Remove extension
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return Profile(name)
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SAFEGAI_LISTEN_ADDR"); v != "" {
		cfg.Gateway.ListenAddr = v
	}
	if v := os.Getenv("SAFEGAI_GATEWAY_ID"); v != "" {
		cfg.Gateway.ID = v
	}
	if v := os.Getenv("SAFEGAI_SITE_ID"); v != "" {
		cfg.Gateway.SiteID = v
	}
	if v := os.Getenv("SAFEGAI_TENANT_ID"); v != "" {
		cfg.Gateway.TenantID = v
	}
	if v := os.Getenv("SAFEGAI_DB_PATH"); v != "" {
		cfg.Storage.Path = v
	}
	if v := os.Getenv("SAFEGAI_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
}

// enforceSafetyConstraints ensures safety-critical parameters are never overridden.
// SAFETY: These constraints are FIXED regardless of profile or environment.
func enforceSafetyConstraints(cfg *Config) {
	// SAFETY: auto_restart is ALWAYS false
	cfg.Safety.AutoRestart = false

	// SAFETY: vacancy confirm duration must be positive
	if cfg.Safety.VacancyConfirmDuration <= 0 {
		cfg.Safety.VacancyConfirmDuration = 3 * time.Second
	}

	// SAFETY: stale timeout must be positive
	if cfg.Safety.StaleTimeout <= 0 {
		cfg.Safety.StaleTimeout = 10 * time.Second
	}
}

// validProfilesString returns a comma-separated list of valid profiles.
func validProfilesString() string {
	parts := make([]string, len(ValidProfiles))
	for i, p := range ValidProfiles {
		parts[i] = string(p)
	}
	return strings.Join(parts, ", ")
}
