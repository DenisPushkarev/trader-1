package domain

import "time"

// Scenario defines a replayable simulation scenario.
type Scenario struct {
	ID          string
	Name        string
	Description string
	Events      []SimulatedEvent
}

// SimulatedEvent is a synthetic normalized event for simulation.
type SimulatedEvent struct {
	EventID   string
	Source    string
	EventType string
	Sentiment float64
	Impact    float64
	Content   string
	Timestamp time.Time
}

// SimulationResult holds the outcome of running a scenario.
type SimulationResult struct {
	ScenarioID       string
	ScenarioName     string
	EventsProcessed  int
	SignalsGenerated  int
	BullishCount     int
	BearishCount     int
	NeutralCount     int
	AvgConfidence    float64
	DurationMs       int64
}
