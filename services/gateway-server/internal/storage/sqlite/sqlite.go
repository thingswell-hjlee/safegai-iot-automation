// Package sqlite provides a SQLite WAL storage backend for the SafeGAI gateway.
// It uses CGO with the system libsqlite3 (no external Go dependencies).
//
// Key properties:
//   - WAL mode for concurrent read/write
//   - Schema migrations with version tracking
//   - Idempotent event insertion (duplicate rejection)
//   - Event ordering guard
//   - Stale event guard
//   - Restart recovery
//   - Output replay guard
package sqlite

/*
#cgo LDFLAGS: -lsqlite3
#include <sqlite3.h>
#include <stdlib.h>
#include <string.h>

// Helper to step and finalize a statement
static int step_and_finalize(sqlite3_stmt *stmt) {
    int rc = sqlite3_step(stmt);
    sqlite3_finalize(stmt);
    return rc;
}
*/
import "C"

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// Store provides SQLite-backed persistent storage.
type Store struct {
	mu   sync.RWMutex
	db   *C.sqlite3
	path string
}

// Open opens or creates a SQLite database at the given path with WAL mode.
func Open(path string) (*Store, error) {
	s := &Store{path: path}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	rc := C.sqlite3_open(cPath, &s.db)
	if rc != C.SQLITE_OK {
		errMsg := C.GoString(C.sqlite3_errmsg(s.db))
		C.sqlite3_close(s.db)
		return nil, fmt.Errorf("sqlite open: %s", errMsg)
	}

	// Enable WAL mode
	if err := s.exec("PRAGMA journal_mode=WAL"); err != nil {
		s.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	// Set busy timeout
	if err := s.exec("PRAGMA busy_timeout=5000"); err != nil {
		s.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	// Enable foreign keys
	if err := s.exec("PRAGMA foreign_keys=ON"); err != nil {
		s.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		rc := C.sqlite3_close(s.db)
		if rc != C.SQLITE_OK {
			return fmt.Errorf("sqlite close: code %d", rc)
		}
		s.db = nil
	}
	return nil
}

// Migrate runs schema migrations to the latest version.
func (s *Store) Migrate(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create migrations tracking table
	if err := s.execLocked("CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)"); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Get current version
	currentVersion := s.queryIntLocked("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")

	// Apply migrations in order
	for i, m := range migrations {
		ver := i + 1
		if ver <= currentVersion {
			continue
		}
		if err := s.execLocked(m); err != nil {
			return fmt.Errorf("migration v%d: %w", ver, err)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		insert := fmt.Sprintf("INSERT INTO schema_migrations (version, applied_at) VALUES (%d, '%s')", ver, now)
		if err := s.execLocked(insert); err != nil {
			return fmt.Errorf("record migration v%d: %w", ver, err)
		}
	}

	return nil
}

// SchemaVersion returns the current schema version.
func (s *Store) SchemaVersion() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if migrations table exists
	exists := s.queryIntLocked("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'")
	if exists == 0 {
		return 0, nil
	}

	return s.queryIntLocked("SELECT COALESCE(MAX(version), 0) FROM schema_migrations"), nil
}

// exec executes a SQL statement (acquires lock internally).
func (s *Store) exec(sql string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.execLocked(sql)
}

// execLocked executes a SQL statement (caller must hold lock).
func (s *Store) execLocked(sql string) error {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var errMsg *C.char
	rc := C.sqlite3_exec(s.db, cSQL, nil, nil, &errMsg)
	if rc != C.SQLITE_OK {
		msg := C.GoString(errMsg)
		C.sqlite3_free(unsafe.Pointer(errMsg))
		return fmt.Errorf("exec: %s (sql: %s)", msg, truncateSQL(sql))
	}
	return nil
}

// queryIntLocked queries a single integer value (caller must hold lock).
func (s *Store) queryIntLocked(sql string) int {
	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return 0
	}
	defer C.sqlite3_finalize(stmt)

	if C.sqlite3_step(stmt) == C.SQLITE_ROW {
		return int(C.sqlite3_column_int(stmt, 0))
	}
	return 0
}

// truncateSQL truncates a SQL string for error messages.
func truncateSQL(sql string) string {
	if len(sql) > 100 {
		return sql[:100] + "..."
	}
	return sql
}

// migrations defines the ordered schema migrations.
// Each entry is a complete SQL statement that advances the schema by one version.
var migrations = []string{
	// V1: Core tables
	`CREATE TABLE IF NOT EXISTS events (
		event_id TEXT PRIMARY KEY,
		correlation_id TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		site_id TEXT NOT NULL,
		gateway_id TEXT NOT NULL,
		device_id TEXT NOT NULL,
		zone_id TEXT NOT NULL,
		source TEXT NOT NULL,
		event_type TEXT NOT NULL,
		severity TEXT NOT NULL DEFAULT 'INFO',
		occupancy_state TEXT,
		equipment_state TEXT,
		quality TEXT NOT NULL DEFAULT 'GOOD',
		observed_at TEXT NOT NULL,
		received_at TEXT NOT NULL,
		sequence_no INTEGER NOT NULL DEFAULT 0,
		payload TEXT,
		camera_id TEXT,
		image_key TEXT,
		actions TEXT,
		ack_by TEXT,
		ack_at TEXT,
		resolved_by TEXT,
		resolved_at TEXT,
		classification TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(device_id, sequence_no)
	);
	CREATE INDEX IF NOT EXISTS idx_events_zone ON events(zone_id);
	CREATE INDEX IF NOT EXISTS idx_events_observed ON events(observed_at);
	CREATE INDEX IF NOT EXISTS idx_events_device_seq ON events(device_id, sequence_no);`,

	// V2: Occupancy state tracking
	`CREATE TABLE IF NOT EXISTS occupancy_states (
		zone_id TEXT PRIMARY KEY,
		state TEXT NOT NULL DEFAULT 'UNKNOWN',
		previous_state TEXT,
		last_event_id TEXT,
		last_event_at TEXT,
		vacancy_pending_since TEXT,
		vacancy_confirm_count INTEGER NOT NULL DEFAULT 0,
		camera_online INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`,

	// V3: Equipment state tracking
	`CREATE TABLE IF NOT EXISTS equipment_states (
		equipment_id TEXT PRIMARY KEY,
		state TEXT NOT NULL DEFAULT 'UNKNOWN',
		previous_state TEXT,
		last_event_id TEXT,
		last_reading_at TEXT,
		source TEXT,
		quality TEXT NOT NULL DEFAULT 'GOOD',
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`,

	// V4: Safety decisions
	`CREATE TABLE IF NOT EXISTS safety_decisions (
		decision_id TEXT PRIMARY KEY,
		correlation_id TEXT NOT NULL,
		zone_id TEXT NOT NULL,
		equipment_id TEXT,
		occupancy_state TEXT NOT NULL,
		equipment_state TEXT NOT NULL,
		decision TEXT NOT NULL,
		rule_id TEXT NOT NULL,
		actions TEXT,
		decided_at TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_decisions_zone ON safety_decisions(zone_id);
	CREATE INDEX IF NOT EXISTS idx_decisions_time ON safety_decisions(decided_at)`,

	// V5: Actuation results
	`CREATE TABLE IF NOT EXISTS actuation_results (
		command_id TEXT PRIMARY KEY,
		correlation_id TEXT NOT NULL,
		command_type TEXT NOT NULL,
		target TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'PENDING',
		executed_at TEXT,
		completed_at TEXT,
		error_msg TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_actuation_correlation ON actuation_results(correlation_id)`,

	// V6: Audit log
	`CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		timestamp TEXT NOT NULL,
		actor TEXT NOT NULL,
		role TEXT NOT NULL,
		action TEXT NOT NULL,
		target TEXT NOT NULL,
		detail TEXT,
		ip TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor)`,

	// V7: Cloud outbox
	`CREATE TABLE IF NOT EXISTS cloud_outbox (
		id TEXT PRIMARY KEY,
		event_id TEXT NOT NULL,
		topic TEXT NOT NULL DEFAULT '',
		payload BLOB NOT NULL,
		status TEXT NOT NULL DEFAULT 'PENDING',
		retry_count INTEGER NOT NULL DEFAULT 0,
		last_error TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		sent_at TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_outbox_status ON cloud_outbox(status);
	CREATE INDEX IF NOT EXISTS idx_outbox_created ON cloud_outbox(created_at)`,

	// V8: Configuration versions
	`CREATE TABLE IF NOT EXISTS config_versions (
		id TEXT PRIMARY KEY,
		version INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		created_by TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_config_active ON config_versions(active)`,

	// V9: Users
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		last_login TEXT,
		force_password_change INTEGER NOT NULL DEFAULT 1
	)`,

	// V10: Idempotency tracking for event dedup
	`CREATE TABLE IF NOT EXISTS idempotency_keys (
		key TEXT PRIMARY KEY,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		expires_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_keys(expires_at)`,

	// V11: Boot record for restart recovery
	`CREATE TABLE IF NOT EXISTS boot_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		boot_time TEXT NOT NULL,
		version TEXT NOT NULL,
		schema_version INTEGER NOT NULL,
		clean_shutdown INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`,
}
