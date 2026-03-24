package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

// Hook is a named shutdown function.
type Hook struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Coordinator orchestrates graceful shutdown on OS signals.
type Coordinator struct {
	hooks  []Hook
	mu     sync.Mutex
	logger zerolog.Logger
	done   chan struct{}
}

// NewCoordinator creates a new Coordinator that listens for SIGTERM/SIGINT.
func NewCoordinator(logger zerolog.Logger) *Coordinator {
	return &Coordinator{
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Register adds a named shutdown hook. Hooks run in reverse registration order.
func (c *Coordinator) Register(name string, fn func(ctx context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks = append(c.hooks, Hook{Name: name, Fn: fn})
}

// Wait returns a channel that closes when shutdown begins.
func (c *Coordinator) Wait() <-chan struct{} {
	return c.done
}

// ListenAndShutdown blocks until a signal is received, then runs all shutdown hooks.
// It returns after all hooks complete or the timeout expires.
func (c *Coordinator) ListenAndShutdown(timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	c.logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	close(c.done)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c.mu.Lock()
	hooks := make([]Hook, len(c.hooks))
	copy(hooks, c.hooks)
	c.mu.Unlock()

	// Run hooks in reverse registration order
	for i := len(hooks) - 1; i >= 0; i-- {
		h := hooks[i]
		c.logger.Info().Str("hook", h.Name).Msg("running shutdown hook")
		if err := h.Fn(ctx); err != nil {
			c.logger.Error().Err(err).Str("hook", h.Name).Msg("shutdown hook error")
		}
	}
	c.logger.Info().Msg("shutdown complete")
}
