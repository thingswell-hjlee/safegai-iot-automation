package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.Gateway.ListenAddr != ":8080" {
		t.Errorf("expected listen addr :8080, got %s", cfg.Gateway.ListenAddr)
	}
	if cfg.Storage.WALMode != true {
		t.Error("expected WAL mode enabled by default")
	}
	if cfg.Safety.AutoRestart != false {
		t.Error("SAFETY: auto_restart must always be false")
	}
	if cfg.Safety.VacancyConfirmDuration != 3*time.Second {
		t.Errorf("expected vacancy confirm 3s, got %v", cfg.Safety.VacancyConfirmDuration)
	}
	if cfg.Adapters.Cloud.Type != "disabled" {
		t.Errorf("expected default cloud adapter disabled, got %s", cfg.Adapters.Cloud.Type)
	}
}

func TestProfileValidation(t *testing.T) {
	tests := []struct {
		profile Profile
		valid   bool
	}{
		{ProfileAWSSim, true},
		{ProfileLocalSim, true},
		{ProfileLocalLab, true},
		{ProfileLocalPilot, true},
		{Profile("invalid"), false},
		{Profile(""), false},
	}

	for _, tc := range tests {
		if tc.profile.IsValid() != tc.valid {
			t.Errorf("profile %q: expected valid=%v, got %v", tc.profile, tc.valid, tc.profile.IsValid())
		}
	}
}

func TestLoadWithEnvProfile(t *testing.T) {
	os.Setenv("SAFEGAI_PROFILE", "aws-sim")
	defer os.Unsetenv("SAFEGAI_PROFILE")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Profile != ProfileAWSSim {
		t.Errorf("expected profile aws-sim, got %s", cfg.Profile)
	}
}

func TestLoadInvalidProfile(t *testing.T) {
	os.Setenv("SAFEGAI_PROFILE", "production")
	defer os.Unsetenv("SAFEGAI_PROFILE")

	_, err := Load("")
	if err == nil {
		t.Error("expected error for invalid profile")
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("SAFEGAI_LISTEN_ADDR", ":9090")
	os.Setenv("SAFEGAI_GATEWAY_ID", "gw-test")
	os.Setenv("SAFEGAI_DB_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("SAFEGAI_LISTEN_ADDR")
		os.Unsetenv("SAFEGAI_GATEWAY_ID")
		os.Unsetenv("SAFEGAI_DB_PATH")
		os.Unsetenv("SAFEGAI_PROFILE")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Gateway.ListenAddr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.Gateway.ListenAddr)
	}
	if cfg.Gateway.ID != "gw-test" {
		t.Errorf("expected gw-test, got %s", cfg.Gateway.ID)
	}
	if cfg.Storage.Path != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %s", cfg.Storage.Path)
	}
}

func TestSafetyConstraintsEnforced(t *testing.T) {
	cfg := Default()

	// Attempt to set auto_restart to true
	cfg.Safety.AutoRestart = true
	enforceSafetyConstraints(cfg)

	if cfg.Safety.AutoRestart != false {
		t.Error("SAFETY: auto_restart must be enforced to false")
	}
}

func TestProfileFromPath(t *testing.T) {
	tests := []struct {
		path    string
		profile Profile
	}{
		{"/etc/safegai/aws-sim.yaml", ProfileAWSSim},
		{"/etc/safegai/local-sim.yaml", ProfileLocalSim},
		{"/etc/safegai/local-lab.yaml", ProfileLocalLab},
		{"/etc/safegai/local-pilot.yaml", ProfileLocalPilot},
		{"configs/common.yaml", Profile("common")},
	}

	for _, tc := range tests {
		got := profileFromPath(tc.path)
		if got != tc.profile {
			t.Errorf("profileFromPath(%q) = %q, want %q", tc.path, got, tc.profile)
		}
	}
}
