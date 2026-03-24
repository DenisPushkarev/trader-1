package dedup

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
)

// Deduplicator uses Redis to ensure at-most-once processing per logical event.
type Deduplicator struct {
	redis *redisclient.Client
}

// NewDeduplicator creates a new Deduplicator backed by the given Redis client.
func NewDeduplicator(redis *redisclient.Client) *Deduplicator {
	return &Deduplicator{redis: redis}
}

// IsDuplicate returns true if the key has already been processed.
// It also marks the key atomically (SetNX), so a true result means "already processed",
// and false means "first time — now marked".
func (d *Deduplicator) IsDuplicate(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	set, err := d.redis.SetNX(ctx, key, "1", ttl)
	if err != nil {
		return false, fmt.Errorf("dedup setnx %s: %w", key, err)
	}
	// SetNX returns true if the key was newly set (i.e., NOT a duplicate)
	return !set, nil
}

// Mark explicitly marks a key as processed (idempotent).
func (d *Deduplicator) Mark(ctx context.Context, key string, ttl time.Duration) error {
	return d.redis.Set(ctx, key, "1", ttl)
}

// RawEventKey returns the dedup key for a raw event.
func RawEventKey(source, sourceEventID string) string {
	return fmt.Sprintf("dedup:raw:%s:%s", source, sourceEventID)
}

// NormalizedEventKey returns the dedup key for a normalized event.
func NormalizedEventKey(eventID string) string {
	return fmt.Sprintf("dedup:norm:%s", eventID)
}

// SignalKey returns the dedup key for a generated signal.
func SignalKey(signalID string) string {
	return fmt.Sprintf("dedup:signal:%s", signalID)
}

// RiskKey returns the dedup key for a risk-adjusted signal.
func RiskKey(signalID string) string {
	return fmt.Sprintf("dedup:risk:%s", signalID)
}

// ExplainKey returns the dedup key for an explained signal.
func ExplainKey(signalID string) string {
	return fmt.Sprintf("dedup:explain:%s", signalID)
}

// ErrNotFound is re-exported from go-redis for callers to check.
var ErrNotFound = redis.Nil
