package domain

import "time"

// Source is the external data source type.
type Source string

const (
	SourceTelegram Source = "telegram"
	SourceTwitter  Source = "twitter"
	SourceRSS      Source = "rss"
	SourceExchange Source = "exchange"
	SourceOnChain  Source = "onchain"
	SourceMiniApp  Source = "miniapp"
)

// RawEvent represents a raw external event before normalization.
type RawEvent struct {
	EventID       string
	Source        Source
	SourceEventID string
	Payload       string
	Timestamp     time.Time
	Metadata      map[string]string
}
