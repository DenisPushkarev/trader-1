package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/collector-service/internal/domain"
)

// RSSAdapter is a stub adapter for RSS feed events.
type RSSAdapter struct {
	logger  zerolog.Logger
	counter int
}

// NewRSSAdapter creates a new RSSAdapter.
func NewRSSAdapter(logger zerolog.Logger) *RSSAdapter {
	return &RSSAdapter{logger: logger}
}

// SourceName returns the source identifier.
func (a *RSSAdapter) SourceName() domain.Source {
	return domain.SourceRSS
}

// FetchEvents returns stubbed RSS news items about TON.
func (a *RSSAdapter) FetchEvents(_ context.Context) ([]*domain.RawEvent, error) {
	a.counter++
	events := []*domain.RawEvent{
		{
			EventID:       uuid.NewString(),
			Source:        domain.SourceRSS,
			SourceEventID: fmt.Sprintf("rss-item-%d", a.counter*300),
			Payload:       fmt.Sprintf(`{"title":"TON blockchain reaches new transaction milestone","link":"https://blog.ton.org/milestone-%d","summary":"The Open Network processes record transactions."}`, a.counter),
			Timestamp:     time.Now(),
			Metadata:      map[string]string{"feed": "ton-blog", "type": "news"},
		},
	}
	a.logger.Debug().Int("count", len(events)).Msg("rss fetch")
	return events, nil
}
