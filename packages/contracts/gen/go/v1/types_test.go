package contractsv1_test

import (
	"testing"

	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
)

func TestDirectionString(t *testing.T) {
	tests := []struct {
		d    contractsv1.Direction
		want string
	}{
		{contractsv1.Direction_DIRECTION_BULLISH, "BULLISH"},
		{contractsv1.Direction_DIRECTION_BEARISH, "BEARISH"},
		{contractsv1.Direction_DIRECTION_NEUTRAL, "NEUTRAL"},
		{contractsv1.Direction_DIRECTION_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("Direction(%d).String() = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		r    contractsv1.RiskLevel
		want string
	}{
		{contractsv1.RiskLevel_RISK_LEVEL_LOW, "LOW"},
		{contractsv1.RiskLevel_RISK_LEVEL_MEDIUM, "MEDIUM"},
		{contractsv1.RiskLevel_RISK_LEVEL_HIGH, "HIGH"},
		{contractsv1.RiskLevel_RISK_LEVEL_CRITICAL, "CRITICAL"},
		{contractsv1.RiskLevel_RISK_LEVEL_UNSPECIFIED, "UNSPECIFIED"},
	}
	for _, tt := range tests {
		if got := tt.r.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.r, got, tt.want)
		}
	}
}

func TestSubjectConstants(t *testing.T) {
	expected := map[string]string{
		"SubjectEventsRaw":            contractsv1.SubjectEventsRaw,
		"SubjectEventsNormalized":     contractsv1.SubjectEventsNormalized,
		"SubjectMarketContextUpdated": contractsv1.SubjectMarketContextUpdated,
		"SubjectSignalsGenerated":     contractsv1.SubjectSignalsGenerated,
		"SubjectSignalsRiskAdjusted":  contractsv1.SubjectSignalsRiskAdjusted,
		"SubjectSignalsExplained":     contractsv1.SubjectSignalsExplained,
	}
	for name, val := range expected {
		if val == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}
