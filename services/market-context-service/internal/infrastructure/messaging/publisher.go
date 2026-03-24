package messaging

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/market-context-service/internal/domain"
)

// NATSPublisher publishes market context snapshots.
type NATSPublisher struct {
	client *natsclient.Client
	logger zerolog.Logger
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(client *natsclient.Client, logger zerolog.Logger) *NATSPublisher {
	return &NATSPublisher{client: client, logger: logger}
}

// Publish publishes a snapshot to market.context.updated.
func (p *NATSPublisher) Publish(ctx context.Context, snap *domain.MarketContextSnapshot) error {
	msg := &contractsv1.MarketContextSnapshot{
		ContextId:   snap.ContextID,
		Asset:       snap.Asset,
		Price:       snap.Price,
		Volume24H:   snap.Volume24H,
		Volatility:  snap.Volatility,
		TimestampMs: snap.Timestamp.UnixMilli(),
		Indicators:  snap.Indicators,
	}
	if err := p.client.PublishJSON(ctx, contractsv1.SubjectMarketContextUpdated, msg); err != nil {
		return fmt.Errorf("publish market context: %w", err)
	}
	return nil
}
