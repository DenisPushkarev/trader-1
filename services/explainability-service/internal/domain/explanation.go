package domain

// ExplainedSignal is the human-readable form of a risk-adjusted signal.
type ExplainedSignal struct {
	ExplainID            string
	SignalID             string
	Summary              string
	Factors              []Factor
	Recommendation       string
	ExplainConfigVersion string
}

// Factor is a named scoring contributor in the explanation.
type Factor struct {
	Name        string
	Description string
	Weight      float64
}
