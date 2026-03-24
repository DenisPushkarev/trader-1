package messaging

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/risk-engine-service/internal/domain"
)

// NATSPublisher publishes risk-adjusted signals.
type NATSPublisher struct {
	client *natsclient.Client
	logger zerolog.Logger
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(client *natsclient.Client, logger zerolog.Logger) *NATSPublisher {
	return &NATSPublisher{client: client, logger: logger}
}

// Publish publishes a risk-adjusted signal to signals.risk_adjusted.
func (p *NATSPublisher) Publish(ctx context.Context, sig *domain.RiskAdjustedSignal) error {
	adjustments := make([]*contractsv1.ConfidenceAdjustment, len(sig.Adjustments))
	for i, a := range sig.Adjustments {
		adjustments[i] = &contractsv1.ConfidenceAdjustment{Reason: a.Reason, Delta: a.Delta}
	}
	msg := &contractsv1.RiskAdjustedSignal{
		RiskSignalId:       sig.RiskSignalID,
		OriginalSignalId:   sig.OriginalSignalID,
		AdjustedConfidence: sig.AdjustedConfidence,
		RiskLevel:          contractsv1.RiskLevel(sig.RiskLevel),
		Blocked:            sig.Blocked,
		BlockReason:        sig.BlockReason,
		Adjustments:        adjustments,
		RiskConfigVersion:  sig.RiskConfigVersion,
	}
	if err := p.client.PublishJSON(ctx, contractsv1.SubjectSignalsRiskAdjusted, msg); err != nil {
		return fmt.Errorf("publish risk-adjusted: %w", err)
	}
	return nil
}
