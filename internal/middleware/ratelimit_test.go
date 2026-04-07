package middleware

import (
	"testing"

	"db-mcp/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  true,
		Requests: 10,
		Burst:    20,
	}

	rl := NewRateLimiter(cfg)

	assert.NotNil(t, rl)
	assert.True(t, rl.Enabled())
	assert.Equal(t, 10, rl.Requests())
	assert.Equal(t, 20, rl.Burst())
}

func TestRateLimiter_Allow(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  true,
		Requests: 10,
		Burst:    5,
	}

	rl := NewRateLimiter(cfg)

	// First few requests should be allowed (burst)
	for i := 0; i < 5; i++ {
		allowed := rl.Allow()
		assert.True(t, allowed, "Request %d should be allowed within burst", i)
	}

	// After burst, requests should still be allowed (rate limiting is eventual)
	// The actual rate limiting happens over time
}

func TestRateLimiter_Disabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  false,
		Requests: 10,
		Burst:    5,
	}

	rl := NewRateLimiter(cfg)

	assert.False(t, rl.Enabled())

	// All requests should be allowed when disabled
	for i := 0; i < 20; i++ {
		allowed := rl.Allow()
		assert.True(t, allowed, "Request %d should be allowed when rate limit is disabled", i)
	}
}

func TestRateLimiter_Requests(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  true,
		Requests: 50,
		Burst:    25,
	}

	rl := NewRateLimiter(cfg)

	assert.Equal(t, 50, rl.Requests())
	assert.Equal(t, 25, rl.Burst())
}

func TestRateLimiter_Burst(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:  true,
		Requests: 100,
		Burst:    50,
	}

	rl := NewRateLimiter(cfg)

	assert.Equal(t, 100, rl.Requests())
	assert.Equal(t, 50, rl.Burst())
}
