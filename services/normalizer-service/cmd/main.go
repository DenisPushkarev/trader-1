package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
	"github.com/trader-1/trader-1/packages/shared/health"
	"github.com/trader-1/trader-1/packages/shared/logging"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/packages/shared/shutdown"
	"github.com/trader-1/trader-1/services/normalizer-service/internal/application"
	"github.com/trader-1/trader-1/services/normalizer-service/internal/infrastructure/enrichment"
	"github.com/trader-1/trader-1/services/normalizer-service/internal/infrastructure/messaging"

	natspkg "github.com/nats-io/nats.go"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("normalizer-service", getEnv("LOG_LEVEL", "info"))

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpAddr := getEnv("HTTP_ADDR", ":8081")

	nc, err := natsclient.NewClient(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect nats")
	}

	// Ensure input stream exists
	if err := nc.EnsureStream(&natspkg.StreamConfig{
		Name:      contractsv1.StreamEventsRaw,
		Subjects:  []string{contractsv1.SubjectEventsRaw},
		Retention: natspkg.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Storage:   natspkg.FileStorage,
	}); err != nil {
		logger.Warn().Err(err).Msg("ensure raw stream")
	}

	// Ensure output stream
	if err := nc.EnsureStream(&natspkg.StreamConfig{
		Name:      contractsv1.StreamEventsNormalized,
		Subjects:  []string{contractsv1.SubjectEventsNormalized},
		Retention: natspkg.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Storage:   natspkg.FileStorage,
	}); err != nil {
		logger.Warn().Err(err).Msg("ensure normalized stream")
	}

	rc, err := redis.NewClient(redisAddr, getEnv("REDIS_PASSWORD", ""), 0)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect redis")
	}

	sentimentAnalyzer := enrichment.NewSentimentAnalyzer()
	impactScorer := enrichment.NewImpactScorer()
	pub := messaging.NewNATSPublisher(nc, logger)
	handler := application.NewNormalizeHandler(logger, pub, rc, sentimentAnalyzer, impactScorer)

	consumerCfg := natsclient.ConsumerConfig{
		Stream:     contractsv1.StreamEventsRaw,
		Durable:    "normalizer-events-raw",
		Subject:    contractsv1.SubjectEventsRaw,
		MaxDeliver: 5,
		AckWait:    30 * time.Second,
	}

	healthHandler := health.NewHandler()
	r := chi.NewRouter()
	r.Get("/healthz", healthHandler.LivenessHandler())
	r.Get("/readyz", healthHandler.ReadinessHandler())

	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	coord := shutdown.NewCoordinator(logger)
	coord.Register("http", func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	})
	coord.Register("nats", func(_ context.Context) error {
		nc.Close()
		return nil
	})
	coord.Register("redis", func(_ context.Context) error {
		return rc.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	coord.Register("consumer", func(_ context.Context) error {
		cancel()
		return nil
	})

	go func() {
		logger.Info().Str("addr", httpAddr).Msg("HTTP server starting")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	healthHandler.SetReady(true)
	logger.Info().Msg("normalizer-service ready")

	go func() {
		if err := nc.ConsumeMessages(ctx, consumerCfg, handler.Handle); err != nil {
			logger.Error().Err(err).Msg("consume error")
		}
	}()

	coord.ListenAndShutdown(30 * time.Second)
}
