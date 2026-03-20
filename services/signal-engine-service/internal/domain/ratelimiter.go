package domain

import (
	"sync"
	"time"
)

// RateLimiter controls whether a signal emission is allowed.
// Implementations must be safe for concurrent use.
type RateLimiter interface {
	Allow() bool
}

// Clock abstracts wall-clock time to allow deterministic testing.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// noopLimiter always permits (used when MaxSignalsPerMinute == 0).
type noopLimiter struct{}

func (noopLimiter) Allow() bool { return true }

// slidingWindowLimiter counts signal timestamps within the last minute.
// It is safe for concurrent use.
type slidingWindowLimiter struct {
	mu           sync.Mutex
	maxPerMinute int
	clock        Clock
	timestamps   []time.Time
}

// NewSignalRateLimiter creates a RateLimiter that allows at most maxPerMinute
// signals per 60-second sliding window.
//
// If maxPerMinute <= 0, a no-op limiter is returned that always allows.
func NewSignalRateLimiter(maxPerMinute int) RateLimiter {
	return NewSignalRateLimiterWithClock(maxPerMinute, realClock{})
}

// NewSignalRateLimiterWithClock is like NewSignalRateLimiter but accepts an
// injectable Clock for deterministic testing.
func NewSignalRateLimiterWithClock(maxPerMinute int, clock Clock) RateLimiter {
	if maxPerMinute <= 0 {
		return noopLimiter{}
	}
	return &slidingWindowLimiter{
		maxPerMinute: maxPerMinute,
		clock:        clock,
		timestamps:   make([]time.Time, 0, maxPerMinute),
	}
}

// Allow reports whether a signal may be emitted.
// It returns false when the sliding-window count has reached maxPerMinute.
func (l *slidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	windowStart := now.Add(-time.Minute)

	// Evict timestamps outside the 1-minute window (re-use backing array).
	valid := l.timestamps[:0]
	for _, t := range l.timestamps {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	l.timestamps = valid

	if len(l.timestamps) >= l.maxPerMinute {
		return false
	}

	l.timestamps = append(l.timestamps, now)
	return true
}
