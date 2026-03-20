package domain

import "time"

// Direction represents signal trading direction.
type Direction int

const (
	DirectionUnspecified Direction = 0
	DirectionBullish     Direction = 1
	DirectionBearish     Direction = 2
	DirectionNeutral     Direction = 3
)

// GeneratedSignal is a trading signal produced by the scoring engine.
type GeneratedSignal struct {
	SignalID           string
	Direction          Direction
	Confidence         float64
	ContributingEvents []string
	MarketContextID    string
	HalfLifeSeconds    int64
	MinConfidence      float64
	ConfigVersion      string
	Timestamp          time.Time
}

// DecayedConfidence returns confidence adjusted for elapsed time since signal generation.
func (s *GeneratedSignal) DecayedConfidence(now time.Time) float64 {
	if s.HalfLifeSeconds <= 0 {
		return s.Confidence
	}
	elapsed := now.Sub(s.Timestamp).Seconds()
	halfLife := float64(s.HalfLifeSeconds)
	decayed := s.Confidence * pow2(-elapsed/halfLife)
	if decayed < s.MinConfidence {
		return s.MinConfidence
	}
	return decayed
}

// pow2 returns 2^x using successive multiplication approximation.
func pow2(x float64) float64 {
	// 2^x = e^(x * ln2)
	// Use a simple approach: if x is 0, return 1; otherwise use exponential decay
	if x == 0 {
		return 1.0
	}
	// ln(2) ≈ 0.693147
	return expApprox(x * 0.693147)
}

// expApprox is a simple e^x approximation using Taylor series for small values.
func expApprox(x float64) float64 {
	// For simplicity, use stdlib math. This avoids importing math just for this.
	// We'll use a Horner's method approximation good enough for decay.
	result := 1.0
	term := 1.0
	for i := 1; i <= 20; i++ {
		term *= x / float64(i)
		result += term
		if term < 1e-10 && term > -1e-10 {
			break
		}
	}
	return result
}
