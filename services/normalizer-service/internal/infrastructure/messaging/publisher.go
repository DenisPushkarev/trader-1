package messaging

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/normalizer-service/internal/domain"
)

// NATSPublisher publishes normalized events to NATS.
type NATSPublisher struct {
	client *natsclient.Client
	logger zerolog.Logger
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(client *natsclient.Client, logger zerolog.Logger) *NATSPublisher {
	return &NATSPublisher{client: client, logger: logger}
}

// Publish publishes a normalized event to events.normalized.
func (p *NATSPublisher) Publish(ctx context.Context, ev *domain.NormalizedEvent) error {
	msg := &contractsv1.NormalizedEvent{
		EventId: ev.EventID,
		SourceRef: &contractsv1.SourceReference{
			Source:        ev.SourceRef.Source,
			SourceEventId: ev.SourceRef.SourceEventID,
		},
		EventType:         ev.EventType,
		Asset:             ev.Asset,
		Sentiment:         ev.Sentiment,
		Impact:            ev.Impact,
		Content:           ev.Content,
		TimestampMs:       ev.Timestamp.UnixMilli(),
		EnrichmentVersion: ev.EnrichmentVersion,
		Metadata:          ev.Metadata,
	}
	if err := p.client.PublishJSON(ctx, contractsv1.SubjectEventsNormalized, msg); err != nil {
		return fmt.Errorf("publish normalized: %w", err)
	}
	return nil
}
