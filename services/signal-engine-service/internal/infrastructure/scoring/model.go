package scoring

import (
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/domain"
)

const defaultConfigVersion = "v1"

// DefaultModel is a config-driven, deterministic scoring model.
type DefaultModel struct {
	sentimentWeight   float64
	impactWeight      float64
	volatilityPenalty float64
	minConfidence     float64
	version           string
}

// NewDefaultModel creates a DefaultModel with default weights.
func NewDefaultModel() *DefaultModel {
	return &DefaultModel{
		sentimentWeight:   0.6,
		impactWeight:      0.4,
		volatilityPenalty: 0.3,
		minConfidence:     0.1,
		version:           defaultConfigVersion,
	}
}

// ConfigVersion returns the model version string.
func (m *DefaultModel) ConfigVersion() string {
	return m.version
}

// Score produces a direction and confidence from events and market context.
// Deterministic for identical inputs.
func (m *DefaultModel) Score(events []*contractsv1.NormalizedEvent, ctx *contractsv1.MarketContextSnapshot) (domain.Direction, float64) {
	if len(events) == 0 {
		return domain.DirectionNeutral, 0
	}

	var totalSentiment, totalImpact float64
	for _, ev := range events {
		totalSentiment += ev.Sentiment
		totalImpact += ev.Impact
	}
	avgSentiment := totalSentiment / float64(len(events))
	avgImpact := totalImpact / float64(len(events))

	rawScore := avgSentiment*m.sentimentWeight + avgImpact*m.impactWeight

	// Apply volatility penalty from market context
	if ctx != nil && ctx.Volatility > 0.05 {
		rawScore *= (1 - m.volatilityPenalty*ctx.Volatility)
	}

	// Clamp confidence to [0, 1]
	confidence := rawScore
	if confidence < 0 {
		confidence = -confidence
	}
	if confidence > 1 {
		confidence = 1
	}
	if confidence < m.minConfidence {
		confidence = m.minConfidence
	}

	var direction domain.Direction
	switch {
	case rawScore > 0.1:
		direction = domain.DirectionBullish
	case rawScore < -0.1:
		direction = domain.DirectionBearish
	default:
		direction = domain.DirectionNeutral
	}

	return direction, confidence
}
