package enrichment

import "strings"

// SentimentAnalyzer provides deterministic stub sentiment analysis.
type SentimentAnalyzer struct{}

// NewSentimentAnalyzer creates a SentimentAnalyzer.
func NewSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{}
}

var bullishKeywords = []string{"bullish", "major", "milestone", "partnership", "listing", "record", "strong", "incredible", "surge", "moon", "ATH"}
var bearishKeywords = []string{"bearish", "crash", "dump", "scam", "hack", "fraud", "decline", "drop", "warning", "sell"}

// Analyze returns a sentiment score in [-1, 1]. Deterministic for the same input.
func (a *SentimentAnalyzer) Analyze(text string) float64 {
	lower := strings.ToLower(text)
	score := 0.0
	for _, kw := range bullishKeywords {
		if strings.Contains(lower, kw) {
			score += 0.15
		}
	}
	for _, kw := range bearishKeywords {
		if strings.Contains(lower, kw) {
			score -= 0.15
		}
	}
	if score > 1.0 {
		return 1.0
	}
	if score < -1.0 {
		return -1.0
	}
	return score
}
