package domain

import "time"

// SignalReadModel is a denormalized signal for API queries.
type SignalReadModel struct {
	SignalID       string    `json:"signal_id"`
	Direction      string    `json:"direction"`
	Confidence     float64   `json:"confidence"`
	RiskLevel      string    `json:"risk_level"`
	Summary        string    `json:"summary"`
	Recommendation string    `json:"recommendation"`
	Timestamp      time.Time `json:"timestamp"`
}

// EventReadModel is a denormalized event for API queries.
type EventReadModel struct {
	EventID   string    `json:"event_id"`
	Source    string    `json:"source"`
	EventType string    `json:"event_type"`
	Asset     string    `json:"asset"`
	Sentiment float64   `json:"sentiment"`
	Impact    float64   `json:"impact"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
