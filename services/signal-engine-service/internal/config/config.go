package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// Config holds signal-engine-service configuration.
type Config struct {
	// NATSUrl is the NATS server connection URL.
	NATSUrl string

	// MaxSignalsPerMinute controls the maximum number of signals emitted per minute.
	// Set to 0 (default) to disable rate limiting.
	//
	// NOTE: For deterministic replay/backfill operations, this must be set to 0.
	// A non-zero value may cause replay to produce different outputs than the
	// original run, violating the platform invariant of deterministic replay.
	MaxSignalsPerMinute int
}

// Load reads configuration from environment variables, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		NATSUrl:             getEnv("NATS_URL", "nats://localhost:4222"),
		MaxSignalsPerMinute: 0,
	}

	if v := os.Getenv("SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SIGNAL_ENGINE_MAX_SIGNALS_PER_MINUTE %q: %w", v, err)
		}
		cfg.MaxSignalsPerMinute = n
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all configuration values are acceptable.
func (c *Config) Validate() error {
	if c.MaxSignalsPerMinute < 0 {
		return errors.New("MaxSignalsPerMinute must be >= 0")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
