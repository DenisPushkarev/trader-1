package scoring_test

import (
	"testing"

	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/domain"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/infrastructure/scoring"
)

func TestDefaultModel_Bullish(t *testing.T) {
	m := scoring.NewDefaultModel()
	events := []*contractsv1.NormalizedEvent{
		{EventId: "e1", Sentiment: 0.8, Impact: 0.7},
		{EventId: "e2", Sentiment: 0.6, Impact: 0.5},
	}
	dir, conf := m.Score(events, nil)
	if dir != domain.DirectionBullish {
		t.Errorf("expected BULLISH, got %d", dir)
	}
	if conf <= 0 || conf > 1 {
		t.Errorf("confidence out of range: %f", conf)
	}
}

func TestDefaultModel_Bearish(t *testing.T) {
	m := scoring.NewDefaultModel()
	events := []*contractsv1.NormalizedEvent{
		{EventId: "e1", Sentiment: -0.8, Impact: 0.5},
	}
	dir, conf := m.Score(events, nil)
	if dir != domain.DirectionBearish {
		t.Errorf("expected BEARISH, got %d", dir)
	}
	if conf <= 0 {
		t.Errorf("confidence should be positive: %f", conf)
	}
}

func TestDefaultModel_Deterministic(t *testing.T) {
	m := scoring.NewDefaultModel()
	events := []*contractsv1.NormalizedEvent{
		{EventId: "e1", Sentiment: 0.5, Impact: 0.6},
		{EventId: "e2", Sentiment: 0.3, Impact: 0.4},
	}
	ctx := &contractsv1.MarketContextSnapshot{ContextId: "ctx-1", Volatility: 0.02}
	d1, c1 := m.Score(events, ctx)
	d2, c2 := m.Score(events, ctx)
	if d1 != d2 || c1 != c2 {
		t.Errorf("scoring is not deterministic: (%d,%f) vs (%d,%f)", d1, c1, d2, c2)
	}
}

func TestDefaultModel_EmptyEvents(t *testing.T) {
	m := scoring.NewDefaultModel()
	dir, conf := m.Score(nil, nil)
	if dir != domain.DirectionNeutral {
		t.Errorf("expected NEUTRAL for empty events, got %d", dir)
	}
	if conf != 0 {
		t.Errorf("expected 0 confidence for empty events, got %f", conf)
	}
}

func TestDefaultModel_VolatilityPenalty(t *testing.T) {
	m := scoring.NewDefaultModel()
	events := []*contractsv1.NormalizedEvent{
		{EventId: "e1", Sentiment: 0.8, Impact: 0.8},
	}
	lowVol := &contractsv1.MarketContextSnapshot{Volatility: 0.01}
	highVol := &contractsv1.MarketContextSnapshot{Volatility: 0.20}

	_, confLow := m.Score(events, lowVol)
	_, confHigh := m.Score(events, highVol)

	if confHigh >= confLow {
		t.Errorf("high volatility should reduce confidence: low=%f high=%f", confLow, confHigh)
	}
}
