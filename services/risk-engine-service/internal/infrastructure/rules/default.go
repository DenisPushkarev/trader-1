package rules

import (
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/services/risk-engine-service/internal/domain"
)

const defaultRiskConfigVersion = "v1"

// DefaultRuleSet applies standard risk rules to generated signals.
type DefaultRuleSet struct {
	minConfidenceThreshold float64
	highRiskThreshold      float64
	criticalRiskThreshold  float64
}

// NewDefaultRuleSet creates a DefaultRuleSet with standard parameters.
func NewDefaultRuleSet() *DefaultRuleSet {
	return &DefaultRuleSet{
		minConfidenceThreshold: 0.2,
		highRiskThreshold:      0.4,
		criticalRiskThreshold:  0.6,
	}
}

// ConfigVersion returns the rule set version.
func (r *DefaultRuleSet) ConfigVersion() string {
	return defaultRiskConfigVersion
}

// Apply evaluates a generated signal and returns a risk-adjusted result.
func (r *DefaultRuleSet) Apply(sig *contractsv1.GeneratedSignal) *domain.RiskAdjustedSignal {
	result := &domain.RiskAdjustedSignal{
		AdjustedConfidence: sig.Confidence,
		Adjustments:        []domain.Adjustment{},
	}

	// Rule 1: Block signals with insufficient source confirmation (< 2 contributing events)
	if len(sig.ContributingEvents) < 2 {
		result.Blocked = true
		result.BlockReason = "insufficient_source_confirmation"
		result.RiskLevel = domain.RiskLevelHigh
		return result
	}

	// Rule 2: Penalize low-confidence signals
	if sig.Confidence < r.minConfidenceThreshold {
		result.Blocked = true
		result.BlockReason = "confidence_below_threshold"
		result.RiskLevel = domain.RiskLevelHigh
		return result
	}

	// Rule 3: Penalize NEUTRAL signals
	if sig.Direction == contractsv1.Direction_DIRECTION_NEUTRAL {
		delta := -0.1
		result.AdjustedConfidence += delta
		result.Adjustments = append(result.Adjustments, domain.Adjustment{
			Reason: "neutral_direction_penalty",
			Delta:  delta,
		})
	}

	// Rule 4: Assign risk level based on adjusted confidence
	switch {
	case result.AdjustedConfidence >= r.criticalRiskThreshold:
		result.RiskLevel = domain.RiskLevelLow
	case result.AdjustedConfidence >= r.highRiskThreshold:
		result.RiskLevel = domain.RiskLevelMedium
	default:
		result.RiskLevel = domain.RiskLevelHigh
	}

	// Clamp confidence
	if result.AdjustedConfidence < 0 {
		result.AdjustedConfidence = 0
	}
	if result.AdjustedConfidence > 1 {
		result.AdjustedConfidence = 1
	}

	return result
}
