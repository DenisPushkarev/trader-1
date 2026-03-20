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
	"github.com/trader-1/trader-1/services/explainability-service/internal/domain"
)

const (
	explainConfigVersion = "v1"
	dedupTTL             = 1 * time.Hour
)

// Publisher publishes explained signals.
type Publisher interface {
	Publish(ctx context.Context, sig *domain.ExplainedSignal) error
}

// Generator generates human-readable explanations.
type Generator interface {
	Generate(sig *contractsv1.RiskAdjustedSignal) *domain.ExplainedSignal
	ConfigVersion() string
}

// ExplainHandler processes risk-adjusted signals and publishes explanations.
type ExplainHandler struct {
	logger    zerolog.Logger
	publisher Publisher
	dedup     *shareddedup.Deduplicator
	generator Generator
}

// NewExplainHandler creates an ExplainHandler.
func NewExplainHandler(
	logger zerolog.Logger,
	publisher Publisher,
	redis *redisclient.Client,
	generator Generator,
) *ExplainHandler {
	return &ExplainHandler{
		logger:    logger,
		publisher: publisher,
		dedup:     shareddedup.NewDeduplicator(redis),
		generator: generator,
	}
}

// Handle processes a raw NATS message payload.
func (h *ExplainHandler) Handle(data []byte) error {
	ctx := context.Background()

	var sig contractsv1.RiskAdjustedSignal
	if err := json.Unmarshal(data, &sig); err != nil {
		return fmt.Errorf("unmarshal risk-adjusted signal: %w", err)
	}

	// Skip blocked signals — no explanation needed
	if sig.Blocked {
		h.logger.Debug().Str("signal_id", sig.OriginalSignalId).Msg("skipping blocked signal")
		return nil
	}

	explainID := uuid.NewSHA1(uuid.NameSpaceURL,
		[]byte(sig.RiskSignalId+":"+explainConfigVersion)).String()

	isDup, err := h.dedup.IsDuplicate(ctx, shareddedup.ExplainKey(explainID), dedupTTL)
	if err != nil {
		h.logger.Warn().Err(err).Msg("dedup check error")
	}
	if isDup {
		h.logger.Debug().Str("explain_id", explainID).Msg("duplicate explanation, skipping")
		return nil
	}

	explained := h.generator.Generate(&sig)
	explained.ExplainID = explainID
	explained.ExplainConfigVersion = explainConfigVersion

	if err := h.publisher.Publish(ctx, explained); err != nil {
		return fmt.Errorf("publish explained signal: %w", err)
	}

	h.logger.Info().
		Str("explain_id", explainID).
		Str("signal_id", sig.OriginalSignalId).
		Str("summary", explained.Summary).
		Msg("explained signal published")

	return nil
}
