package ratelimit

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// Limiter wraps a rate limiter for controlling event throughput
type Limiter struct {
	limiter *rate.Limiter
	enabled bool
}

// NewLimiter creates a new rate limiter
func NewLimiter(eventsPerSecond int) *Limiter {
	if eventsPerSecond <= 0 {
		return &Limiter{enabled: false}
	}

	// Create token bucket rate limiter
	// Allow burst of 2x the rate to handle bursty workloads
	limiter := rate.NewLimiter(rate.Limit(eventsPerSecond), eventsPerSecond*2)

	return &Limiter{
		limiter: limiter,
		enabled: true,
	}
}

// Wait waits for permission to send n events
func (l *Limiter) Wait(ctx context.Context, n int) error {
	if !l.enabled {
		return nil
	}

	// Reserve tokens for n events
	reservation := l.limiter.ReserveN(time.Now(), n)
	if !reservation.OK() {
		return nil // Skip if reservation fails
	}

	// Wait for the required delay
	delay := reservation.Delay()
	if delay > 0 {
		select {
		case <-time.After(delay):
			return nil
		case <-ctx.Done():
			reservation.Cancel()
			return ctx.Err()
		}
	}

	return nil
}

// WaitOne waits for permission to send one event
func (l *Limiter) WaitOne(ctx context.Context) error {
	return l.Wait(ctx, 1)
}
