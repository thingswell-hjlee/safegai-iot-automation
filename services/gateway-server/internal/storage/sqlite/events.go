package sqlite

/*
#include <sqlite3.h>
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage"
)

// InsertEvent stores a new safety event. Returns error if eventId already exists (idempotent).
func (s *Store) InsertEvent(ctx context.Context, event *events.SafetyEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check idempotency: reject duplicate event IDs
	existing := s.queryIntLocked(fmt.Sprintf("SELECT COUNT(*) FROM events WHERE event_id='%s'", escapeSQLString(event.EventID)))
	if existing > 0 {
		return fmt.Errorf("duplicate event: %s", event.EventID)
	}

	// Event ordering guard: reject events with sequence_no <= last known for same device
	if event.SequenceNo > 0 {
		lastSeq := s.queryIntLocked(fmt.Sprintf(
			"SELECT COALESCE(MAX(sequence_no), 0) FROM events WHERE device_id='%s'",
			escapeSQLString(event.DeviceID)))
		if int64(lastSeq) >= event.SequenceNo {
			return fmt.Errorf("out-of-order event: device=%s seq=%d <= last=%d", event.DeviceID, event.SequenceNo, lastSeq)
		}
	}

	// Stale event guard: reject events older than stale threshold
	if !event.ObservedAt.IsZero() {
		age := time.Since(event.ObservedAt)
		if age > 60*time.Second {
			return fmt.Errorf("stale event: age=%v exceeds threshold", age)
		}
	}

	actions := strings.Join(event.Actions, ",")
	sql := fmt.Sprintf(
		`INSERT INTO events (event_id, correlation_id, tenant_id, site_id, gateway_id, device_id, zone_id, source, event_type, severity, occupancy_state, equipment_state, quality, observed_at, received_at, sequence_no, payload, camera_id, image_key, actions)
		 VALUES ('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s',%d,NULL,'%s','%s','%s')`,
		escapeSQLString(event.EventID),
		escapeSQLString(event.CorrelationID),
		escapeSQLString(event.TenantID),
		escapeSQLString(event.SiteID),
		escapeSQLString(event.GatewayID),
		escapeSQLString(event.DeviceID),
		escapeSQLString(event.ZoneID),
		escapeSQLString(event.Source),
		escapeSQLString(string(event.Source)),
		escapeSQLString(string(event.Severity)),
		escapeSQLString(string(event.OccupancyState)),
		escapeSQLString(string(event.EquipmentState)),
		escapeSQLString(string(event.Quality)),
		event.ObservedAt.UTC().Format(time.RFC3339),
		event.ReceivedAt.UTC().Format(time.RFC3339),
		event.SequenceNo,
		escapeSQLString(event.CameraID),
		escapeSQLString(event.ImageKey),
		escapeSQLString(actions),
	)

	return s.execLocked(sql)
}

// GetEvent retrieves an event by its ID.
func (s *Store) GetEvent(ctx context.Context, eventID string) (*events.SafetyEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := fmt.Sprintf("SELECT event_id, correlation_id, tenant_id, site_id, gateway_id, device_id, zone_id, source, severity, occupancy_state, equipment_state, quality, observed_at, received_at, sequence_no, camera_id, ack_by, ack_at, resolved_by, resolved_at, classification FROM events WHERE event_id='%s'",
		escapeSQLString(eventID))

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	if C.sqlite3_step(stmt) != C.SQLITE_ROW {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}

	event := &events.SafetyEvent{}
	event.EventID = columnText(stmt, 0)
	event.CorrelationID = columnText(stmt, 1)
	event.TenantID = columnText(stmt, 2)
	event.SiteID = columnText(stmt, 3)
	event.GatewayID = columnText(stmt, 4)
	event.DeviceID = columnText(stmt, 5)
	event.ZoneID = columnText(stmt, 6)
	event.Source = columnText(stmt, 7)
	event.Severity = events.Severity(columnText(stmt, 8))
	event.OccupancyState = events.OccupancyState(columnText(stmt, 9))
	event.EquipmentState = events.EquipmentState(columnText(stmt, 10))
	event.Quality = events.Quality(columnText(stmt, 11))
	event.ObservedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 12))
	event.ReceivedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 13))
	event.SequenceNo = int64(C.sqlite3_column_int64(stmt, 14))
	event.CameraID = columnText(stmt, 15)
	event.AckBy = columnText(stmt, 16)
	if ackAt := columnText(stmt, 17); ackAt != "" {
		t, _ := time.Parse(time.RFC3339, ackAt)
		event.AckAt = &t
	}
	event.ResolvedBy = columnText(stmt, 18)
	if resolvedAt := columnText(stmt, 19); resolvedAt != "" {
		t, _ := time.Parse(time.RFC3339, resolvedAt)
		event.ResolvedAt = &t
	}
	event.Classification = columnText(stmt, 20)

	return event, nil
}

// ListEvents retrieves events with pagination.
func (s *Store) ListEvents(ctx context.Context, opts storage.ListOptions) ([]*events.SafetyEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	sql := fmt.Sprintf("SELECT event_id, correlation_id, tenant_id, site_id, gateway_id, device_id, zone_id, source, severity, occupancy_state, equipment_state, quality, observed_at, received_at, sequence_no, camera_id FROM events ORDER BY received_at DESC LIMIT %d OFFSET %d",
		limit, opts.Offset)

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	var result []*events.SafetyEvent
	for C.sqlite3_step(stmt) == C.SQLITE_ROW {
		event := &events.SafetyEvent{}
		event.EventID = columnText(stmt, 0)
		event.CorrelationID = columnText(stmt, 1)
		event.TenantID = columnText(stmt, 2)
		event.SiteID = columnText(stmt, 3)
		event.GatewayID = columnText(stmt, 4)
		event.DeviceID = columnText(stmt, 5)
		event.ZoneID = columnText(stmt, 6)
		event.Source = columnText(stmt, 7)
		event.Severity = events.Severity(columnText(stmt, 8))
		event.OccupancyState = events.OccupancyState(columnText(stmt, 9))
		event.EquipmentState = events.EquipmentState(columnText(stmt, 10))
		event.Quality = events.Quality(columnText(stmt, 11))
		event.ObservedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 12))
		event.ReceivedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 13))
		event.SequenceNo = int64(C.sqlite3_column_int64(stmt, 14))
		event.CameraID = columnText(stmt, 15)
		result = append(result, event)
	}

	return result, nil
}

// AckEvent acknowledges an event.
func (s *Store) AckEvent(ctx context.Context, eventID string, actor string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sql := fmt.Sprintf("UPDATE events SET ack_by='%s', ack_at='%s' WHERE event_id='%s'",
		escapeSQLString(actor), at.UTC().Format(time.RFC3339), escapeSQLString(eventID))
	return s.execLocked(sql)
}

// ResolveEvent resolves an event.
func (s *Store) ResolveEvent(ctx context.Context, eventID string, actor string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sql := fmt.Sprintf("UPDATE events SET resolved_by='%s', resolved_at='%s' WHERE event_id='%s'",
		escapeSQLString(actor), at.UTC().Format(time.RFC3339), escapeSQLString(eventID))
	return s.execLocked(sql)
}

// ClassifyEvent assigns a classification label.
func (s *Store) ClassifyEvent(ctx context.Context, eventID string, classification string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sql := fmt.Sprintf("UPDATE events SET classification='%s' WHERE event_id='%s'",
		escapeSQLString(classification), escapeSQLString(eventID))
	return s.execLocked(sql)
}

// columnText safely extracts a text column value from a SQLite statement.
func columnText(stmt *C.sqlite3_stmt, col C.int) string {
	p := C.sqlite3_column_text(stmt, col)
	if p == nil {
		return ""
	}
	return C.GoString((*C.char)(unsafe.Pointer(p)))
}

// escapeSQLString escapes single quotes in SQL string values.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
