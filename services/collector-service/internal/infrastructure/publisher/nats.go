package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/collector-service/internal/domain"
)

// NATSPublisher publishes raw events to NATS.
type NATSPublisher struct {
	client *natsclient.Client
	logger zerolog.Logger
}

// NewNATSPublisher creates a new NATSPublisher.
func NewNATSPublisher(client *natsclient.Client, logger zerolog.Logger) *NATSPublisher {
	return &NATSPublisher{client: client, logger: logger}
}

// Publish marshals a domain RawEvent to a contracts RawEvent and publishes it.
func (p *NATSPublisher) Publish(ctx context.Context, ev *domain.RawEvent) error {
	msg := &contractsv1.RawEvent{
		EventId:       ev.EventID,
		Source:        string(ev.Source),
		SourceEventId: ev.SourceEventID,
		Payload:       ev.Payload,
		TimestampMs:   ev.Timestamp.UnixMilli(),
		Metadata:      ev.Metadata,
	}
	if err := p.client.PublishJSON(ctx, contractsv1.SubjectEventsRaw, msg); err != nil {
		return fmt.Errorf("publish raw event: %w", err)
	}
	p.logger.Info().
		Str("event_id", ev.EventID).
		Str("source", string(ev.Source)).
		Dur("age", time.Since(ev.Timestamp)).
		Msg("published raw event")
	return nil
}
