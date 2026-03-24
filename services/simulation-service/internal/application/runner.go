package application

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/simulation-service/internal/domain"
)

// ScenarioRegistry provides access to simulation scenarios.
type ScenarioRegistry interface {
	Get(id string) (*domain.Scenario, bool)
	List() []*domain.Scenario
}

// SimulationRunner executes simulation scenarios in isolation.
type SimulationRunner struct {
	logger   zerolog.Logger
	registry ScenarioRegistry
}

// NewSimulationRunner creates a SimulationRunner.
func NewSimulationRunner(logger zerolog.Logger, registry ScenarioRegistry) *SimulationRunner {
	return &SimulationRunner{logger: logger, registry: registry}
}

// Run executes a scenario and returns the result.
// It is completely isolated from live production streams.
func (r *SimulationRunner) Run(_ context.Context, scenarioID string) (*domain.SimulationResult, error) {
	scenario, ok := r.registry.Get(scenarioID)
	if !ok {
		return nil, &ScenarioNotFoundError{ID: scenarioID}
	}

	start := time.Now()
	r.logger.Info().Str("scenario", scenarioID).Msg("simulation started")

	result := &domain.SimulationResult{
		ScenarioID:      scenario.ID,
		ScenarioName:    scenario.Name,
		EventsProcessed: len(scenario.Events),
	}

	// Replay events through inline scoring (isolated from live NATS)
	var totalConfidence float64
	windowSize := 3
	for i := 0; i+windowSize <= len(scenario.Events); i += windowSize {
		window := scenario.Events[i : i+windowSize]
		dir, conf := scoreWindow(window)
		totalConfidence += conf
		result.SignalsGenerated++
		switch dir {
		case 1:
			result.BullishCount++
		case 2:
			result.BearishCount++
		default:
			result.NeutralCount++
		}
	}

	if result.SignalsGenerated > 0 {
		result.AvgConfidence = totalConfidence / float64(result.SignalsGenerated)
	}
	result.DurationMs = time.Since(start).Milliseconds()

	r.logger.Info().
		Str("scenario", scenarioID).
		Int("signals", result.SignalsGenerated).
		Float64("avg_confidence", result.AvgConfidence).
		Msg("simulation complete")

	return result, nil
}

// scoreWindow applies a simple scoring to a window of events.
// This mirrors the signal engine logic but runs in isolation.
func scoreWindow(events []domain.SimulatedEvent) (direction int, confidence float64) {
	var totalSentiment float64
	for _, ev := range events {
		totalSentiment += ev.Sentiment
	}
	avg := totalSentiment / float64(len(events))
	conf := avg
	if conf < 0 {
		conf = -conf
	}
	if conf > 1 {
		conf = 1
	}
	switch {
	case avg > 0.1:
		return 1, conf // bullish
	case avg < -0.1:
		return 2, conf // bearish
	default:
		return 3, conf // neutral
	}
}

// ScenarioNotFoundError is returned when a scenario ID is not found.
type ScenarioNotFoundError struct {
	ID string
}

func (e *ScenarioNotFoundError) Error() string {
	return "scenario not found: " + e.ID
}
