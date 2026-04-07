package middleware_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/config"
	"db-mcp/internal/middleware"
)

func TestNewRateLimiter_Enabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  true,
		Requests: 100,
		Burst:    200,
	}

	limiter := middleware.NewRateLimiter(cfg)
	assert.NotNil(t, limiter)
	assert.True(t, limiter.Enabled())
	assert.Equal(t, 100, limiter.Requests())
	assert.Equal(t, 200, limiter.Burst())
}

func TestNewRateLimiter_Disabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled: false,
	}

	limiter := middleware.NewRateLimiter(cfg)
	assert.NotNil(t, limiter)
	assert.False(t, limiter.Enabled())
}

func TestRateLimiter_Allow_Enabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  true,
		Requests: 100,
		Burst:    200,
	}

	limiter := middleware.NewRateLimiter(cfg)

	for i := 0; i < 200; i++ {
		assert.True(t, limiter.Allow(), "request %d should be allowed", i)
	}
}

func TestRateLimiter_Allow_Disabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled: false,
	}

	limiter := middleware.NewRateLimiter(cfg)
	assert.True(t, limiter.Allow())
}
