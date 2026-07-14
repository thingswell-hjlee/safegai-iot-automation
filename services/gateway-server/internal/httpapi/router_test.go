package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/auth"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/domain/events"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/observability"
	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/storage/memory"
)

func setupTestRouter() (*Router, *auth.SessionStore, *memory.Store) {
	store := memory.NewStore()
	sessionStore := auth.NewSessionStore([]byte("test-secret"))
	healthCollector := observability.NewHealthCollector()
	handlers := NewHandlers(store, sessionStore, healthCollector)
	router := NewRouter(handlers, sessionStore)
	return router, sessionStore, store
}

func authenticatedRequest(method, path string, body []byte, token string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestRouter_LoginEndpoint(t *testing.T) {
	router, _, store := setupTestRouter()

	// Create a user first
	store.CreateUser(nil, &events.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: hashPassword("password123"),
		Role:         "OPERATOR",
		CreatedAt:    time.Now().UTC(),
	})

	body := []byte(`{"username":"testuser","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected token in response")
	}
	if resp["role"] != "OPERATOR" {
		t.Errorf("expected role=OPERATOR, got %v", resp["role"])
	}
}

func TestRouter_LoginInvalidCredentials(t *testing.T) {
	router, _, store := setupTestRouter()

	store.CreateUser(nil, &events.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: hashPassword("password123"),
		Role:         "OPERATOR",
		CreatedAt:    time.Now().UTC(),
	})

	body := []byte(`{"username":"testuser","password":"wrongpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRouter_ProtectedEndpointNoAuth(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRouter_SystemStatus(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	session := sessionStore.CreateSession("admin", auth.RoleMaintainer)

	req := authenticatedRequest(http.MethodGet, "/api/v1/system/status", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "running" {
		t.Errorf("expected status=running, got %v", resp["status"])
	}
}

func TestRouter_SystemHardware(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	session := sessionStore.CreateSession("admin", auth.RoleMaintainer)

	req := authenticatedRequest(http.MethodGet, "/api/v1/system/hardware", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRouter_UserCannotAccessHardware(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	session := sessionStore.CreateSession("user1", auth.RoleUser)

	req := authenticatedRequest(http.MethodGet, "/api/v1/system/hardware", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for USER accessing hardware, got %d", w.Code)
	}
}

func TestRouter_ListEvents(t *testing.T) {
	router, sessionStore, store := setupTestRouter()

	// Insert test events
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		evt := &events.SafetyEvent{
			EventEnvelope: events.EventEnvelope{
				SchemaVersion: "1.0.0",
				EventID:       "evt-" + itoa(i),
				CorrelationID: "corr-" + itoa(i),
				TenantID:      "tenant-1",
				SiteID:        "site-1",
				GatewayID:     "gw-1",
				DeviceID:      "cam-1",
				ZoneID:        "zone-1",
				ObservedAt:    now,
				ReceivedAt:    now,
				SequenceNo:    int64(i),
				Source:        "test",
				Quality:       events.QualityGood,
			},
			Severity:       events.SeverityInfo,
			OccupancyState: events.OccupancyOccupied,
			EquipmentState: events.EquipmentRunning,
			DetectedAt:     now,
		}
		store.InsertEvent(nil, evt)
	}

	session := sessionStore.CreateSession("user1", auth.RoleUser)
	req := authenticatedRequest(http.MethodGet, "/api/v1/events?offset=0&limit=10", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	count := resp["count"].(float64)
	if count != 3 {
		t.Errorf("expected count=3, got %v", count)
	}
}

func TestRouter_AckEvent_OperatorAllowed(t *testing.T) {
	router, sessionStore, store := setupTestRouter()

	now := time.Now().UTC()
	evt := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-ack-test",
			CorrelationID: "corr-1",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-1",
			ZoneID:        "zone-1",
			ObservedAt:    now,
			ReceivedAt:    now,
			SequenceNo:    1,
			Source:        "test",
			Quality:       events.QualityGood,
		},
		Severity:       events.SeverityWarning,
		OccupancyState: events.OccupancyOccupied,
		EquipmentState: events.EquipmentRunning,
		DetectedAt:     now,
	}
	store.InsertEvent(nil, evt)

	session := sessionStore.CreateSession("operator1", auth.RoleOperator)
	req := authenticatedRequest(http.MethodPost, "/api/v1/events/evt-ack-test/ack", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRouter_AckEvent_UserForbidden(t *testing.T) {
	router, sessionStore, store := setupTestRouter()

	now := time.Now().UTC()
	evt := &events.SafetyEvent{
		EventEnvelope: events.EventEnvelope{
			SchemaVersion: "1.0.0",
			EventID:       "evt-ack-forbidden",
			CorrelationID: "corr-1",
			TenantID:      "tenant-1",
			SiteID:        "site-1",
			GatewayID:     "gw-1",
			DeviceID:      "cam-1",
			ZoneID:        "zone-1",
			ObservedAt:    now,
			ReceivedAt:    now,
			SequenceNo:    1,
			Source:        "test",
			Quality:       events.QualityGood,
		},
		Severity:       events.SeverityWarning,
		OccupancyState: events.OccupancyOccupied,
		EquipmentState: events.EquipmentRunning,
		DetectedAt:     now,
	}
	store.InsertEvent(nil, evt)

	session := sessionStore.CreateSession("user1", auth.RoleUser)
	req := authenticatedRequest(http.MethodPost, "/api/v1/events/evt-ack-forbidden/ack", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for USER acking event, got %d", w.Code)
	}
}

func TestRouter_DiagnosticsRequiresMaintainer(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	// Operator cannot access diagnostics
	session := sessionStore.CreateSession("operator1", auth.RoleOperator)
	req := authenticatedRequest(http.MethodGet, "/api/v1/maintenance/diagnostics", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for OPERATOR accessing diagnostics, got %d", w.Code)
	}

	// Maintainer can access
	session2 := sessionStore.CreateSession("admin", auth.RoleMaintainer)
	req2 := authenticatedRequest(http.MethodGet, "/api/v1/maintenance/diagnostics", nil, session2.SessionID)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for MAINTAINER accessing diagnostics, got %d", w2.Code)
	}
}

func TestRouter_IOTestRequiresMaintainer(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	// Operator cannot do I/O test
	session := sessionStore.CreateSession("operator1", auth.RoleOperator)
	req := authenticatedRequest(http.MethodPost, "/api/v1/maintenance/io-test", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for OPERATOR doing IO test, got %d", w.Code)
	}
}

func TestRouter_CamerasEndpoint(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	session := sessionStore.CreateSession("user1", auth.RoleUser)
	req := authenticatedRequest(http.MethodGet, "/api/v1/cameras", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRouter_ZonesEndpoint(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	session := sessionStore.CreateSession("user1", auth.RoleUser)
	req := authenticatedRequest(http.MethodGet, "/api/v1/zones", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRouter_EquipmentEndpoint(t *testing.T) {
	router, sessionStore, _ := setupTestRouter()

	session := sessionStore.CreateSession("user1", auth.RoleUser)
	req := authenticatedRequest(http.MethodGet, "/api/v1/equipment", nil, session.SessionID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// itoa converts int to string for test helpers.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
