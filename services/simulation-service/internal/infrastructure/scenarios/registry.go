package scenarios

import (
	"fmt"
	"sync"
	"time"

	"github.com/trader-1/trader-1/services/simulation-service/internal/domain"
)

// Registry holds all available simulation scenarios.
type Registry struct {
	mu        sync.RWMutex
	scenarios map[string]*domain.Scenario
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{scenarios: make(map[string]*domain.Scenario)}
}

// Register adds a scenario to the registry.
func (r *Registry) Register(s *domain.Scenario) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scenarios[s.ID] = s
}

// Get retrieves a scenario by ID.
func (r *Registry) Get(id string) (*domain.Scenario, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scenarios[id]
	return s, ok
}

// List returns all registered scenarios.
func (r *Registry) List() []*domain.Scenario {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domain.Scenario, 0, len(r.scenarios))
	for _, s := range r.scenarios {
		result = append(result, s)
	}
	return result
}

// BullishRunScenario returns a scenario with predominantly positive events.
func BullishRunScenario() *domain.Scenario {
	return &domain.Scenario{
		ID:          "bullish_run",
		Name:        "Bullish Run",
		Description: "Strong positive sentiment from multiple sources",
		Events:      generateEvents(12, 0.7, 0.6),
	}
}

// BearishRunScenario returns a scenario with predominantly negative events.
func BearishRunScenario() *domain.Scenario {
	return &domain.Scenario{
		ID:          "bearish_run",
		Name:        "Bearish Run",
		Description: "Negative sentiment from multiple sources",
		Events:      generateEvents(12, -0.7, 0.5),
	}
}

// FakeHypeScenario returns a high-sentiment, low-impact scenario (pump and dump risk).
func FakeHypeScenario() *domain.Scenario {
	return &domain.Scenario{
		ID:          "fake_hype",
		Name:        "Fake Hype",
		Description: "High sentiment but low credibility and impact",
		Events:      generateEvents(9, 0.8, 0.2),
	}
}

// ConflictingSignalsScenario returns a mixed signals scenario.
func ConflictingSignalsScenario() *domain.Scenario {
	events := make([]domain.SimulatedEvent, 12)
	base := time.Now()
	for i := range events {
		sentiment := 0.6
		if i%2 == 0 {
			sentiment = -0.6
		}
		events[i] = domain.SimulatedEvent{
			EventID:   fmt.Sprintf("conflict-evt-%d", i),
			Source:    "mixed",
			EventType: "social_post",
			Sentiment: sentiment,
			Impact:    0.4,
			Content:   "conflicting signal",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
		}
	}
	return &domain.Scenario{
		ID:          "conflicting_signals",
		Name:        "Conflicting Signals",
		Description: "Mixed bullish and bearish signals",
		Events:      events,
	}
}

func generateEvents(n int, sentiment, impact float64) []domain.SimulatedEvent {
	events := make([]domain.SimulatedEvent, n)
	base := time.Now()
	for i := range events {
		events[i] = domain.SimulatedEvent{
			EventID:   fmt.Sprintf("sim-evt-%d", i),
			Source:    "telegram",
			EventType: "social_post",
			Sentiment: sentiment,
			Impact:    impact,
			Content:   "simulated event",
			Timestamp: base.Add(time.Duration(i) * time.Minute),
		}
	}
	return events
}
