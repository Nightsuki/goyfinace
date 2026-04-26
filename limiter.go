package yfinance

import (
	"context"
	"sync"
	"time"
)

// RateLimiter is a minimal token-bucket limiter satisfying Limiter. It
// produces tokens at a steady rate and supports a burst capacity. The zero
// value is unusable; use NewRateLimiter.
type RateLimiter struct {
	mu        sync.Mutex
	tokens    float64
	burst     float64
	rate      float64 // tokens per second
	last      time.Time
	now       func() time.Time
	sleepImpl func(ctx context.Context, d time.Duration) error
}

// NewRateLimiter returns a limiter that allows requests at requestsPerSecond
// with the given burst size (minimum 1). A zero or negative rate disables
// rate limiting and Wait returns immediately.
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	now := time.Now
	return &RateLimiter{
		tokens: float64(burst),
		burst:  float64(burst),
		rate:   requestsPerSecond,
		last:   now(),
		now:    now,
		sleepImpl: func(ctx context.Context, d time.Duration) error {
			if d <= 0 {
				return nil
			}
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		},
	}
}

// Wait blocks until a token is available or ctx is cancelled. When the
// limiter's rate is zero or negative, Wait returns immediately.
func (r *RateLimiter) Wait(ctx context.Context) error {
	if r == nil || r.rate <= 0 {
		return nil
	}
	r.mu.Lock()
	now := r.now()
	elapsed := now.Sub(r.last).Seconds()
	r.tokens += elapsed * r.rate
	if r.tokens > r.burst {
		r.tokens = r.burst
	}
	r.last = now
	if r.tokens >= 1 {
		r.tokens--
		r.mu.Unlock()
		return nil
	}
	deficit := 1 - r.tokens
	wait := time.Duration(deficit / r.rate * float64(time.Second))
	r.tokens = 0
	r.last = now.Add(wait)
	r.mu.Unlock()
	return r.sleepImpl(ctx, wait)
}
