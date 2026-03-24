package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	shareddedup "github.com/trader-1/trader-1/packages/shared/dedup"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/services/normalizer-service/internal/domain"
)

const (
	enrichmentVersion = "v1"
	dedupTTL          = 24 * time.Hour
)

// Publisher publishes normalized events.
type Publisher interface {
	Publish(ctx context.Context, ev *domain.NormalizedEvent) error
}

// SentimentAnalyzer scores text sentiment in [-1, 1].
type SentimentAnalyzer interface {
	Analyze(text string) float64
}

// ImpactScorer scores event impact in [0, 1].
type ImpactScorer interface {
	Score(source, eventType, text string) float64
}

// NormalizeHandler processes raw events and publishes normalized events.
type NormalizeHandler struct {
	logger    zerolog.Logger
	publisher Publisher
	dedup     *shareddedup.Deduplicator
	sentiment SentimentAnalyzer
	impact    ImpactScorer
}

// NewNormalizeHandler creates a NormalizeHandler.
func NewNormalizeHandler(
	logger zerolog.Logger,
	publisher Publisher,
	redis *redisclient.Client,
	sentiment SentimentAnalyzer,
	impact ImpactScorer,
) *NormalizeHandler {
	return &NormalizeHandler{
		logger:    logger,
		publisher: publisher,
		dedup:     shareddedup.NewDeduplicator(redis),
		sentiment: sentiment,
		impact:    impact,
	}
}

// Handle processes a raw NATS message payload.
func (h *NormalizeHandler) Handle(data []byte) error {
	ctx := context.Background()

	var raw contractsv1.RawEvent
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal raw event: %w", err)
	}

	// Deterministic event ID: UUID v5 from source+sourceEventId+version
	eventID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(raw.Source+":"+raw.SourceEventId+":"+enrichmentVersion)).String()

	isDup, err := h.dedup.IsDuplicate(ctx, shareddedup.NormalizedEventKey(eventID), dedupTTL)
	if err != nil {
		h.logger.Warn().Err(err).Msg("dedup check error, proceeding")
	}
	if isDup {
		h.logger.Debug().Str("event_id", eventID).Msg("duplicate normalized event, skipping")
		return nil
	}

	content := raw.Payload
	sentiment := h.sentiment.Analyze(content)
	impact := h.impact.Score(raw.Source, raw.Metadata["type"], content)

	ev := &domain.NormalizedEvent{
		EventID: eventID,
		SourceRef: domain.SourceRef{
			Source:        raw.Source,
			SourceEventID: raw.SourceEventId,
		},
		EventType:         classifyEventType(raw.Source, raw.Metadata),
		Asset:             "TON/USDT",
		Sentiment:         sentiment,
		Impact:            impact,
		Content:           content,
		Timestamp:         time.UnixMilli(raw.TimestampMs),
		EnrichmentVersion: enrichmentVersion,
		Metadata:          raw.Metadata,
	}

	if err := h.publisher.Publish(ctx, ev); err != nil {
		return fmt.Errorf("publish normalized event: %w", err)
	}

	h.logger.Info().
		Str("event_id", eventID).
		Str("source", raw.Source).
		Float64("sentiment", sentiment).
		Float64("impact", impact).
		Msg("normalized event published")

	return nil
}

func classifyEventType(source string, metadata map[string]string) string {
	if t, ok := metadata["type"]; ok && t != "" {
		return t
	}
	switch source {
	case "telegram":
		return "social_post"
	case "twitter":
		return "social_post"
	case "rss":
		return "news"
	case "exchange":
		return "listing"
	default:
		return "unknown"
	}
}
