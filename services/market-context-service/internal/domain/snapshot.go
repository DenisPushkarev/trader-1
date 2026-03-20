package domain

import "time"

// MarketContextSnapshot represents a point-in-time market state for an asset.
type MarketContextSnapshot struct {
	ContextID  string
	Asset      string
	Price      float64
	Volume24H  float64
	Volatility float64
	Timestamp  time.Time
	Indicators map[string]float64
}
