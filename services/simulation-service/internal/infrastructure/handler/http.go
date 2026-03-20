package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/simulation-service/internal/domain"
)

// Runner executes simulations.
type Runner interface {
	Run(ctx context.Context, scenarioID string) (*domain.SimulationResult, error)
	// Need ListScenarios too
}

// ScenarioLister lists available scenarios.
type ScenarioLister interface {
	List() []*domain.Scenario
}

// SimulationRunner is the combined interface.
type SimulationRunner interface {
	Run(ctx context.Context, scenarioID string) (*domain.SimulationResult, error)
}

// SimulationHandler handles HTTP simulation requests.
type SimulationHandler struct {
	runner SimulationRunner
	logger zerolog.Logger
}

// NewSimulationHandler creates a SimulationHandler.
func NewSimulationHandler(runner SimulationRunner, logger zerolog.Logger) *SimulationHandler {
	return &SimulationHandler{runner: runner, logger: logger}
}

// RunSimulation handles POST /run
func (h *SimulationHandler) RunSimulation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string `json:"scenario_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.ScenarioID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scenario_id required"})
		return
	}
	result, err := h.runner.Run(r.Context(), req.ScenarioID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ListScenarios handles GET /scenarios (uses in-memory registry via runner)
func (h *SimulationHandler) ListScenarios(w http.ResponseWriter, r *http.Request) {
	// In a full implementation this would query the registry.
	// For now, return a static list of available scenario IDs.
	writeJSON(w, http.StatusOK, map[string]any{
		"scenarios": []map[string]string{
			{"id": "bullish_run", "name": "Bullish Run"},
			{"id": "bearish_run", "name": "Bearish Run"},
			{"id": "fake_hype", "name": "Fake Hype"},
			{"id": "conflicting_signals", "name": "Conflicting Signals"},
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
