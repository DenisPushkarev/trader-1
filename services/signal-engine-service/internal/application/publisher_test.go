package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"signal-engine-service/internal/domain"
)

// countingLimiter is a test double that denies after max calls.
type countingLimiter struct {
	max  int
	seen int
}

func (l *countingLimiter) Allow() bool {
	if l.seen >= l.max {
		return false
	}
	l.seen++
	return true
}

// fixedClock returns a constant time, satisfying domain.Clock.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func nopLogger() *zap.Logger { return zap.NewNop() }

// --- rate limiting integration ---

func TestPublish_DropsWhenRateLimitExceeded(t *testing.T) {
	limiter := &countingLimiter{max: 2}
	pub := NewSignalPublisher(limiter, nopLogger())

	ctx := context.Background()
	called := 0
	fn := func() error { called++; return nil }

	// First two calls should succeed.
	for i := 0; i < 2; i++ {
		ok, err := pub.Publish(ctx, "evt-1", fn)
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i+1, err)
		}
		if !ok {
			t.Fatalf("expected emission on call %d", i+1)
		}
	}

	// Third call should be dropped.
	ok, err := pub.Publish(ctx, "evt-1", fn)
	if err != nil {
		t.Fatalf("unexpected error on dropped call: %v", err)
	}
	if ok {
		t.Fatal("expected Publish to return false when rate limited")
	}
	if called != 2 {
		t.Errorf("expected publishFn called 2 times, got %d", called)
	}
}

func TestPublish_NoRateLimit_AllowsAll(t *testing.T) {
	rl := domain.NewSignalRateLimiter(0) // disabled
	pub := NewSignalPublisher(rl, nopLogger())

	ctx := context.Background()
	called := 0
	fn := func() error { called++; return nil }

	const n = 200
	for i := 0; i < n; i++ {
		ok, err := pub.Publish(ctx, "evt-x", fn)
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i+1, err)
		}
		if !ok {
			t.Fatalf("expected emission on call %d with limit=0", i+1)
		}
	}
	if called != n {
		t.Errorf("expected publishFn called %d times, got %d", n, called)
	}
}

func TestPublish_PropagatesPublishError(t *testing.T) {
	rl := domain.NewSignalRateLimiter(0)
	pub := NewSignalPublisher(rl, nopLogger())

	want := errors.New("nats unavailable")
	ok, err := pub.Publish(context.Background(), "evt-err", func() error { return want })
	if ok {
		t.Fatal("expected ok=false on publish error")
	}
	if !errors.Is(err, want) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestPublish_WithDomainLimiter_DropsExcess(t *testing.T) {
	clk := fixedClock{t: time.Now()}
	rl := domain.NewSignalRateLimiterWithClock(3, clk)
	pub := NewSignalPublisher(rl, nopLogger())

	ctx := context.Background()
	allowed := 0
	fn := func() error { allowed++; return nil }

	for i := 0; i < 6; i++ {
		pub.Publish(ctx, "evt-2", fn) //nolint:errcheck
	}

	if allowed != 3 {
		t.Errorf("expected 3 allowed signals with limit=3, got %d", allowed)
	}
}
