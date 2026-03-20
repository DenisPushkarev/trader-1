package domain

import (
	"sync"
	"testing"
	"time"
)

// fixedClock is a Clock implementation whose time can be advanced manually.
type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFixedClock(t time.Time) *fixedClock { return &fixedClock{now: t} }

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// --- no-op limiter (zero / negative maxPerMinute) ---

func TestNewSignalRateLimiter_Zero_AlwaysAllows(t *testing.T) {
	rl := NewSignalRateLimiter(0)
	for i := 0; i < 1000; i++ {
		if !rl.Allow() {
			t.Fatalf("expected Allow()=true on call %d with limit=0", i)
		}
	}
}

func TestNewSignalRateLimiter_Negative_AlwaysAllows(t *testing.T) {
	rl := NewSignalRateLimiter(-10)
	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Fatalf("expected Allow()=true on call %d with limit=-10", i)
		}
	}
}

// --- sliding window: basic allowance ---

func TestSlidingWindow_AllowsUpToLimit(t *testing.T) {
	clk := newFixedClock(time.Now())
	rl := NewSignalRateLimiterWithClock(5, clk)

	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Fatalf("expected Allow()=true on call %d (limit=5)", i+1)
		}
	}
}

func TestSlidingWindow_DropsOverLimit(t *testing.T) {
	clk := newFixedClock(time.Now())
	rl := NewSignalRateLimiterWithClock(3, clk)

	// Consume the full quota.
	for i := 0; i < 3; i++ {
		rl.Allow()
	}

	// Next call must be denied.
	if rl.Allow() {
		t.Fatal("expected Allow()=false after limit reached")
	}
}

func TestSlidingWindow_ExactlyAtLimit(t *testing.T) {
	clk := newFixedClock(time.Now())
	const limit = 10
	rl := NewSignalRateLimiterWithClock(limit, clk)

	allowed := 0
	for i := 0; i < limit+5; i++ {
		if rl.Allow() {
			allowed++
		}
	}
	if allowed != limit {
		t.Errorf("expected %d allowed signals, got %d", limit, allowed)
	}
}

// --- sliding window: window reset ---

func TestSlidingWindow_ResetsAfterMinute(t *testing.T) {
	clk := newFixedClock(time.Now())
	rl := NewSignalRateLimiterWithClock(3, clk)

	// Exhaust quota.
	for i := 0; i < 3; i++ {
		rl.Allow()
	}
	if rl.Allow() {
		t.Fatal("expected drop before window reset")
	}

	// Advance past the 1-minute window.
	clk.Advance(61 * time.Second)

	// Quota should be replenished.
	if !rl.Allow() {
		t.Fatal("expected Allow()=true after window reset")
	}
}

func TestSlidingWindow_PartialReset(t *testing.T) {
	base := time.Now()
	clk := newFixedClock(base)
	rl := NewSignalRateLimiterWithClock(3, clk)

	// Emit 2 signals at t=0.
	rl.Allow()
	rl.Allow()

	// Advance 61 seconds — first 2 timestamps fall outside window.
	clk.Advance(61 * time.Second)

	// Emit 1 more at t=61s (within fresh window).
	rl.Allow()

	// Now we should have capacity for 2 more (3 - 1 = 2).
	if !rl.Allow() {
		t.Fatal("expected Allow()=true: only 1 signal in window")
	}
	if !rl.Allow() {
		t.Fatal("expected Allow()=true: only 2 signals in window")
	}
	// Third in the new window should be denied (limit=3 and we have 3).
	if rl.Allow() {
		t.Fatal("expected Allow()=false: 3 signals already in window")
	}
}

// --- concurrency safety ---

func TestSlidingWindow_ConcurrentAccess(t *testing.T) {
	clk := newFixedClock(time.Now())
	const limit = 50
	rl := NewSignalRateLimiterWithClock(limit, clk)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed > limit {
		t.Errorf("concurrent: allowed %d signals, expected <= %d", allowed, limit)
	}
}
