package httpapi

import (
	"net/http"
	"strings"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/auth"
)

// Router manages HTTP route registration and dispatch.
type Router struct {
	mux      *http.ServeMux
	handlers *Handlers
	sessions *auth.SessionStore
}

// NewRouter creates a new Router with registered routes.
func NewRouter(handlers *Handlers, sessions *auth.SessionStore) *Router {
	r := &Router{
		mux:      http.NewServeMux(),
		handlers: handlers,
		sessions: sessions,
	}
	r.registerRoutes()
	return r
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler := chain(r.mux, corsMiddleware, loggingMiddleware)
	handler.ServeHTTP(w, req)
}

// Handler returns the router as an http.Handler.
func (r *Router) Handler() http.Handler {
	return r
}

// registerRoutes sets up all API route handlers.
func (r *Router) registerRoutes() {
	// Public endpoint (no auth required)
	r.mux.HandleFunc("/api/v1/session/login", r.handlers.HandleLogin)

	// Protected endpoints with auth + permission checks
	r.mux.Handle("/api/v1/system/status",
		requireAuth(r.sessions,
			requirePerm(auth.ActionViewStatus,
				http.HandlerFunc(r.handlers.HandleSystemStatus))))

	r.mux.Handle("/api/v1/system/hardware",
		requireAuth(r.sessions,
			requirePerm(auth.ActionViewHardware,
				http.HandlerFunc(r.handlers.HandleSystemHardware))))

	r.mux.Handle("/api/v1/cameras",
		requireAuth(r.sessions,
			requirePerm(auth.ActionViewCameras,
				http.HandlerFunc(r.handlers.HandleCameras))))

	r.mux.Handle("/api/v1/zones",
		requireAuth(r.sessions,
			requirePerm(auth.ActionViewZones,
				http.HandlerFunc(r.handlers.HandleZones))))

	r.mux.Handle("/api/v1/equipment",
		requireAuth(r.sessions,
			requirePerm(auth.ActionViewEquipment,
				http.HandlerFunc(r.handlers.HandleEquipment))))

	r.mux.Handle("/api/v1/events",
		requireAuth(r.sessions,
			requirePerm(auth.ActionViewEvents,
				http.HandlerFunc(r.handlers.HandleListEvents))))

	// Events ack/resolve use pattern matching via a single handler
	r.mux.Handle("/api/v1/events/",
		requireAuth(r.sessions,
			http.HandlerFunc(r.handleEventActions)))

	r.mux.Handle("/api/v1/work-windows",
		requireAuth(r.sessions,
			requirePerm(auth.ActionWorkWindows,
				http.HandlerFunc(r.handlers.HandleCreateWorkWindow))))

	// Work windows close endpoint
	r.mux.Handle("/api/v1/work-windows/",
		requireAuth(r.sessions,
			requirePerm(auth.ActionCloseWorkWindows,
				http.HandlerFunc(r.handlers.HandleCloseWorkWindow))))

	r.mux.Handle("/api/v1/maintenance/diagnostics",
		requireAuth(r.sessions,
			requirePerm(auth.ActionDiagnostics,
				http.HandlerFunc(r.handlers.HandleDiagnostics))))

	r.mux.Handle("/api/v1/maintenance/io-test",
		requireAuth(r.sessions,
			requirePerm(auth.ActionIOTest,
				http.HandlerFunc(r.handlers.HandleIOTest))))

	r.mux.Handle("/api/v1/realtime",
		requireAuth(r.sessions,
			requirePerm(auth.ActionRealtime,
				http.HandlerFunc(r.handlers.HandleRealtime))))
}

// handleEventActions routes /api/v1/events/{id}/ack and /api/v1/events/{id}/resolve
func (r *Router) handleEventActions(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	switch {
	case strings.HasSuffix(path, "/ack"):
		session := auth.GetSessionFromContext(req.Context())
		if session == nil || !auth.HasPermission(session.Role, auth.ActionAckEvents) {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		r.handlers.HandleAckEvent(w, req)
	case strings.HasSuffix(path, "/resolve"):
		session := auth.GetSessionFromContext(req.Context())
		if session == nil || !auth.HasPermission(session.Role, auth.ActionResolveEvents) {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		r.handlers.HandleResolveEvent(w, req)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
