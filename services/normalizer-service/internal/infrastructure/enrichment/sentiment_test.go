package enrichment_test

import (
	"testing"

	"github.com/trader-1/trader-1/services/normalizer-service/internal/infrastructure/enrichment"
)

func TestSentimentAnalyzer(t *testing.T) {
	a := enrichment.NewSentimentAnalyzer()

	tests := []struct {
		text    string
		wantPos bool // true = positive expected
	}{
		{"TON bullish major partnership listing", true},
		{"crash dump scam warning bearish", false},
		{"TON blockchain update", false}, // neutral
	}

	for _, tt := range tests {
		score := a.Analyze(tt.text)
		if tt.wantPos && score <= 0 {
			t.Errorf("expected positive for %q, got %f", tt.text, score)
		}
		if !tt.wantPos && score > 0.3 {
			t.Errorf("expected non-positive for %q, got %f", tt.text, score)
		}
		if score < -1 || score > 1 {
			t.Errorf("score out of range [-1,1]: %f", score)
		}
	}
}

func TestImpactScorer(t *testing.T) {
	s := enrichment.NewImpactScorer()

	exchange := s.Score("exchange", "listing", "")
	twitter := s.Score("twitter", "social_post", "")

	if exchange <= twitter {
		t.Errorf("exchange listing should score higher than twitter post: %f vs %f", exchange, twitter)
	}
	if exchange > 1.0 || twitter < 0 {
		t.Errorf("scores out of range [0,1]: exchange=%f twitter=%f", exchange, twitter)
	}
}
