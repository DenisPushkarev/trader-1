package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/market-context-service/internal/domain"
)

// PriceProvider fetches current market data.
type PriceProvider interface {
	FetchSnapshot(ctx context.Context, asset string) (*domain.MarketContextSnapshot, error)
}

// Publisher publishes market context snapshots.
type Publisher interface {
	Publish(ctx context.Context, snap *domain.MarketContextSnapshot) error
}

// Cache stores the latest market context.
type Cache interface {
	Set(ctx context.Context, snap *domain.MarketContextSnapshot) error
	Get(ctx context.Context, asset string) (*domain.MarketContextSnapshot, error)
}

// UpdateContextService periodically fetches and publishes market context.
type UpdateContextService struct {
	logger         zerolog.Logger
	provider       PriceProvider
	publisher      Publisher
	cache          Cache
	updateInterval time.Duration
}

// NewUpdateContextService creates a new UpdateContextService.
func NewUpdateContextService(
	logger zerolog.Logger,
	provider PriceProvider,
	publisher Publisher,
	cache Cache,
	updateInterval time.Duration,
) *UpdateContextService {
	return &UpdateContextService{
		logger:         logger,
		provider:       provider,
		publisher:      publisher,
		cache:          cache,
		updateInterval: updateInterval,
	}
}

// Run starts the periodic update loop.
func (s *UpdateContextService) Run(ctx context.Context) {
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	s.update(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("market context service stopped")
			return
		case <-ticker.C:
			s.update(ctx)
		}
	}
}

func (s *UpdateContextService) update(ctx context.Context) {
	snap, err := s.provider.FetchSnapshot(ctx, "TON/USDT")
	if err != nil {
		s.logger.Error().Err(err).Msg("fetch market snapshot error")
		return
	}
	snap.ContextID = uuid.NewSHA1(uuid.NameSpaceURL,
		[]byte(fmt.Sprintf("%s:%d", snap.Asset, snap.Timestamp.UnixMilli()))).String()

	if err := s.cache.Set(ctx, snap); err != nil {
		s.logger.Warn().Err(err).Msg("cache set error")
	}

	if err := s.publisher.Publish(ctx, snap); err != nil {
		s.logger.Error().Err(err).Msg("publish market context error")
		return
	}

	s.logger.Info().
		Str("context_id", snap.ContextID).
		Float64("price", snap.Price).
		Float64("volatility", snap.Volatility).
		Msg("market context updated")
}
