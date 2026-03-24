package domain

// RiskLevel represents signal risk classification.
type RiskLevel int

const (
	RiskLevelUnspecified RiskLevel = 0
	RiskLevelLow        RiskLevel = 1
	RiskLevelMedium     RiskLevel = 2
	RiskLevelHigh       RiskLevel = 3
	RiskLevelCritical   RiskLevel = 4
)

// Adjustment is a confidence modification with a reason.
type Adjustment struct {
	Reason string
	Delta  float64
}

// RiskAdjustedSignal is a signal with risk evaluation applied.
type RiskAdjustedSignal struct {
	RiskSignalID       string
	OriginalSignalID   string
	AdjustedConfidence float64
	RiskLevel          RiskLevel
	Blocked            bool
	BlockReason        string
	Adjustments        []Adjustment
	RiskConfigVersion  string
}
