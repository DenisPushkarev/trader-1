package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/collector-service/internal/domain"
)

// TwitterAdapter is a stub adapter for Twitter/X events.
type TwitterAdapter struct {
	logger  zerolog.Logger
	counter int
}

// NewTwitterAdapter creates a new TwitterAdapter.
func NewTwitterAdapter(logger zerolog.Logger) *TwitterAdapter {
	return &TwitterAdapter{logger: logger}
}

// SourceName returns the source identifier.
func (a *TwitterAdapter) SourceName() domain.Source {
	return domain.SourceTwitter
}

// FetchEvents returns stubbed Twitter posts about TON.
func (a *TwitterAdapter) FetchEvents(_ context.Context) ([]*domain.RawEvent, error) {
	a.counter++
	events := []*domain.RawEvent{
		{
			EventID:       uuid.NewString(),
			Source:        domain.SourceTwitter,
			SourceEventID: fmt.Sprintf("tweet-%d", a.counter*200),
			Payload:       fmt.Sprintf(`{"text":"$TON is looking incredibly strong right now. Major exchange listing incoming? #TON #crypto #%d","author":"@crypto_whale"}`, a.counter),
			Timestamp:     time.Now(),
			Metadata:      map[string]string{"platform": "twitter", "type": "tweet"},
		},
	}
	a.logger.Debug().Int("count", len(events)).Msg("twitter fetch")
	return events, nil
}
