package cache

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/domain"
)

const (
	eventListKey = "cache:events:latest"
	maxEvents    = 200
)

// EventCache stores event read models in memory with Redis backup.
type EventCache struct {
	redis  *redisclient.Client
	logger zerolog.Logger
	events []*domain.EventReadModel
}

// NewEventCache creates an EventCache.
func NewEventCache(redis *redisclient.Client, logger zerolog.Logger) *EventCache {
	return &EventCache{redis: redis, logger: logger, events: make([]*domain.EventReadModel, 0)}
}

// Add adds a new event to the cache.
func (c *EventCache) Add(ev *domain.EventReadModel) {
	c.events = append([]*domain.EventReadModel{ev}, c.events...)
	if len(c.events) > maxEvents {
		c.events = c.events[:maxEvents]
	}
	if data, err := json.Marshal(c.events); err == nil {
		c.redis.Set(context.Background(), eventListKey, string(data), signalTTL) //nolint:errcheck
	}
}

// GetLatest returns the most recent events.
func (c *EventCache) GetLatest(_ context.Context, limit int) ([]*domain.EventReadModel, error) {
	if len(c.events) == 0 {
		return []*domain.EventReadModel{}, nil
	}
	if limit <= 0 || limit > len(c.events) {
		limit = len(c.events)
	}
	result := make([]*domain.EventReadModel, limit)
	copy(result, c.events[:limit])
	return result, nil
}
