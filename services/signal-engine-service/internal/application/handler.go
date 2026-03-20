package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	shareddedup "github.com/trader-1/trader-1/packages/shared/dedup"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/domain"
)

const (
	configVersion = "v1"
	dedupTTL      = 1 * time.Hour
	windowSize    = 10 // events to accumulate before generating signal
)

// Publisher publishes generated signals.
type Publisher interface {
	Publish(ctx context.Context, signal *domain.GeneratedSignal) error
}

// ScoringModel scores a batch of normalized events against a market context.
type ScoringModel interface {
	Score(events []*contractsv1.NormalizedEvent, ctx *contractsv1.MarketContextSnapshot) (direction domain.Direction, confidence float64)
	ConfigVersion() string
}

// SignalHandler processes normalized events and market context to generate signals.
type SignalHandler struct {
	logger        zerolog.Logger
	publisher     Publisher
	dedup         *shareddedup.Deduplicator
	scoring       ScoringModel
	mu            sync.Mutex
	eventBuffer   []*contractsv1.NormalizedEvent
	latestContext *contractsv1.MarketContextSnapshot
}

// NewSignalHandler creates a SignalHandler.
func NewSignalHandler(
	logger zerolog.Logger,
	publisher Publisher,
	redis *redisclient.Client,
	scoring ScoringModel,
) *SignalHandler {
	return &SignalHandler{
		logger:      logger,
		publisher:   publisher,
		dedup:       shareddedup.NewDeduplicator(redis),
		scoring:     scoring,
		eventBuffer: make([]*contractsv1.NormalizedEvent, 0, windowSize),
	}
}

// HandleMarketContext processes a market context update message.
func (h *SignalHandler) HandleMarketContext(data []byte) error {
	var snap contractsv1.MarketContextSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("unmarshal market context: %w", err)
	}
	h.mu.Lock()
	h.latestContext = &snap
	h.mu.Unlock()
	h.logger.Debug().Str("context_id", snap.ContextId).Float64("price", snap.Price).Msg("market context updated")
	return nil
}

// HandleNormalizedEvent processes a normalized event and may generate a signal.
func (h *SignalHandler) HandleNormalizedEvent(data []byte) error {
	var ev contractsv1.NormalizedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("unmarshal normalized event: %w", err)
	}

	h.mu.Lock()
	h.eventBuffer = append(h.eventBuffer, &ev)
	buffered := make([]*contractsv1.NormalizedEvent, len(h.eventBuffer))
	copy(buffered, h.eventBuffer)
	ctx := h.latestContext
	h.mu.Unlock()

	if len(buffered) < windowSize {
		return nil
	}

	// Clear buffer atomically
	h.mu.Lock()
	h.eventBuffer = h.eventBuffer[:0]
	h.mu.Unlock()

	return h.generateSignal(context.Background(), buffered, ctx)
}

func (h *SignalHandler) generateSignal(ctx context.Context, events []*contractsv1.NormalizedEvent, marketCtx *contractsv1.MarketContextSnapshot) error {
	direction, confidence := h.scoring.Score(events, marketCtx)

	// Deterministic signal ID from sorted event IDs + context + config version
	ids := make([]string, len(events))
	for i, ev := range events {
		ids[i] = ev.EventId
	}
	sort.Strings(ids)

	ctxID := ""
	if marketCtx != nil {
		ctxID = marketCtx.ContextId
	}

	idInput := fmt.Sprintf("%v:%s:%s", ids, ctxID, h.scoring.ConfigVersion())
	signalID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(idInput)).String()

	isDup, err := h.dedup.IsDuplicate(ctx, shareddedup.SignalKey(signalID), dedupTTL)
	if err != nil {
		h.logger.Warn().Err(err).Msg("dedup check error")
	}
	if isDup {
		h.logger.Debug().Str("signal_id", signalID).Msg("duplicate signal, skipping")
		return nil
	}

	eventIDs := make([]string, len(events))
	for i, ev := range events {
		eventIDs[i] = ev.EventId
	}

	sig := &domain.GeneratedSignal{
		SignalID:           signalID,
		Direction:          direction,
		Confidence:         confidence,
		ContributingEvents: eventIDs,
		MarketContextID:    ctxID,
		HalfLifeSeconds:    3600,
		MinConfidence:      0.05,
		ConfigVersion:      h.scoring.ConfigVersion(),
		Timestamp:          time.Now(),
	}

	if err := h.publisher.Publish(ctx, sig); err != nil {
		return fmt.Errorf("publish signal: %w", err)
	}

	h.logger.Info().
		Str("signal_id", signalID).
		Int("direction", int(direction)).
		Float64("confidence", confidence).
		Int("events", len(events)).
		Msg("signal generated")

	return nil
}
