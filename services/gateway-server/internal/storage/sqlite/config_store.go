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

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
)

// GetCurrent retrieves the currently active configuration version.
func (s *Store) GetCurrent(ctx context.Context) (*events.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := "SELECT id, version, content, created_at, created_by, active FROM config_versions WHERE active=1 ORDER BY version DESC LIMIT 1"

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	if C.sqlite3_step(stmt) != C.SQLITE_ROW {
		return nil, nil // No active config
	}

	cfg := &events.ConfigVersion{
		ID:        columnText(stmt, 0),
		Version:   int(C.sqlite3_column_int(stmt, 1)),
		Content:   columnText(stmt, 2),
		CreatedBy: columnText(stmt, 4),
		Active:    C.sqlite3_column_int(stmt, 5) == 1,
	}
	cfg.CreatedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 3))

	return cfg, nil
}

// SaveVersion stores a new configuration version and marks it active.
func (s *Store) SaveVersion(ctx context.Context, cfg *events.ConfigVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Deactivate all current configs
	if err := s.execLocked("UPDATE config_versions SET active=0 WHERE active=1"); err != nil {
		return fmt.Errorf("deactivate: %w", err)
	}

	active := 0
	if cfg.Active {
		active = 1
	}

	sql := fmt.Sprintf(
		`INSERT INTO config_versions (id, version, content, created_at, created_by, active)
		 VALUES ('%s',%d,'%s','%s','%s',%d)`,
		escapeSQLString(cfg.ID),
		cfg.Version,
		escapeSQLString(cfg.Content),
		cfg.CreatedAt.UTC().Format(time.RFC3339),
		escapeSQLString(cfg.CreatedBy),
		active,
	)
	return s.execLocked(sql)
}

// ListVersions retrieves all configuration versions.
func (s *Store) ListVersions(ctx context.Context) ([]*events.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := "SELECT id, version, content, created_at, created_by, active FROM config_versions ORDER BY version DESC"

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	var result []*events.ConfigVersion
	for C.sqlite3_step(stmt) == C.SQLITE_ROW {
		cfg := &events.ConfigVersion{
			ID:        columnText(stmt, 0),
			Version:   int(C.sqlite3_column_int(stmt, 1)),
			Content:   columnText(stmt, 2),
			CreatedBy: columnText(stmt, 4),
			Active:    C.sqlite3_column_int(stmt, 5) == 1,
		}
		cfg.CreatedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 3))
		result = append(result, cfg)
	}

	return result, nil
}
