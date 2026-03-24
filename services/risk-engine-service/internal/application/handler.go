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
	"github.com/trader-1/trader-1/services/risk-engine-service/internal/domain"
)

const (
	riskConfigVersion = "v1"
	dedupTTL          = 1 * time.Hour
)

// Publisher publishes risk-adjusted signals.
type Publisher interface {
	Publish(ctx context.Context, sig *domain.RiskAdjustedSignal) error
}

// RuleSet applies risk rules to a signal.
type RuleSet interface {
	Apply(signal *contractsv1.GeneratedSignal) *domain.RiskAdjustedSignal
	ConfigVersion() string
}

// RiskHandler processes generated signals and publishes risk-adjusted versions.
type RiskHandler struct {
	logger    zerolog.Logger
	publisher Publisher
	dedup     *shareddedup.Deduplicator
	rules     RuleSet
}

// NewRiskHandler creates a RiskHandler.
func NewRiskHandler(
	logger zerolog.Logger,
	publisher Publisher,
	redis *redisclient.Client,
	rules RuleSet,
) *RiskHandler {
	return &RiskHandler{
		logger:    logger,
		publisher: publisher,
		dedup:     shareddedup.NewDeduplicator(redis),
		rules:     rules,
	}
}

// Handle processes a raw NATS message payload.
func (h *RiskHandler) Handle(data []byte) error {
	ctx := context.Background()

	var sig contractsv1.GeneratedSignal
	if err := json.Unmarshal(data, &sig); err != nil {
		return fmt.Errorf("unmarshal generated signal: %w", err)
	}

	riskID := uuid.NewSHA1(uuid.NameSpaceURL,
		[]byte(sig.SignalId+":"+riskConfigVersion)).String()

	isDup, err := h.dedup.IsDuplicate(ctx, shareddedup.RiskKey(riskID), dedupTTL)
	if err != nil {
		h.logger.Warn().Err(err).Msg("dedup check error")
	}
	if isDup {
		h.logger.Debug().Str("risk_id", riskID).Msg("duplicate risk signal, skipping")
		return nil
	}

	adjusted := h.rules.Apply(&sig)
	adjusted.RiskSignalID = riskID
	adjusted.OriginalSignalID = sig.SignalId
	adjusted.RiskConfigVersion = riskConfigVersion

	if err := h.publisher.Publish(ctx, adjusted); err != nil {
		return fmt.Errorf("publish risk-adjusted signal: %w", err)
	}

	h.logger.Info().
		Str("risk_id", riskID).
		Str("signal_id", sig.SignalId).
		Bool("blocked", adjusted.Blocked).
		Int("risk_level", int(adjusted.RiskLevel)).
		Float64("confidence", adjusted.AdjustedConfidence).
		Msg("risk-adjusted signal published")

	return nil
}
