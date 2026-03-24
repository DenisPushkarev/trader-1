package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/simulation-service/internal/application"
	"github.com/trader-1/trader-1/services/simulation-service/internal/domain"
)

type mockRegistry struct {
	scenarios map[string]*domain.Scenario
}

func (m *mockRegistry) Get(id string) (*domain.Scenario, bool) {
	s, ok := m.scenarios[id]
	return s, ok
}
func (m *mockRegistry) List() []*domain.Scenario {
	result := make([]*domain.Scenario, 0)
	for _, s := range m.scenarios {
		result = append(result, s)
	}
	return result
}

func TestSimulationRunner_KnownScenario(t *testing.T) {
	scenario := &domain.Scenario{
		ID:   "test_scenario",
		Name: "Test",
		Events: []domain.SimulatedEvent{
			{EventID: "e1", Sentiment: 0.8, Impact: 0.7, Timestamp: time.Now()},
			{EventID: "e2", Sentiment: 0.6, Impact: 0.5, Timestamp: time.Now()},
			{EventID: "e3", Sentiment: 0.7, Impact: 0.6, Timestamp: time.Now()},
		},
	}
	reg := &mockRegistry{scenarios: map[string]*domain.Scenario{"test_scenario": scenario}}
	runner := application.NewSimulationRunner(zerolog.Nop(), reg)

	result, err := runner.Run(context.Background(), "test_scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EventsProcessed != 3 {
		t.Errorf("expected 3 events processed, got %d", result.EventsProcessed)
	}
	if result.SignalsGenerated != 1 {
		t.Errorf("expected 1 signal generated, got %d", result.SignalsGenerated)
	}
	if result.BullishCount != 1 {
		t.Errorf("expected 1 bullish signal, got %d", result.BullishCount)
	}
}

func TestSimulationRunner_UnknownScenario(t *testing.T) {
	reg := &mockRegistry{scenarios: map[string]*domain.Scenario{}}
	runner := application.NewSimulationRunner(zerolog.Nop(), reg)
	_, err := runner.Run(context.Background(), "unknown")
	if err == nil {
		t.Error("expected error for unknown scenario")
	}
}
