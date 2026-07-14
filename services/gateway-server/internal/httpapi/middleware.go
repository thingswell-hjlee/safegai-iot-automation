// Package httpapi provides the local REST API for the SafeGAI gateway.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/thingswell-hjlee/safegai-iot-automation/services/gateway-server/internal/auth"
)

// loggingMiddleware logs each request (placeholder for structured logging).
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers for local development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// chain applies a sequence of middleware to a handler.
func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// requireAuth wraps a handler with auth middleware.
func requireAuth(store *auth.SessionStore, handler http.Handler) http.Handler {
	return auth.AuthMiddleware(store)(handler)
}

// requirePerm wraps a handler with permission middleware.
func requirePerm(action auth.Action, handler http.Handler) http.Handler {
	return auth.RequirePermission(action)(handler)
}
