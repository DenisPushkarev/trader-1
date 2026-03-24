package messaging

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/domain"
)

// NATSPublisher publishes generated signals.
type NATSPublisher struct {
	client *natsclient.Client
	logger zerolog.Logger
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(client *natsclient.Client, logger zerolog.Logger) *NATSPublisher {
	return &NATSPublisher{client: client, logger: logger}
}

// Publish publishes a generated signal to signals.generated.
func (p *NATSPublisher) Publish(ctx context.Context, sig *domain.GeneratedSignal) error {
	msg := &contractsv1.GeneratedSignal{
		SignalId:           sig.SignalID,
		Direction:          contractsv1.Direction(sig.Direction),
		Confidence:         sig.Confidence,
		ContributingEvents: sig.ContributingEvents,
		MarketContextId:    sig.MarketContextID,
		DecayConfig: &contractsv1.DecayConfig{
			HalfLifeSeconds: sig.HalfLifeSeconds,
			MinConfidence:   sig.MinConfidence,
		},
		ConfigVersion: sig.ConfigVersion,
		TimestampMs:   sig.Timestamp.UnixMilli(),
	}
	if err := p.client.PublishJSON(ctx, contractsv1.SubjectSignalsGenerated, msg); err != nil {
		return fmt.Errorf("publish generated signal: %w", err)
	}
	return nil
}
