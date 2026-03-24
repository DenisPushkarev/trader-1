package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	natspkg "github.com/nats-io/nats.go"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/packages/shared/health"
	"github.com/trader-1/trader-1/packages/shared/logging"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/packages/shared/shutdown"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/application"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/infrastructure/cache"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/infrastructure/handler"
	"github.com/trader-1/trader-1/services/api-gateway-service/internal/infrastructure/subscriber"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("api-gateway-service", getEnv("LOG_LEVEL", "info"))

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpAddr := getEnv("HTTP_ADDR", ":8080")

	nc, err := natsclient.NewClient(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect nats")
	}

	for _, cfg := range []struct{ name, subject string }{
		{contractsv1.StreamSignalsExplained, contractsv1.SubjectSignalsExplained},
		{contractsv1.StreamEventsNormalized, contractsv1.SubjectEventsNormalized},
	} {
		if err := nc.EnsureStream(&natspkg.StreamConfig{
			Name: cfg.name, Subjects: []string{cfg.subject},
			Retention: natspkg.LimitsPolicy, MaxAge: 7 * 24 * time.Hour,
			Storage: natspkg.FileStorage,
		}); err != nil {
			logger.Warn().Err(err).Str("stream", cfg.name).Msg("ensure stream")
		}
	}

	rc, err := redis.NewClient(redisAddr, getEnv("REDIS_PASSWORD", ""), 0)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect redis")
	}

	signalCache := cache.NewSignalCache(rc, logger)
	eventCache := cache.NewEventCache(rc, logger)
	queryService := application.NewQueryService(signalCache, eventCache, logger)

	sub := subscriber.NewNATSSubscriber(nc, signalCache, eventCache, logger)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	healthHandler := health.NewHandler()
	r.Get("/healthz", healthHandler.LivenessHandler())
	r.Get("/readyz", healthHandler.ReadinessHandler())

	apiHandler := handler.NewAPIHandler(queryService, logger)
	r.Route("/signals", func(r chi.Router) {
		r.Get("/latest", apiHandler.GetLatestSignals)
		r.Get("/history", apiHandler.GetSignalHistory)
	})
	r.Get("/events", apiHandler.GetEvents)
	r.Post("/simulate", apiHandler.PostSimulate)

	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	coord := shutdown.NewCoordinator(logger)
	coord.Register("http", func(ctx context.Context) error { return httpServer.Shutdown(ctx) })
	coord.Register("nats", func(_ context.Context) error { nc.Close(); return nil })
	coord.Register("redis", func(_ context.Context) error { return rc.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	coord.Register("subscribers", func(_ context.Context) error { cancel(); return nil })

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()
	healthHandler.SetReady(true)
	logger.Info().Str("addr", httpAddr).Msg("api-gateway-service ready")

	go sub.Start(ctx)

	coord.ListenAndShutdown(30 * time.Second)
}
