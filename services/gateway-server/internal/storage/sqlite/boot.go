package sqlite

/*
#include <sqlite3.h>
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"fmt"
	"time"
	"unsafe"
)

// RecordBoot inserts a boot record for restart recovery tracking.
// This allows the system to detect unclean shutdowns and avoid replaying
// stale output commands from a previous boot cycle.
func (s *Store) RecordBoot(ctx context.Context, version string, schemaVersion int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	sql := fmt.Sprintf(
		`INSERT INTO boot_records (boot_time, version, schema_version, clean_shutdown)
		 VALUES ('%s','%s',%d,0)`,
		now,
		escapeSQLString(version),
		schemaVersion,
	)
	return s.execLocked(sql)
}

// MarkCleanShutdown marks the latest boot record as cleanly shut down.
func (s *Store) MarkCleanShutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.execLocked("UPDATE boot_records SET clean_shutdown=1 WHERE id=(SELECT MAX(id) FROM boot_records)")
}

// GetLastBootTime returns the boot time of the last recorded boot.
func (s *Store) GetLastBootTime(ctx context.Context) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := "SELECT boot_time FROM boot_records ORDER BY id DESC LIMIT 1"

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return time.Time{}, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	if C.sqlite3_step(stmt) != C.SQLITE_ROW {
		return time.Time{}, nil // No boot records yet
	}

	bootTimeStr := columnText(stmt, 0)
	t, err := time.Parse(time.RFC3339, bootTimeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse boot time: %w", err)
	}
	return t, nil
}

// WasCleanShutdown returns true if the last boot was cleanly shut down.
func (s *Store) WasCleanShutdown(ctx context.Context) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clean := s.queryIntLocked("SELECT clean_shutdown FROM boot_records ORDER BY id DESC LIMIT 1")
	return clean == 1, nil
}

// Ensure the Store interface includes boot record operations
var _ BootRecordStore = (*Store)(nil)

// BootRecordStore defines the interface for boot record operations.
type BootRecordStore interface {
	RecordBoot(ctx context.Context, version string, schemaVersion int) error
	MarkCleanShutdown(ctx context.Context) error
	GetLastBootTime(ctx context.Context) (time.Time, error)
	WasCleanShutdown(ctx context.Context) (bool, error)
}
