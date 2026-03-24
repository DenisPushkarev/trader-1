package application

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/domain"
)

// SignalStore provides signal read access.
type SignalStore interface {
	GetLatest(ctx context.Context, limit int) ([]*domain.SignalReadModel, error)
	GetHistory(ctx context.Context, page, pageSize int) ([]*domain.SignalReadModel, error)
}

// EventStore provides event read access.
type EventStore interface {
	GetLatest(ctx context.Context, limit int) ([]*domain.EventReadModel, error)
}

// QueryService provides read-side query logic for the API.
type QueryService struct {
	signals SignalStore
	events  EventStore
	logger  zerolog.Logger
}

// NewQueryService creates a QueryService.
func NewQueryService(signals SignalStore, events EventStore, logger zerolog.Logger) *QueryService {
	return &QueryService{signals: signals, events: events, logger: logger}
}

// GetLatestSignals returns the most recent signals.
func (q *QueryService) GetLatestSignals(ctx context.Context, limit int) ([]*domain.SignalReadModel, error) {
	return q.signals.GetLatest(ctx, limit)
}

// GetSignalHistory returns paginated signal history.
func (q *QueryService) GetSignalHistory(ctx context.Context, page, pageSize int) ([]*domain.SignalReadModel, error) {
	return q.signals.GetHistory(ctx, page, pageSize)
}

// GetEvents returns the most recent normalized events.
func (q *QueryService) GetEvents(ctx context.Context, limit int) ([]*domain.EventReadModel, error) {
	return q.events.GetLatest(ctx, limit)
}
