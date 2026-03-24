package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	natspkg "github.com/nats-io/nats.go"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/packages/shared/health"
	"github.com/trader-1/trader-1/packages/shared/logging"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/packages/shared/shutdown"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/application"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/infrastructure/messaging"
	"github.com/trader-1/trader-1/services/signal-engine-service/internal/infrastructure/scoring"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("signal-engine-service", getEnv("LOG_LEVEL", "info"))

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpAddr := getEnv("HTTP_ADDR", ":8083")

	nc, err := natsclient.NewClient(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect nats")
	}

	for _, cfg := range []struct{ name, subject string }{
		{contractsv1.StreamEventsNormalized, contractsv1.SubjectEventsNormalized},
		{contractsv1.StreamMarketContext, contractsv1.SubjectMarketContextUpdated},
		{contractsv1.StreamSignalsGenerated, contractsv1.SubjectSignalsGenerated},
	} {
		if err := nc.EnsureStream(&natspkg.StreamConfig{
			Name:      cfg.name,
			Subjects:  []string{cfg.subject},
			Retention: natspkg.LimitsPolicy,
			MaxAge:    7 * 24 * time.Hour,
			Storage:   natspkg.FileStorage,
		}); err != nil {
			logger.Warn().Err(err).Str("stream", cfg.name).Msg("ensure stream")
		}
	}

	rc, err := redis.NewClient(redisAddr, getEnv("REDIS_PASSWORD", ""), 0)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect redis")
	}

	scoringModel := scoring.NewDefaultModel()
	pub := messaging.NewNATSPublisher(nc, logger)

	handler := application.NewSignalHandler(logger, pub, rc, scoringModel)

	healthHandler := health.NewHandler()
	r := chi.NewRouter()
	r.Get("/healthz", healthHandler.LivenessHandler())
	r.Get("/readyz", healthHandler.ReadinessHandler())

	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	coord := shutdown.NewCoordinator(logger)
	coord.Register("http", func(ctx context.Context) error { return httpServer.Shutdown(ctx) })
	coord.Register("nats", func(_ context.Context) error { nc.Close(); return nil })
	coord.Register("redis", func(_ context.Context) error { return rc.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	coord.Register("consumers", func(_ context.Context) error { cancel(); return nil })

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	healthHandler.SetReady(true)
	logger.Info().Msg("signal-engine-service ready")

	// Consume market context updates to keep in-memory cache
	go func() {
		cfg := natsclient.ConsumerConfig{
			Stream:  contractsv1.StreamMarketContext,
			Durable: "signal-engine-market-context",
			Subject: contractsv1.SubjectMarketContextUpdated,
		}
		if err := nc.ConsumeMessages(ctx, cfg, handler.HandleMarketContext); err != nil {
			logger.Error().Err(err).Msg("market context consumer error")
		}
	}()

	// Consume normalized events
	go func() {
		cfg := natsclient.ConsumerConfig{
			Stream:  contractsv1.StreamEventsNormalized,
			Durable: "signal-engine-events-normalized",
			Subject: contractsv1.SubjectEventsNormalized,
		}
		if err := nc.ConsumeMessages(ctx, cfg, handler.HandleNormalizedEvent); err != nil {
			logger.Error().Err(err).Msg("normalized event consumer error")
		}
	}()

	coord.ListenAndShutdown(30 * time.Second)
}
