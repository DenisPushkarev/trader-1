package application

import (
	"context"

	"go.uber.org/zap"

	"signal-engine-service/internal/domain"
)

// SignalPublisher controls signal emission to the signals.generated subject,
// enforcing the configured rate limit before delegating to the underlying
// publish function.
type SignalPublisher struct {
	rateLimiter domain.RateLimiter
	logger      *zap.Logger
}

// NewSignalPublisher creates a SignalPublisher.
// rateLimiter must not be nil; use domain.NewSignalRateLimiter(0) to disable rate limiting.
func NewSignalPublisher(rateLimiter domain.RateLimiter, logger *zap.Logger) *SignalPublisher {
	return &SignalPublisher{
		rateLimiter: rateLimiter,
		logger:      logger,
	}
}

// Publish emits a signal by calling publishFn if the rate limit allows.
// Returns (true, nil) on successful emission, (false, nil) when the signal is
// dropped due to rate limiting, or (false, err) on a publish error.
//
// eventID is used for log correlation when a signal is dropped.
func (p *SignalPublisher) Publish(ctx context.Context, eventID string, publishFn func() error) (bool, error) {
	if !p.rateLimiter.Allow() {
		p.logger.Debug("signal dropped: rate limit exceeded",
			zap.String("event_id", eventID),
		)
		return false, nil
	}

	if err := publishFn(); err != nil {
		return false, err
	}

	return true, nil
}
