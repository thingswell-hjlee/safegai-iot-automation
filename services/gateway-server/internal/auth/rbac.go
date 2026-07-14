// Package auth provides role-based access control (RBAC) for the SafeGAI gateway.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Role represents a user role in the RBAC system.
type Role string

const (
	RoleUser       Role = "USER"
	RoleOperator   Role = "OPERATOR"
	RoleMaintainer Role = "MAINTAINER"
)

// Action represents a protected action in the system.
type Action string

const (
	ActionViewEvents       Action = "VIEW_EVENTS"
	ActionAckEvents        Action = "ACK_EVENTS"
	ActionResolveEvents    Action = "RESOLVE_EVENTS"
	ActionViewCameras      Action = "VIEW_CAMERAS"
	ActionViewZones        Action = "VIEW_ZONES"
	ActionViewEquipment    Action = "VIEW_EQUIPMENT"
	ActionViewStatus       Action = "VIEW_STATUS"
	ActionViewHardware     Action = "VIEW_HARDWARE"
	ActionWorkWindows      Action = "WORK_WINDOWS"
	ActionCloseWorkWindows Action = "CLOSE_WORK_WINDOWS"
	ActionDiagnostics      Action = "DIAGNOSTICS"
	ActionIOTest           Action = "IO_TEST"
	ActionManageUsers      Action = "MANAGE_USERS"
	ActionConfigureSystem  Action = "CONFIGURE_SYSTEM"
	ActionViewAudit        Action = "VIEW_AUDIT"
	ActionRealtime         Action = "REALTIME"
)

// permissionMatrix defines which roles can perform which actions.
// Roles are hierarchical: MAINTAINER > OPERATOR > USER.
var permissionMatrix = map[Role]map[Action]bool{
	RoleUser: {
		ActionViewEvents:    true,
		ActionViewCameras:   true,
		ActionViewZones:     true,
		ActionViewEquipment: true,
		ActionViewStatus:    true,
		ActionRealtime:      true,
	},
	RoleOperator: {
		ActionViewEvents:       true,
		ActionAckEvents:        true,
		ActionResolveEvents:    true,
		ActionViewCameras:      true,
		ActionViewZones:        true,
		ActionViewEquipment:    true,
		ActionViewStatus:       true,
		ActionViewHardware:     true,
		ActionWorkWindows:      true,
		ActionCloseWorkWindows: true,
		ActionViewAudit:        true,
		ActionRealtime:         true,
	},
	RoleMaintainer: {
		ActionViewEvents:       true,
		ActionAckEvents:        true,
		ActionResolveEvents:    true,
		ActionViewCameras:      true,
		ActionViewZones:        true,
		ActionViewEquipment:    true,
		ActionViewStatus:       true,
		ActionViewHardware:     true,
		ActionWorkWindows:      true,
		ActionCloseWorkWindows: true,
		ActionDiagnostics:      true,
		ActionIOTest:           true,
		ActionManageUsers:      true,
		ActionConfigureSystem:  true,
		ActionViewAudit:        true,
		ActionRealtime:         true,
	},
}

// HasPermission checks if a role has permission to perform an action.
func HasPermission(role Role, action Action) bool {
	perms, exists := permissionMatrix[role]
	if !exists {
		return false
	}
	return perms[action]
}

// ValidRole returns true if the given string is a valid role.
func ValidRole(r string) bool {
	switch Role(r) {
	case RoleUser, RoleOperator, RoleMaintainer:
		return true
	}
	return false
}

// Session represents an authenticated user session.
type Session struct {
	SessionID string    `json:"sessionId"`
	Username  string    `json:"username"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SessionStore manages active sessions in-memory.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	secret   []byte
}

// NewSessionStore creates a new session store with the given HMAC secret.
func NewSessionStore(secret []byte) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
		secret:   secret,
	}
}

// CreateSession creates a new session for the given user.
func (ss *SessionStore) CreateSession(username string, role Role) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	now := time.Now().UTC()
	sessionID := generateSessionID(ss.secret, username, now)

	session := &Session{
		SessionID: sessionID,
		Username:  username,
		Role:      role,
		CreatedAt: now,
		ExpiresAt: now.Add(8 * time.Hour),
	}

	ss.sessions[sessionID] = session
	return session
}

// GetSession retrieves a session by ID. Returns nil if not found or expired.
func (ss *SessionStore) GetSession(sessionID string) *Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	session, exists := ss.sessions[sessionID]
	if !exists {
		return nil
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return nil
	}

	return session
}

// DeleteSession removes a session by ID.
func (ss *SessionStore) DeleteSession(sessionID string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, sessionID)
}

// generateSessionID creates a deterministic session ID using HMAC.
func generateSessionID(secret []byte, username string, now time.Time) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(username))
	mac.Write([]byte(now.Format(time.RFC3339Nano)))
	return hex.EncodeToString(mac.Sum(nil))[:48]
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const sessionContextKey contextKey = "session"

// WithSession stores a session in the context.
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSessionFromContext retrieves the session from the context.
func GetSessionFromContext(ctx context.Context) *Session {
	session, _ := ctx.Value(sessionContextKey).(*Session)
	return session
}

// AuthMiddleware validates the session token and adds the session to the context.
func AuthMiddleware(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing authorization token")
				return
			}

			session := store.GetSession(token)
			if session == nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid or expired session")
				return
			}

			ctx := WithSession(r.Context(), session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission creates middleware that checks if the session role has the given permission.
func RequirePermission(action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := GetSessionFromContext(r.Context())
			if session == nil {
				writeAuthError(w, http.StatusUnauthorized, "no session in context")
				return
			}

			if !HasPermission(session.Role, action) {
				writeAuthError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken gets the session token from the Authorization header.
// Expected format: "Bearer <token>"
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// writeAuthError writes a JSON error response for auth failures.
func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{"error": message}
	json.NewEncoder(w).Encode(resp)
}
