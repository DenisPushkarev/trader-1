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
	"github.com/trader-1/trader-1/services/market-context-service/internal/application"
	"github.com/trader-1/trader-1/services/market-context-service/internal/infrastructure/cache"
	"github.com/trader-1/trader-1/services/market-context-service/internal/infrastructure/messaging"
	"github.com/trader-1/trader-1/services/market-context-service/internal/infrastructure/provider"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger := logging.NewLogger("market-context-service", getEnv("LOG_LEVEL", "info"))

	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	httpAddr := getEnv("HTTP_ADDR", ":8082")
	updateInterval := 60 * time.Second

	nc, err := natsclient.NewClient(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect nats")
	}

	if err := nc.EnsureStream(&natspkg.StreamConfig{
		Name:      contractsv1.StreamMarketContext,
		Subjects:  []string{contractsv1.SubjectMarketContextUpdated},
		Retention: natspkg.LimitsPolicy,
		MaxAge:    30 * 24 * time.Hour,
		Storage:   natspkg.FileStorage,
	}); err != nil {
		logger.Warn().Err(err).Msg("ensure market context stream")
	}

	rc, err := redis.NewClient(redisAddr, getEnv("REDIS_PASSWORD", ""), 0)
	if err != nil {
		logger.Fatal().Err(err).Msg("connect redis")
	}

	priceProvider := provider.NewStubPriceProvider()
	pub := messaging.NewNATSPublisher(nc, logger)
	contextCache := cache.NewRedisCache(rc, logger)

	svc := application.NewUpdateContextService(logger, priceProvider, pub, contextCache, updateInterval)

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
	coord.Register("service", func(_ context.Context) error { cancel(); return nil })

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	healthHandler.SetReady(true)
	logger.Info().Msg("market-context-service ready")

	go svc.Run(ctx)

	coord.ListenAndShutdown(30 * time.Second)
}
