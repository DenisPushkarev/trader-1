package dedup

import (
	"context"
	"time"

	shareddedup "github.com/trader-1/trader-1/packages/shared/dedup"
	redisclient "github.com/trader-1/trader-1/packages/shared/redis"
)

const dedupTTL = 24 * time.Hour

// CollectorDedup implements Deduplicator for the collector service.
type CollectorDedup struct {
	dedup *shareddedup.Deduplicator
}

// NewCollectorDedup creates a new CollectorDedup.
func NewCollectorDedup(redis *redisclient.Client) *CollectorDedup {
	return &CollectorDedup{dedup: shareddedup.NewDeduplicator(redis)}
}

// IsDuplicate returns true if this source+sourceEventID has already been processed.
func (d *CollectorDedup) IsDuplicate(ctx context.Context, source, sourceEventID string) (bool, error) {
	key := shareddedup.RawEventKey(source, sourceEventID)
	return d.dedup.IsDuplicate(ctx, key, dedupTTL)
}
