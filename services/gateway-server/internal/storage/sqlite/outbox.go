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

// Enqueue adds a new item to the cloud outbox.
func (s *Store) Enqueue(ctx context.Context, item *events.OutboxItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sql := fmt.Sprintf(
		`INSERT INTO cloud_outbox (id, event_id, payload, status, created_at)
		 VALUES ('%s','%s',X'%x','PENDING','%s')`,
		escapeSQLString(item.ID),
		escapeSQLString(item.EventID),
		item.Payload,
		item.CreatedAt.UTC().Format(time.RFC3339),
	)
	return s.execLocked(sql)
}

// Dequeue retrieves the next pending outbox item without removing it.
func (s *Store) Dequeue(ctx context.Context) (*events.OutboxItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := "SELECT id, event_id, payload, status, created_at, retry_count, last_error FROM cloud_outbox WHERE status='PENDING' ORDER BY created_at ASC LIMIT 1"

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	if C.sqlite3_step(stmt) != C.SQLITE_ROW {
		return nil, nil // No pending items
	}

	item := &events.OutboxItem{
		ID:      columnText(stmt, 0),
		EventID: columnText(stmt, 1),
		Status:  columnText(stmt, 3),
	}

	// Read payload blob
	blobPtr := C.sqlite3_column_blob(stmt, 2)
	blobLen := C.sqlite3_column_bytes(stmt, 2)
	if blobPtr != nil && blobLen > 0 {
		item.Payload = C.GoBytes(blobPtr, blobLen)
	}

	item.CreatedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 4))
	item.RetryCount = int(C.sqlite3_column_int(stmt, 5))
	item.LastError = columnText(stmt, 6)

	return item, nil
}

// MarkSent marks an outbox item as sent.
func (s *Store) MarkSent(ctx context.Context, itemID string, sentAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sql := fmt.Sprintf("UPDATE cloud_outbox SET status='SENT', sent_at='%s' WHERE id='%s'",
		sentAt.UTC().Format(time.RFC3339), escapeSQLString(itemID))
	return s.execLocked(sql)
}

// GetPending retrieves all pending outbox items.
func (s *Store) GetPending(ctx context.Context) ([]*events.OutboxItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := "SELECT id, event_id, payload, status, created_at, retry_count, last_error FROM cloud_outbox WHERE status='PENDING' ORDER BY created_at ASC"

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	var result []*events.OutboxItem
	for C.sqlite3_step(stmt) == C.SQLITE_ROW {
		item := &events.OutboxItem{
			ID:      columnText(stmt, 0),
			EventID: columnText(stmt, 1),
			Status:  columnText(stmt, 3),
		}
		blobPtr := C.sqlite3_column_blob(stmt, 2)
		blobLen := C.sqlite3_column_bytes(stmt, 2)
		if blobPtr != nil && blobLen > 0 {
			item.Payload = C.GoBytes(blobPtr, blobLen)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 4))
		item.RetryCount = int(C.sqlite3_column_int(stmt, 5))
		item.LastError = columnText(stmt, 6)
		result = append(result, item)
	}

	return result, nil
}

// GetDepth returns the number of pending items.
func (s *Store) GetDepth(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := s.queryIntLocked("SELECT COUNT(*) FROM cloud_outbox WHERE status='PENDING'")
	return count, nil
}

// Ensure Store implements storage.OutboxStore
var _ storage.OutboxStore = (*Store)(nil)
