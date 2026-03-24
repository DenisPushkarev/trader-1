package domain_test

import (
	"testing"
	"time"

	"github.com/trader-1/trader-1/services/signal-engine-service/internal/domain"
)

func TestDecayedConfidence_NoDecay(t *testing.T) {
	sig := &domain.GeneratedSignal{
		Confidence:      0.8,
		HalfLifeSeconds: 0,
		MinConfidence:   0.1,
		Timestamp:       time.Now(),
	}
	got := sig.DecayedConfidence(time.Now())
	if got != 0.8 {
		t.Errorf("expected 0.8, got %f", got)
	}
}

func TestDecayedConfidence_HalfLife(t *testing.T) {
	now := time.Now()
	sig := &domain.GeneratedSignal{
		Confidence:      1.0,
		HalfLifeSeconds: 3600,
		MinConfidence:   0.05,
		Timestamp:       now.Add(-3600 * time.Second), // exactly one half-life ago
	}
	got := sig.DecayedConfidence(now)
	// After one half-life, confidence should be ~0.5
	if got < 0.45 || got > 0.55 {
		t.Errorf("expected ~0.5 after one half-life, got %f", got)
	}
}

func TestDecayedConfidence_MinFloor(t *testing.T) {
	now := time.Now()
	sig := &domain.GeneratedSignal{
		Confidence:      1.0,
		HalfLifeSeconds: 1,
		MinConfidence:   0.1,
		Timestamp:       now.Add(-24 * time.Hour), // very old
	}
	got := sig.DecayedConfidence(now)
	if got < 0.1 {
		t.Errorf("confidence should not fall below MinConfidence=0.1, got %f", got)
	}
}
