package subscriber

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/domain"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/infrastructure/cache"
)

// NATSSubscriber listens for explained signals and events to populate caches.
type NATSSubscriber struct {
	nc          *natsclient.Client
	signalCache *cache.SignalCache
	eventCache  *cache.EventCache
	logger      zerolog.Logger
}

// NewNATSSubscriber creates a NATSSubscriber.
func NewNATSSubscriber(
	nc *natsclient.Client,
	signalCache *cache.SignalCache,
	eventCache *cache.EventCache,
	logger zerolog.Logger,
) *NATSSubscriber {
	return &NATSSubscriber{nc: nc, signalCache: signalCache, eventCache: eventCache, logger: logger}
}

// Start begins consuming from NATS subjects to populate read models.
func (s *NATSSubscriber) Start(ctx context.Context) {
	go func() {
		cfg := natsclient.ConsumerConfig{
			Stream:  contractsv1.StreamSignalsExplained,
			Durable: "api-gateway-signals-explained",
			Subject: contractsv1.SubjectSignalsExplained,
		}
		s.nc.ConsumeMessages(ctx, cfg, s.handleExplainedSignal) //nolint:errcheck
	}()
	go func() {
		cfg := natsclient.ConsumerConfig{
			Stream:  contractsv1.StreamEventsNormalized,
			Durable: "api-gateway-events-normalized",
			Subject: contractsv1.SubjectEventsNormalized,
		}
		s.nc.ConsumeMessages(ctx, cfg, s.handleNormalizedEvent) //nolint:errcheck
	}()
}

func (s *NATSSubscriber) handleExplainedSignal(data []byte) error {
	var sig contractsv1.ExplainedSignal
	if err := json.Unmarshal(data, &sig); err != nil {
		s.logger.Warn().Err(err).Msg("unmarshal explained signal")
		return nil
	}
	model := &domain.SignalReadModel{
		SignalID:       sig.SignalId,
		Summary:        sig.Summary,
		Recommendation: sig.Recommendation,
		Timestamp:      time.UnixMilli(sig.TimestampMs),
	}
	s.signalCache.Add(model)
	s.logger.Debug().Str("signal_id", sig.SignalId).Msg("signal read model updated")
	return nil
}

func (s *NATSSubscriber) handleNormalizedEvent(data []byte) error {
	var ev contractsv1.NormalizedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		s.logger.Warn().Err(err).Msg("unmarshal normalized event")
		return nil
	}
	model := &domain.EventReadModel{
		EventID:   ev.EventId,
		EventType: ev.EventType,
		Asset:     ev.Asset,
		Sentiment: ev.Sentiment,
		Impact:    ev.Impact,
		Content:   ev.Content,
		Timestamp: time.UnixMilli(ev.TimestampMs),
	}
	if ev.SourceRef != nil {
		model.Source = ev.SourceRef.Source
	}
	s.eventCache.Add(model)
	return nil
}
