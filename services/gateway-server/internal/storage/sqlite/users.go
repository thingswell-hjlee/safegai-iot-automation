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

// GetUser retrieves a user by username.
func (s *Store) GetUser(ctx context.Context, username string) (*events.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := fmt.Sprintf("SELECT id, username, password_hash, role, created_at, last_login, force_password_change FROM users WHERE username='%s'",
		escapeSQLString(username))

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	if C.sqlite3_step(stmt) != C.SQLITE_ROW {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	user := &events.User{
		ID:                  columnText(stmt, 0),
		Username:            columnText(stmt, 1),
		PasswordHash:        columnText(stmt, 2),
		Role:                columnText(stmt, 3),
		ForcePasswordChange: C.sqlite3_column_int(stmt, 6) == 1,
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 4))
	if lastLogin := columnText(stmt, 5); lastLogin != "" {
		t, _ := time.Parse(time.RFC3339, lastLogin)
		user.LastLogin = &t
	}

	return user, nil
}

// ListUsers retrieves all users.
func (s *Store) ListUsers(ctx context.Context) ([]*events.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sql := "SELECT id, username, role, created_at, last_login, force_password_change FROM users ORDER BY username"

	cSQL := C.CString(sql)
	defer C.free(unsafe.Pointer(cSQL))

	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(s.db, cSQL, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, fmt.Errorf("prepare: %s", C.GoString(C.sqlite3_errmsg(s.db)))
	}
	defer C.sqlite3_finalize(stmt)

	var result []*events.User
	for C.sqlite3_step(stmt) == C.SQLITE_ROW {
		user := &events.User{
			ID:                  columnText(stmt, 0),
			Username:            columnText(stmt, 1),
			Role:                columnText(stmt, 2),
			ForcePasswordChange: C.sqlite3_column_int(stmt, 5) == 1,
		}
		user.CreatedAt, _ = time.Parse(time.RFC3339, columnText(stmt, 3))
		if lastLogin := columnText(stmt, 4); lastLogin != "" {
			t, _ := time.Parse(time.RFC3339, lastLogin)
			user.LastLogin = &t
		}
		result = append(result, user)
	}

	return result, nil
}

// CreateUser stores a new user. Returns error if username already exists.
func (s *Store) CreateUser(ctx context.Context, user *events.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	forcePwdChange := 0
	if user.ForcePasswordChange {
		forcePwdChange = 1
	}

	sql := fmt.Sprintf(
		`INSERT INTO users (id, username, password_hash, role, created_at, force_password_change)
		 VALUES ('%s','%s','%s','%s','%s',%d)`,
		escapeSQLString(user.ID),
		escapeSQLString(user.Username),
		escapeSQLString(user.PasswordHash),
		escapeSQLString(user.Role),
		user.CreatedAt.UTC().Format(time.RFC3339),
		forcePwdChange,
	)
	return s.execLocked(sql)
}

// UpdateUser updates an existing user record.
func (s *Store) UpdateUser(ctx context.Context, user *events.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	forcePwdChange := 0
	if user.ForcePasswordChange {
		forcePwdChange = 1
	}

	lastLogin := ""
	if user.LastLogin != nil {
		lastLogin = user.LastLogin.UTC().Format(time.RFC3339)
	}

	sql := fmt.Sprintf(
		`UPDATE users SET password_hash='%s', role='%s', last_login='%s', force_password_change=%d WHERE id='%s'`,
		escapeSQLString(user.PasswordHash),
		escapeSQLString(user.Role),
		lastLogin,
		forcePwdChange,
		escapeSQLString(user.ID),
	)
	return s.execLocked(sql)
}
