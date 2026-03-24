package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/explainability-service/internal/domain"
)

// NATSPublisher publishes explained signals.
type NATSPublisher struct {
	client *natsclient.Client
	logger zerolog.Logger
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(client *natsclient.Client, logger zerolog.Logger) *NATSPublisher {
	return &NATSPublisher{client: client, logger: logger}
}

// Publish publishes an explained signal to signals.explained.
func (p *NATSPublisher) Publish(ctx context.Context, sig *domain.ExplainedSignal) error {
	factors := make([]*contractsv1.ExplanationFactor, len(sig.Factors))
	for i, f := range sig.Factors {
		factors[i] = &contractsv1.ExplanationFactor{Name: f.Name, Description: f.Description, Weight: f.Weight}
	}
	msg := &contractsv1.ExplainedSignal{
		ExplainId:            sig.ExplainID,
		SignalId:             sig.SignalID,
		Summary:              sig.Summary,
		Factors:              factors,
		Recommendation:       sig.Recommendation,
		ExplainConfigVersion: sig.ExplainConfigVersion,
		TimestampMs:          time.Now().UnixMilli(),
	}
	if err := p.client.PublishJSON(ctx, contractsv1.SubjectSignalsExplained, msg); err != nil {
		return fmt.Errorf("publish explained signal: %w", err)
	}
	return nil
}
