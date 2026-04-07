package middleware

import (
	"context"

	"db-mcp/internal/config"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter  *rate.Limiter
	enabled  bool
	requests int
	burst    int
}

func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	if !cfg.Enabled {
		return &RateLimiter{enabled: false}
	}

	return &RateLimiter{
		enabled:  true,
		limiter:  rate.NewLimiter(rate.Limit(cfg.Requests), cfg.Burst),
		requests: cfg.Requests,
		burst:    cfg.Burst,
	}
}

func (r *RateLimiter) Allow() bool {
	if !r.enabled {
		return true
	}
	return r.limiter.Allow()
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	if !r.enabled {
		return nil
	}
	return r.limiter.Wait(ctx)
}

func (r *RateLimiter) Enabled() bool {
	return r.enabled
}

func (r *RateLimiter) Requests() int {
	return r.requests
}

func (r *RateLimiter) Burst() int {
	return r.burst
}
