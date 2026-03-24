package generator_test

import (
	"strings"
	"testing"

	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/services/explainability-service/internal/infrastructure/generator"
)

func TestTemplateGenerator_NoFactorsForBlocked(t *testing.T) {
	g := generator.NewTemplateGenerator()
	sig := &contractsv1.RiskAdjustedSignal{
		OriginalSignalId:   "sig-1",
		AdjustedConfidence: 0.7,
		RiskLevel:          contractsv1.RiskLevel_RISK_LEVEL_LOW,
		Adjustments:        nil,
	}
	result := g.Generate(sig)
	if result.SignalID != "sig-1" {
		t.Errorf("expected signal_id=sig-1, got %s", result.SignalID)
	}
	if result.Summary == "" {
		t.Error("summary should not be empty")
	}
	if result.Recommendation == "" {
		t.Error("recommendation should not be empty")
	}
	if len(result.Factors) == 0 {
		t.Error("expected at least base factors")
	}
}

func TestTemplateGenerator_IncludesAdjustmentsAsFactors(t *testing.T) {
	g := generator.NewTemplateGenerator()
	sig := &contractsv1.RiskAdjustedSignal{
		OriginalSignalId:   "sig-2",
		AdjustedConfidence: 0.4,
		RiskLevel:          contractsv1.RiskLevel_RISK_LEVEL_MEDIUM,
		Adjustments: []*contractsv1.ConfidenceAdjustment{
			{Reason: "neutral_direction_penalty", Delta: -0.1},
		},
	}
	result := g.Generate(sig)
	found := false
	for _, f := range result.Factors {
		if strings.Contains(f.Name, "neutral") {
			found = true
		}
	}
	if !found {
		t.Error("expected adjustment factor in explanation")
	}
}

func TestTemplateGenerator_LowRiskRecommendation(t *testing.T) {
	g := generator.NewTemplateGenerator()
	sig := &contractsv1.RiskAdjustedSignal{
		OriginalSignalId:   "sig-3",
		AdjustedConfidence: 0.75,
		RiskLevel:          contractsv1.RiskLevel_RISK_LEVEL_LOW,
	}
	result := g.Generate(sig)
	if !strings.Contains(result.Recommendation, "position") {
		t.Errorf("expected position sizing recommendation, got: %s", result.Recommendation)
	}
}
