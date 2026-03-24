package provider

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/trader-1/trader-1/services/market-context-service/internal/domain"
)

// StubPriceProvider returns deterministic-ish test market data.
type StubPriceProvider struct {
	rng *rand.Rand
}

// NewStubPriceProvider creates a StubPriceProvider.
func NewStubPriceProvider() *StubPriceProvider {
	return &StubPriceProvider{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// FetchSnapshot returns a stubbed market context for the given asset.
func (p *StubPriceProvider) FetchSnapshot(_ context.Context, asset string) (*domain.MarketContextSnapshot, error) {
	basePrice := 2.45
	noise := (p.rng.Float64() - 0.5) * 0.1
	price := math.Round((basePrice+noise)*1000) / 1000

	return &domain.MarketContextSnapshot{
		Asset:      asset,
		Price:      price,
		Volume24H:  1_250_000 + p.rng.Float64()*500_000,
		Volatility: 0.02 + p.rng.Float64()*0.05,
		Timestamp:  time.Now(),
		Indicators: map[string]float64{
			"rsi":   45 + p.rng.Float64()*20,
			"macd":  (p.rng.Float64() - 0.5) * 0.1,
			"ema20": price * (1 + (p.rng.Float64()-0.5)*0.02),
		},
	}, nil
}
