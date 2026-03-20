package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/domain"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/infrastructure/handler"
)

type mockQuery struct {
	signals []*domain.SignalReadModel
	events  []*domain.EventReadModel
}

func (m *mockQuery) GetLatestSignals(_ context.Context, _ int) ([]*domain.SignalReadModel, error) {
	return m.signals, nil
}
func (m *mockQuery) GetSignalHistory(_ context.Context, _, _ int) ([]*domain.SignalReadModel, error) {
	return m.signals, nil
}
func (m *mockQuery) GetEvents(_ context.Context, _ int) ([]*domain.EventReadModel, error) {
	return m.events, nil
}

func TestGetLatestSignals(t *testing.T) {
	q := &mockQuery{signals: []*domain.SignalReadModel{{SignalID: "sig-1", Direction: "BULLISH"}}}
	h := handler.NewAPIHandler(q, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/signals/latest", nil)
	w := httptest.NewRecorder()
	h.GetLatestSignals(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["count"].(float64) != 1 {
		t.Errorf("expected count=1")
	}
}

func TestGetEvents(t *testing.T) {
	q := &mockQuery{events: []*domain.EventReadModel{{EventID: "evt-1"}}}
	h := handler.NewAPIHandler(q, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPostSimulate(t *testing.T) {
	q := &mockQuery{}
	h := handler.NewAPIHandler(q, zerolog.Nop())

	body := `{"scenario":"bullish_run","events":10}`
	req := httptest.NewRequest(http.MethodPost, "/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.PostSimulate(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}
