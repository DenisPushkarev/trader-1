package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// NewLogger creates a structured zerolog logger for the given service.
func NewLogger(service, level string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var lvl zerolog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = zerolog.DebugLevel
	case "warn", "warning":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	default:
		lvl = zerolog.InfoLevel
	}

	var w io.Writer = os.Stdout
	// Pretty-print in development if LOG_FORMAT=console
	if os.Getenv("LOG_FORMAT") == "console" {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	return zerolog.New(w).
		Level(lvl).
		With().
		Timestamp().
		Str("service", service).
		Logger()
}

// WithCorrelationID adds a correlation ID field to the logger.
func WithCorrelationID(logger zerolog.Logger, id string) zerolog.Logger {
	return logger.With().Str("correlation_id", id).Logger()
}

// WithEventID adds an event ID field to the logger.
func WithEventID(logger zerolog.Logger, id string) zerolog.Logger {
	return logger.With().Str("event_id", id).Logger()
}
