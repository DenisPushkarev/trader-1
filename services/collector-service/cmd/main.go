package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trader-1/trader-1/packages/shared/health"
	"github.com/trader-1/trader-1/packages/shared/logging"
	natsclient "github.com/trader-1/trader-1/packages/shared/nats"
	"github.com/trader-1/trader-1/packages/shared/redis"
	"github.com/trader-1/trader-1/packages/shared/shutdown"
	"github.com/trader-1/trader-1/services/collector-service/internal/application"
	"github.com/trader-1/trader-1/services/collector-service/internal/infrastructure/adapters"
	redisdedup "github.com/trader-1/trader-1/services/collector-service/internal/infrastructure/dedup"
	"github.com/trader-1/trader-1/services/collector-service/internal/infrastructure/publisher"

	natspkg "github.com/nats-io/nats.go"
	contractsv1 "github.com/trader-1/trader-1/packages/contracts/gen/go/v1"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("collector-service", getEnv("LOG_LEVEL", "info"))

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpAddr := getEnv("HTTP_ADDR", ":8080")
	pollInterval := 30 * time.Second
	if v := getEnv("POLL_INTERVAL_SECONDS", ""); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil {
			pollInterval = d
		}
	}

	// NATS
	nc, err := natsclient.NewClient(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect nats")
	}

	// Ensure JetStream stream
	if err := nc.EnsureStream(&natspkg.StreamConfig{
		Name:      contractsv1.StreamEventsRaw,
		Subjects:  []string{contractsv1.SubjectEventsRaw},
		Retention: natspkg.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		MaxMsgs:   1_000_000,
		Storage:   natspkg.FileStorage,
	}); err != nil {
		logger.Warn().Err(err).Msg("ensure stream (may already exist)")
	}

	// Redis
	rc, err := redis.NewClient(redisAddr, getEnv("REDIS_PASSWORD", ""), 0)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect redis")
	}

	// Dependencies
	pub := publisher.NewNATSPublisher(nc, logger)
	dd := redisdedup.NewCollectorDedup(rc)

	srcAdapters := []application.SourceAdapter{
		adapters.NewTelegramAdapter(logger),
		adapters.NewTwitterAdapter(logger),
		adapters.NewRSSAdapter(logger),
	}

	svc := application.NewCollectService(logger, pub, dd, srcAdapters, pollInterval)

	// Health
	healthHandler := health.NewHandler()
	r := chi.NewRouter()
	r.Get("/healthz", healthHandler.LivenessHandler())
	r.Get("/readyz", healthHandler.ReadinessHandler())

	httpServer := &http.Server{Addr: httpAddr, Handler: r}

	// Shutdown
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
	coord.Register("collector", func(_ context.Context) error {
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
	logger.Info().Msg("collector-service ready")

	go svc.Run(ctx)

	coord.ListenAndShutdown(30 * time.Second)
}
