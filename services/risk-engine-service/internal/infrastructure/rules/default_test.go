package rules_test

import (
	"testing"

	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/services/risk-engine-service/internal/domain"
	"github.com/trader-1/trader-1/services/risk-engine-service/internal/infrastructure/rules"
)

func TestDefaultRuleSet_BlocksLowConfidence(t *testing.T) {
	r := rules.NewDefaultRuleSet()
	sig := &contractsv1.GeneratedSignal{
		SignalId:           "sig-1",
		Confidence:         0.1,
		Direction:          contractsv1.Direction_DIRECTION_BULLISH,
		ContributingEvents: []string{"e1", "e2"},
	}
	result := r.Apply(sig)
	if !result.Blocked {
		t.Error("expected signal to be blocked due to low confidence")
	}
}

func TestDefaultRuleSet_BlocksInsufficientSources(t *testing.T) {
	r := rules.NewDefaultRuleSet()
	sig := &contractsv1.GeneratedSignal{
		SignalId:           "sig-2",
		Confidence:         0.8,
		Direction:          contractsv1.Direction_DIRECTION_BULLISH,
		ContributingEvents: []string{"e1"}, // only 1 event
	}
	result := r.Apply(sig)
	if !result.Blocked {
		t.Error("expected signal to be blocked due to insufficient sources")
	}
	if result.BlockReason != "insufficient_source_confirmation" {
		t.Errorf("unexpected block reason: %s", result.BlockReason)
	}
}

func TestDefaultRuleSet_AssignsLowRiskForHighConfidence(t *testing.T) {
	r := rules.NewDefaultRuleSet()
	sig := &contractsv1.GeneratedSignal{
		SignalId:           "sig-3",
		Confidence:         0.85,
		Direction:          contractsv1.Direction_DIRECTION_BULLISH,
		ContributingEvents: []string{"e1", "e2", "e3"},
	}
	result := r.Apply(sig)
	if result.Blocked {
		t.Errorf("expected signal NOT to be blocked, blocked reason: %s", result.BlockReason)
	}
	if result.RiskLevel != domain.RiskLevelLow {
		t.Errorf("expected LOW risk, got %d", result.RiskLevel)
	}
}

func TestDefaultRuleSet_NeutralPenalty(t *testing.T) {
	r := rules.NewDefaultRuleSet()
	sig := &contractsv1.GeneratedSignal{
		SignalId:           "sig-4",
		Confidence:         0.5,
		Direction:          contractsv1.Direction_DIRECTION_NEUTRAL,
		ContributingEvents: []string{"e1", "e2"},
	}
	result := r.Apply(sig)
	if result.AdjustedConfidence >= 0.5 {
		t.Errorf("expected confidence to be penalized below 0.5, got %f", result.AdjustedConfidence)
	}
	if len(result.Adjustments) == 0 {
		t.Error("expected at least one adjustment recorded")
	}
}
