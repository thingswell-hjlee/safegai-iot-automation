package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserCannotAckEvents(t *testing.T) {
	if HasPermission(RoleUser, ActionAckEvents) {
		t.Error("USER should not be able to ACK events")
	}
}

func TestUserCannotAccessMaintenance(t *testing.T) {
	if HasPermission(RoleUser, ActionDiagnostics) {
		t.Error("USER should not be able to access diagnostics")
	}
	if HasPermission(RoleUser, ActionIOTest) {
		t.Error("USER should not be able to perform I/O tests")
	}
}

func TestOperatorCanAckButNotConfigureIO(t *testing.T) {
	if !HasPermission(RoleOperator, ActionAckEvents) {
		t.Error("OPERATOR should be able to ACK events")
	}
	if !HasPermission(RoleOperator, ActionResolveEvents) {
		t.Error("OPERATOR should be able to resolve events")
	}
	if HasPermission(RoleOperator, ActionIOTest) {
		t.Error("OPERATOR should not be able to perform I/O tests")
	}
	if HasPermission(RoleOperator, ActionConfigureSystem) {
		t.Error("OPERATOR should not be able to configure system")
	}
}

func TestMaintainerHasFullAccess(t *testing.T) {
	allActions := []Action{
		ActionViewEvents, ActionAckEvents, ActionResolveEvents,
		ActionViewCameras, ActionViewZones, ActionViewEquipment,
		ActionViewStatus, ActionViewHardware, ActionWorkWindows,
		ActionCloseWorkWindows, ActionDiagnostics, ActionIOTest,
		ActionManageUsers, ActionConfigureSystem, ActionViewAudit,
		ActionRealtime,
	}

	for _, action := range allActions {
		if !HasPermission(RoleMaintainer, action) {
			t.Errorf("MAINTAINER should have permission for %s", action)
		}
	}
}

func TestRoleHierarchy(t *testing.T) {
	// User has least permissions
	userPerms := 0
	operatorPerms := 0
	maintainerPerms := 0

	allActions := []Action{
		ActionViewEvents, ActionAckEvents, ActionResolveEvents,
		ActionViewCameras, ActionViewZones, ActionViewEquipment,
		ActionViewStatus, ActionViewHardware, ActionWorkWindows,
		ActionCloseWorkWindows, ActionDiagnostics, ActionIOTest,
		ActionManageUsers, ActionConfigureSystem, ActionViewAudit,
		ActionRealtime,
	}

	for _, action := range allActions {
		if HasPermission(RoleUser, action) {
			userPerms++
		}
		if HasPermission(RoleOperator, action) {
			operatorPerms++
		}
		if HasPermission(RoleMaintainer, action) {
			maintainerPerms++
		}
	}

	if userPerms >= operatorPerms {
		t.Errorf("USER (%d) should have fewer permissions than OPERATOR (%d)", userPerms, operatorPerms)
	}
	if operatorPerms >= maintainerPerms {
		t.Errorf("OPERATOR (%d) should have fewer permissions than MAINTAINER (%d)", operatorPerms, maintainerPerms)
	}
}

func TestInvalidRole(t *testing.T) {
	if HasPermission(Role("ADMIN"), ActionViewEvents) {
		t.Error("invalid role should have no permissions")
	}
	if HasPermission(Role(""), ActionViewEvents) {
		t.Error("empty role should have no permissions")
	}
}

func TestValidRole(t *testing.T) {
	if !ValidRole("USER") {
		t.Error("USER should be valid")
	}
	if !ValidRole("OPERATOR") {
		t.Error("OPERATOR should be valid")
	}
	if !ValidRole("MAINTAINER") {
		t.Error("MAINTAINER should be valid")
	}
	if ValidRole("ADMIN") {
		t.Error("ADMIN should not be valid")
	}
	if ValidRole("") {
		t.Error("empty string should not be valid")
	}
}

func TestSessionStore_CreateAndGet(t *testing.T) {
	store := NewSessionStore([]byte("test-secret"))

	session := store.CreateSession("operator1", RoleOperator)
	if session == nil {
		t.Fatal("expected session, got nil")
	}
	if session.Username != "operator1" {
		t.Errorf("expected username=operator1, got %s", session.Username)
	}
	if session.Role != RoleOperator {
		t.Errorf("expected role=OPERATOR, got %s", session.Role)
	}
	if session.SessionID == "" {
		t.Error("expected non-empty sessionID")
	}

	// Retrieve session
	got := store.GetSession(session.SessionID)
	if got == nil {
		t.Fatal("expected to retrieve session")
	}
	if got.Username != "operator1" {
		t.Errorf("expected username=operator1, got %s", got.Username)
	}
}

func TestSessionStore_InvalidSession(t *testing.T) {
	store := NewSessionStore([]byte("test-secret"))

	got := store.GetSession("nonexistent")
	if got != nil {
		t.Error("expected nil for nonexistent session")
	}
}

func TestSessionStore_DeleteSession(t *testing.T) {
	store := NewSessionStore([]byte("test-secret"))

	session := store.CreateSession("user1", RoleUser)
	store.DeleteSession(session.SessionID)

	got := store.GetSession(session.SessionID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	store := NewSessionStore([]byte("test-secret"))
	middleware := AuthMiddleware(store)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	store := NewSessionStore([]byte("test-secret"))
	middleware := AuthMiddleware(store)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	store := NewSessionStore([]byte("test-secret"))
	session := store.CreateSession("admin1", RoleMaintainer)
	middleware := AuthMiddleware(store)

	called := false
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		s := GetSessionFromContext(r.Context())
		if s == nil {
			t.Error("expected session in context")
		}
		if s.Username != "admin1" {
			t.Errorf("expected username=admin1, got %s", s.Username)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+session.SessionID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should have been called")
	}
}

func TestRequirePermission_Forbidden(t *testing.T) {
	permMiddleware := RequirePermission(ActionIOTest)

	handler := permMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	// Create a request with USER session (cannot do IO tests)
	session := &Session{Username: "user1", Role: RoleUser}
	ctx := WithSession(context.Background(), session)
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	permMiddleware := RequirePermission(ActionIOTest)

	called := false
	handler := permMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	// Maintainer can do IO tests
	session := &Session{Username: "maintainer1", Role: RoleMaintainer}
	ctx := WithSession(context.Background(), session)
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should have been called for MAINTAINER")
	}
}
