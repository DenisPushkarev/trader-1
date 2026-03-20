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
	"github.com/trader-1/trader-1/services/explainability-service/internal/application"
	"github.com/trader-1/trader-1/services/explainability-service/internal/infrastructure/generator"
	"github.com/trader-1/trader-1/services/explainability-service/internal/infrastructure/messaging"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("explainability-service", getEnv("LOG_LEVEL", "info"))

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpAddr := getEnv("HTTP_ADDR", ":8085")

	nc, err := natsclient.NewClient(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect nats")
	}

	for _, cfg := range []struct{ name, subject string }{
		{contractsv1.StreamSignalsRiskAdjusted, contractsv1.SubjectSignalsRiskAdjusted},
		{contractsv1.StreamSignalsExplained, contractsv1.SubjectSignalsExplained},
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

	gen := generator.NewTemplateGenerator()
	pub := messaging.NewNATSPublisher(nc, logger)
	handler := application.NewExplainHandler(logger, pub, rc, gen)

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
	coord.Register("consumer", func(_ context.Context) error { cancel(); return nil })

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()
	healthHandler.SetReady(true)
	logger.Info().Msg("explainability-service ready")

	go func() {
		cfg := natsclient.ConsumerConfig{
			Stream: contractsv1.StreamSignalsRiskAdjusted, Durable: "explain-signals-risk-adjusted",
			Subject: contractsv1.SubjectSignalsRiskAdjusted, MaxDeliver: 5,
		}
		if err := nc.ConsumeMessages(ctx, cfg, handler.Handle); err != nil {
			logger.Error().Err(err).Msg("consume error")
		}
	}()

	coord.ListenAndShutdown(30 * time.Second)
}
