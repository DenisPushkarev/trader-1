package generator

import (
	"fmt"

	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/services/explainability-service/internal/domain"
)

const templateConfigVersion = "v1"

// TemplateGenerator produces factor-based natural language explanations.
type TemplateGenerator struct{}

// NewTemplateGenerator creates a TemplateGenerator.
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{}
}

// ConfigVersion returns the generator version.
func (g *TemplateGenerator) ConfigVersion() string {
	return templateConfigVersion
}

// Generate produces an ExplainedSignal from a risk-adjusted signal.
// The explanation references actual factors from the input signal.
func (g *TemplateGenerator) Generate(sig *contractsv1.RiskAdjustedSignal) *domain.ExplainedSignal {
	factors := g.buildFactors(sig)
	summary := g.buildSummary(sig)
	recommendation := g.buildRecommendation(sig)

	return &domain.ExplainedSignal{
		SignalID:       sig.OriginalSignalId,
		Summary:        summary,
		Factors:        factors,
		Recommendation: recommendation,
	}
}

func (g *TemplateGenerator) buildFactors(sig *contractsv1.RiskAdjustedSignal) []domain.Factor {
	factors := []domain.Factor{
		{
			Name:        "adjusted_confidence",
			Description: fmt.Sprintf("Signal confidence after risk adjustment: %.2f%%", sig.AdjustedConfidence*100),
			Weight:      sig.AdjustedConfidence,
		},
		{
			Name:        "risk_level",
			Description: fmt.Sprintf("Risk classification: %s", sig.RiskLevel.String()),
			Weight:      riskWeight(sig.RiskLevel),
		},
	}

	for _, adj := range sig.Adjustments {
		factors = append(factors, domain.Factor{
			Name:        adj.Reason,
			Description: fmt.Sprintf("Confidence adjustment of %.3f due to: %s", adj.Delta, adj.Reason),
			Weight:      adj.Delta,
		})
	}

	return factors
}

func (g *TemplateGenerator) buildSummary(sig *contractsv1.RiskAdjustedSignal) string {
	riskStr := sig.RiskLevel.String()
	confPct := sig.AdjustedConfidence * 100
	return fmt.Sprintf("TON/USDT signal with %.1f%% adjusted confidence and %s risk profile.", confPct, riskStr)
}

func (g *TemplateGenerator) buildRecommendation(sig *contractsv1.RiskAdjustedSignal) string {
	switch {
	case sig.RiskLevel == contractsv1.RiskLevel_RISK_LEVEL_LOW && sig.AdjustedConfidence >= 0.6:
		return "Consider position sizing proportional to confidence. Risk is managed."
	case sig.RiskLevel == contractsv1.RiskLevel_RISK_LEVEL_MEDIUM:
		return "Proceed with caution. Reduce position size relative to standard allocation."
	case sig.RiskLevel == contractsv1.RiskLevel_RISK_LEVEL_HIGH:
		return "High risk detected. Monitor closely and use tight stop-loss levels."
	default:
		return "Signal does not meet minimum criteria for action. Stand aside."
	}
}

func riskWeight(r contractsv1.RiskLevel) float64 {
	switch r {
	case contractsv1.RiskLevel_RISK_LEVEL_LOW:
		return 1.0
	case contractsv1.RiskLevel_RISK_LEVEL_MEDIUM:
		return 0.6
	case contractsv1.RiskLevel_RISK_LEVEL_HIGH:
		return 0.3
	default:
		return 0.1
	}
}
