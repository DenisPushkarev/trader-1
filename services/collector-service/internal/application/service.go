package application

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/collector-service/internal/domain"
)

// SourceAdapter polls or receives events from a specific external source.
type SourceAdapter interface {
	FetchEvents(ctx context.Context) ([]*domain.RawEvent, error)
	SourceName() domain.Source
}

// Publisher publishes raw events to the message bus.
type Publisher interface {
	Publish(ctx context.Context, event *domain.RawEvent) error
}

// Deduplicator checks and marks events as processed.
type Deduplicator interface {
	IsDuplicate(ctx context.Context, source string, sourceEventID string) (bool, error)
}

// CollectService orchestrates event collection from all sources.
type CollectService struct {
	logger       zerolog.Logger
	publisher    Publisher
	dedup        Deduplicator
	adapters     []SourceAdapter
	pollInterval time.Duration
}

// NewCollectService creates a new CollectService.
func NewCollectService(
	logger zerolog.Logger,
	publisher Publisher,
	dedup Deduplicator,
	adapters []SourceAdapter,
	pollInterval time.Duration,
) *CollectService {
	return &CollectService{
		logger:       logger,
		publisher:    publisher,
		dedup:        dedup,
		adapters:     adapters,
		pollInterval: pollInterval,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (s *CollectService) Run(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Poll immediately on start
	s.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("collector stopped")
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *CollectService) poll(ctx context.Context) {
	for _, adapter := range s.adapters {
		events, err := adapter.FetchEvents(ctx)
		if err != nil {
			s.logger.Error().Err(err).Str("source", string(adapter.SourceName())).Msg("fetch events error")
			continue
		}
		for _, ev := range events {
			s.processEvent(ctx, ev)
		}
	}
}

func (s *CollectService) processEvent(ctx context.Context, ev *domain.RawEvent) {
	isDup, err := s.dedup.IsDuplicate(ctx, string(ev.Source), ev.SourceEventID)
	if err != nil {
		s.logger.Error().Err(err).Str("event_id", ev.EventID).Msg("dedup check error")
		return
	}
	if isDup {
		s.logger.Debug().Str("event_id", ev.EventID).Msg("duplicate event, skipping")
		return
	}
	if err := s.publisher.Publish(ctx, ev); err != nil {
		s.logger.Error().Err(err).Str("event_id", ev.EventID).Msg("publish error")
	}
}
