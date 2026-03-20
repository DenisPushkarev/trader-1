package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/services/market-context-service/internal/domain"
)

const cacheTTL = 120 * time.Second
const cacheKeyPrefix = "cache:market:"

// RedisCache stores market context snapshots in Redis.
type RedisCache struct {
	redis  *redisclient.Client
	logger zerolog.Logger
}

// NewRedisCache creates a RedisCache.
func NewRedisCache(redis *redisclient.Client, logger zerolog.Logger) *RedisCache {
	return &RedisCache{redis: redis, logger: logger}
}

// Set stores the snapshot under cache:market:{asset}:latest.
func (c *RedisCache) Set(ctx context.Context, snap *domain.MarketContextSnapshot) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	key := cacheKeyPrefix + snap.Asset + ":latest"
	return c.redis.Set(ctx, key, string(data), cacheTTL)
}

// Get retrieves the latest snapshot for an asset.
func (c *RedisCache) Get(ctx context.Context, asset string) (*domain.MarketContextSnapshot, error) {
	key := cacheKeyPrefix + asset + ":latest"
	val, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("redis get %s: %w", key, err)
	}
	var snap domain.MarketContextSnapshot
	if err := json.Unmarshal([]byte(val), &snap); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return &snap, nil
}
