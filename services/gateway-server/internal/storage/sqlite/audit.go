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
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

// InsertAudit stores a new audit log entry.
func (s *Store) InsertAudit(ctx context.Context, entry *events.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sql := fmt.Sprintf(
		`INSERT INTO audit_logs (id, timestamp, actor, role, action, target, detail, ip)
		 VALUES ('%s','%s','%s','%s','%s','%s','%s','%s')`,
		escapeSQLString(entry.ID),
		entry.Timestamp.UTC().Format(time.RFC3339),
		escapeSQLString(entry.Actor),
		escapeSQLString(entry.Role),
		escapeSQLString(entry.Action),
		escapeSQLString(entry.Target),
		escapeSQLString(entry.Detail),
		escapeSQLString(entry.IP),
	)
	return s.execLocked(sql)
}

// ListAudits retrieves audit entries with pagination.
func (s *Store) ListAudits(ctx context.Context, opts storage.ListOptions) ([]*events.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	sql := fmt.Sprintf("SELECT id, timestamp, actor, role, action, target, detail, ip FROM audit_logs ORDER BY timestamp DESC LIMIT %d OFFSET %d",
		limit, opts.Offset)

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	var result []*events.AuditEntry
	for C.sqlite3_step(stmt) == C.SQLITE_ROW {
		entry := &events.AuditEntry{
			ID:     columnText(stmt, 0),
			Actor:  columnText(stmt, 2),
			Role:   columnText(stmt, 3),
			Action: columnText(stmt, 4),
			Target: columnText(stmt, 5),
			Detail: columnText(stmt, 6),
			IP:     columnText(stmt, 7),
		}
		entry.Timestamp, _ = time.Parse(time.RFC3339, columnText(stmt, 1))
		result = append(result, entry)
	}

	return result, nil
}
