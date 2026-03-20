package enrichment

// ImpactScorer scores event impact deterministically.
type ImpactScorer struct{}

// NewImpactScorer creates an ImpactScorer.
func NewImpactScorer() *ImpactScorer {
	return &ImpactScorer{}
}

// Score returns an impact score in [0, 1] based on source and event type.
func (s *ImpactScorer) Score(source, eventType, _ string) float64 {
	sourceWeight := map[string]float64{
		"exchange": 0.9,
		"onchain":  0.8,
		"rss":      0.6,
		"telegram": 0.5,
		"twitter":  0.4,
		"miniapp":  0.3,
	}
	typeWeight := map[string]float64{
		"listing":      0.9,
		"news":         0.7,
		"announcement": 0.7,
		"social_post":  0.4,
		"unknown":      0.3,
	}

	sw := sourceWeight[source]
	if sw == 0 {
		sw = 0.3
	}
	tw := typeWeight[eventType]
	if tw == 0 {
		tw = 0.3
	}
	return (sw + tw) / 2.0
}
