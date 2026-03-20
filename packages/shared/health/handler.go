package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Handler provides HTTP liveness and readiness endpoints.
type Handler struct {
	ready atomic.Bool
}

// NewHandler creates a new health Handler. Readiness starts as false.
func NewHandler() *Handler {
	return &Handler{}
}

// SetReady marks the service as ready (or not ready) to accept traffic.
func (h *Handler) SetReady(ready bool) {
	h.ready.Store(ready)
}

// LivenessHandler returns 200 OK as long as the process is running.
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"}) //nolint:errcheck
	}
}

// ReadinessHandler returns 200 if ready, 503 otherwise.
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if h.ready.Load() {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"}) //nolint:errcheck
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"}) //nolint:errcheck
		}
	}
}
