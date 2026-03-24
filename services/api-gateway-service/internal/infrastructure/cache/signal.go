package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/domain"
)

const (
	signalListKey = "cache:signals:latest"
	signalTTL     = 30 * time.Second
	maxSignals    = 100
)

// SignalCache stores signal read models in Redis.
type SignalCache struct {
	redis  *redisclient.Client
	logger zerolog.Logger
	// in-memory fallback
	signals []*domain.SignalReadModel
}

// NewSignalCache creates a SignalCache.
func NewSignalCache(redis *redisclient.Client, logger zerolog.Logger) *SignalCache {
	return &SignalCache{redis: redis, logger: logger, signals: make([]*domain.SignalReadModel, 0)}
}

// Add stores a new signal in the cache (prepend, keep last N).
func (c *SignalCache) Add(sig *domain.SignalReadModel) {
	c.signals = append([]*domain.SignalReadModel{sig}, c.signals...)
	if len(c.signals) > maxSignals {
		c.signals = c.signals[:maxSignals]
	}
	// Best-effort cache to Redis
	if data, err := json.Marshal(c.signals); err == nil {
		c.redis.Set(context.Background(), signalListKey, string(data), signalTTL) //nolint:errcheck
	}
}

// GetLatest returns the most recent signals.
func (c *SignalCache) GetLatest(_ context.Context, limit int) ([]*domain.SignalReadModel, error) {
	if len(c.signals) == 0 {
		return []*domain.SignalReadModel{}, nil
	}
	if limit <= 0 || limit > len(c.signals) {
		limit = len(c.signals)
	}
	result := make([]*domain.SignalReadModel, limit)
	copy(result, c.signals[:limit])
	return result, nil
}

// GetHistory returns paginated signals.
func (c *SignalCache) GetHistory(_ context.Context, page, pageSize int) ([]*domain.SignalReadModel, error) {
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	if start >= len(c.signals) {
		return []*domain.SignalReadModel{}, nil
	}
	end := start + pageSize
	if end > len(c.signals) {
		end = len(c.signals)
	}
	result := make([]*domain.SignalReadModel, end-start)
	copy(result, c.signals[start:end])
	return result, nil
}

// StoreJSON stores a pre-marshalled signal list.
func (c *SignalCache) StoreJSON(_ context.Context, key, value string, ttl time.Duration) error {
	return c.redis.Set(context.Background(), key, value, ttl)
}

// formatKey returns a formatted cache key.
func formatKey(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

var _ = formatKey // suppress unused warning
