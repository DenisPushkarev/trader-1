package domain

import "time"

// NormalizedEvent is the enriched canonical form of an external event.
type NormalizedEvent struct {
	EventID           string
	SourceRef         SourceRef
	EventType         string
	Asset             string
	Sentiment         float64
	Impact            float64
	Content           string
	Timestamp         time.Time
	EnrichmentVersion string
	Metadata          map[string]string
}

// SourceRef identifies the original source event.
type SourceRef struct {
	Source        string
	SourceEventID string
}
