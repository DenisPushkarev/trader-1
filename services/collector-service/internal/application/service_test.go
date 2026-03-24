package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/collector-service/internal/application"
	"github.com/trader-1/trader-1/services/collector-service/internal/domain"
)

type mockAdapter struct {
	source domain.Source
	events []*domain.RawEvent
	err    error
}

func (m *mockAdapter) FetchEvents(_ context.Context) ([]*domain.RawEvent, error) {
	return m.events, m.err
}
func (m *mockAdapter) SourceName() domain.Source { return m.source }

type mockPublisher struct {
	published []*domain.RawEvent
	err       error
}

func (m *mockPublisher) Publish(_ context.Context, ev *domain.RawEvent) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, ev)
	return nil
}

type mockDedup struct {
	duplicates map[string]bool
}

func (m *mockDedup) IsDuplicate(_ context.Context, source, sourceEventID string) (bool, error) {
	key := source + ":" + sourceEventID
	return m.duplicates[key], nil
}

func TestCollectService_PublishesNewEvents(t *testing.T) {
	ev := &domain.RawEvent{
		EventID:       "test-1",
		Source:        domain.SourceTelegram,
		SourceEventID: "tg-100",
		Payload:       `{"text":"test"}`,
		Timestamp:     time.Now(),
	}
	adapter := &mockAdapter{source: domain.SourceTelegram, events: []*domain.RawEvent{ev}}
	pub := &mockPublisher{}
	dd := &mockDedup{duplicates: map[string]bool{}}

	svc := application.NewCollectService(zerolog.Nop(), pub, dd, []application.SourceAdapter{adapter}, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	svc.Run(ctx)

	if len(pub.published) != 1 {
		t.Errorf("expected 1 published event, got %d", len(pub.published))
	}
}

func TestCollectService_SkipsDuplicates(t *testing.T) {
	ev := &domain.RawEvent{
		EventID:       "test-2",
		Source:        domain.SourceTelegram,
		SourceEventID: "tg-200",
		Payload:       `{"text":"dup"}`,
		Timestamp:     time.Now(),
	}
	adapter := &mockAdapter{source: domain.SourceTelegram, events: []*domain.RawEvent{ev}}
	pub := &mockPublisher{}
	dd := &mockDedup{duplicates: map[string]bool{"telegram:tg-200": true}}

	svc := application.NewCollectService(zerolog.Nop(), pub, dd, []application.SourceAdapter{adapter}, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	svc.Run(ctx)

	if len(pub.published) != 0 {
		t.Errorf("expected 0 published events (duplicate), got %d", len(pub.published))
	}
}

func TestCollectService_HandlesAdapterError(t *testing.T) {
	adapter := &mockAdapter{source: domain.SourceRSS, err: errors.New("network error")}
	pub := &mockPublisher{}
	dd := &mockDedup{duplicates: map[string]bool{}}

	svc := application.NewCollectService(zerolog.Nop(), pub, dd, []application.SourceAdapter{adapter}, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	svc.Run(ctx) // should not panic
}
