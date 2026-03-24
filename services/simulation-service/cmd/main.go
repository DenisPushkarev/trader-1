package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trader-1/trader-1/packages/shared/health"
	"github.com/trader-1/trader-1/packages/shared/logging"
	"github.com/trader-1/trader-1/packages/shared/shutdown"
	"github.com/trader-1/trader-1/services/simulation-service/internal/application"
	"github.com/trader-1/trader-1/services/simulation-service/internal/infrastructure/handler"
	"github.com/trader-1/trader-1/services/simulation-service/internal/infrastructure/scenarios"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("simulation-service", getEnv("LOG_LEVEL", "info"))
	httpAddr := getEnv("HTTP_ADDR", ":8087")

	registry := scenarios.NewRegistry()
	registry.Register(scenarios.BullishRunScenario())
	registry.Register(scenarios.BearishRunScenario())
	registry.Register(scenarios.FakeHypeScenario())
	registry.Register(scenarios.ConflictingSignalsScenario())

	runner := application.NewSimulationRunner(logger, registry)

	r := chi.NewRouter()
	healthHandler := health.NewHandler()
	r.Get("/healthz", healthHandler.LivenessHandler())
	r.Get("/readyz", healthHandler.ReadinessHandler())

	simHandler := handler.NewSimulationHandler(runner, logger)
	r.Post("/run", simHandler.RunSimulation)
	r.Get("/scenarios", simHandler.ListScenarios)

	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	coord := shutdown.NewCoordinator(logger)
	coord.Register("http", func(ctx context.Context) error { return httpServer.Shutdown(ctx) })

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	healthHandler.SetReady(true)
	logger.Info().Str("addr", httpAddr).Msg("simulation-service ready")

	coord.ListenAndShutdown(30 * time.Second)
}
