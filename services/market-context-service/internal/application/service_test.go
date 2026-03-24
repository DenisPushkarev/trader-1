package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/market-context-service/internal/application"
	"github.com/trader-1/trader-1/services/market-context-service/internal/domain"
)

type mockProvider struct {
	snap *domain.MarketContextSnapshot
	err  error
}

func (m *mockProvider) FetchSnapshot(_ context.Context, asset string) (*domain.MarketContextSnapshot, error) {
	if m.snap != nil {
		m.snap.Asset = asset
	}
	return m.snap, m.err
}

type mockPublisher struct {
	published []*domain.MarketContextSnapshot
}

func (m *mockPublisher) Publish(_ context.Context, snap *domain.MarketContextSnapshot) error {
	m.published = append(m.published, snap)
	return nil
}

type mockCache struct {
	stored []*domain.MarketContextSnapshot
}

func (m *mockCache) Set(_ context.Context, snap *domain.MarketContextSnapshot) error {
	m.stored = append(m.stored, snap)
	return nil
}

func (m *mockCache) Get(_ context.Context, _ string) (*domain.MarketContextSnapshot, error) {
	if len(m.stored) == 0 {
		return nil, nil
	}
	return m.stored[len(m.stored)-1], nil
}

func TestUpdateContextService_PublishesSnapshot(t *testing.T) {
	snap := &domain.MarketContextSnapshot{
		Asset:      "TON/USDT",
		Price:      2.5,
		Volume24H:  1_000_000,
		Volatility: 0.03,
		Timestamp:  time.Now(),
		Indicators: map[string]float64{"rsi": 55.0},
	}
	provider := &mockProvider{snap: snap}
	pub := &mockPublisher{}
	c := &mockCache{}

	svc := application.NewUpdateContextService(zerolog.Nop(), provider, pub, c, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	svc.Run(ctx)

	if len(pub.published) == 0 {
		t.Error("expected at least one published snapshot")
	}
	if pub.published[0].Asset != "TON/USDT" {
		t.Errorf("expected asset TON/USDT, got %s", pub.published[0].Asset)
	}
	if pub.published[0].ContextID == "" {
		t.Error("context_id should be set")
	}
}
