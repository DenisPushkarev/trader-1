package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/collector-service/internal/domain"
)

// TelegramAdapter is a stub adapter for Telegram events.
type TelegramAdapter struct {
	logger  zerolog.Logger
	counter int
}

// NewTelegramAdapter creates a new TelegramAdapter.
func NewTelegramAdapter(logger zerolog.Logger) *TelegramAdapter {
	return &TelegramAdapter{logger: logger}
}

// SourceName returns the source identifier.
func (a *TelegramAdapter) SourceName() domain.Source {
	return domain.SourceTelegram
}

// FetchEvents returns stubbed Telegram channel messages.
func (a *TelegramAdapter) FetchEvents(_ context.Context) ([]*domain.RawEvent, error) {
	a.counter++
	events := []*domain.RawEvent{
		{
			EventID:       uuid.NewString(),
			Source:        domain.SourceTelegram,
			SourceEventID: fmt.Sprintf("tg-msg-%d", a.counter*100),
			Payload:       fmt.Sprintf(`{"text":"TON Network just announced a major partnership! Bullish on $TON #%d","channel":"@toncoin_official"}`, a.counter),
			Timestamp:     time.Now(),
			Metadata:      map[string]string{"channel": "@toncoin_official", "type": "announcement"},
		},
	}
	a.logger.Debug().Int("count", len(events)).Msg("telegram fetch")
	return events, nil
}
