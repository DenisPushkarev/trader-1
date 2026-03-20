package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/domain"
)

// QueryService provides query operations.
type QueryService interface {
	GetLatestSignals(ctx context.Context, limit int) ([]*domain.SignalReadModel, error)
	GetSignalHistory(ctx context.Context, page, pageSize int) ([]*domain.SignalReadModel, error)
	GetEvents(ctx context.Context, limit int) ([]*domain.EventReadModel, error)
}

// APIHandler handles HTTP requests.
type APIHandler struct {
	query  QueryService
	logger zerolog.Logger
}

// NewAPIHandler creates an APIHandler.
func NewAPIHandler(query QueryService, logger zerolog.Logger) *APIHandler {
	return &APIHandler{query: query, logger: logger}
}

// GetLatestSignals handles GET /signals/latest
func (h *APIHandler) GetLatestSignals(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 10)
	signals, err := h.query.GetLatestSignals(r.Context(), limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"signals": signals, "count": len(signals)})
}

// GetSignalHistory handles GET /signals/history
func (h *APIHandler) GetSignalHistory(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 10)
	signals, err := h.query.GetSignalHistory(r.Context(), page, pageSize)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"signals": signals, "page": page, "page_size": pageSize})
}

// GetEvents handles GET /events
func (h *APIHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	events, err := h.query.GetEvents(r.Context(), limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"events": events, "count": len(events)})
}

// PostSimulate handles POST /simulate
func (h *APIHandler) PostSimulate(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// In production this would delegate to simulation-service via NATS request/reply.
	// Stubbed response for initial delivery.
	h.writeJSON(w, http.StatusAccepted, map[string]any{
		"status":  "accepted",
		"message": "simulation request queued",
		"request": req,
	})
}

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Warn().Err(err).Msg("encode response")
	}
}

func (h *APIHandler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, def int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}
