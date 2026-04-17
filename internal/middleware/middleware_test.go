//go:build integration
// +build integration

package middleware

import (
	"context"
	"testing"
	"time"

	"db-mcp/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Wait(t *testing.T) {
	t.Run("wait with enabled limiter", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:  true,
			Requests: 10,
			Burst:    20,
		}
		limiter := NewRateLimiter(cfg)

		// Should not block - we're within the limit
		for i := 0; i < 5; i++ {
			err := limiter.Wait(context.Background())
			assert.NoError(t, err)
		}
	})

	t.Run("wait with disabled limiter", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled: false,
		}
		limiter := NewRateLimiter(cfg)

		// Should return immediately when disabled
		err := limiter.Wait(context.Background())
		assert.NoError(t, err)
	})

	t.Run("wait blocks when exceeded", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:  true,
			Requests: 1,
			Burst:    1,
		}
		limiter := NewRateLimiter(cfg)

		// Use up the tokens
		err := limiter.Wait(context.Background())
		assert.NoError(t, err)

		// Now it should block with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err = limiter.Wait(ctx)
		assert.Error(t, err) // Should timeout
	})

	t.Run("wait with cancelled context", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:  true,
			Requests: 1,
			Burst:    1,
		}
		limiter := NewRateLimiter(cfg)

		// Use up the token
		limiter.Wait(context.Background())

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := limiter.Wait(ctx)
		assert.Error(t, err)
	})
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("allow with enabled limiter", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:  true,
			Requests: 10,
			Burst:    20,
		}
		limiter := NewRateLimiter(cfg)

		// Should be allowed
		for i := 0; i < 10; i++ {
			assert.True(t, limiter.Allow())
		}
	})

	t.Run("allow with disabled limiter", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled: false,
		}
		limiter := NewRateLimiter(cfg)

		// Should always return true when disabled
		assert.True(t, limiter.Allow())
	})
}

func TestRateLimiter_Getters(t *testing.T) {
	t.Run("enabled limiter getters", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled:  true,
			Requests: 50,
			Burst:    100,
		}
		limiter := NewRateLimiter(cfg)

		assert.True(t, limiter.Enabled())
		assert.Equal(t, 50, limiter.Requests())
		assert.Equal(t, 100, limiter.Burst())
	})

	t.Run("disabled limiter getters", func(t *testing.T) {
		cfg := &config.RateLimitConfig{
			Enabled: false,
		}
		limiter := NewRateLimiter(cfg)

		assert.False(t, limiter.Enabled())
		assert.Equal(t, 0, limiter.Requests())
		assert.Equal(t, 0, limiter.Burst())
	})
}

func TestTimeoutConfig_GetTimeout(t *testing.T) {
	t.Run("get query timeout", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 60 * time.Second,
		}

		timeout := cfg.GetTimeout("query")
		assert.Equal(t, 30*time.Second, timeout)
	})

	t.Run("get mutation timeout for INSERT", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 60 * time.Second,
		}

		timeout := cfg.GetTimeout("INSERT")
		assert.Equal(t, 10*time.Second, timeout)
	})

	t.Run("get mutation timeout for UPDATE", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    15 * time.Second,
			Transaction: 60 * time.Second,
		}

		timeout := cfg.GetTimeout("UPDATE")
		assert.Equal(t, 15*time.Second, timeout)
	})

	t.Run("get mutation timeout for DELETE", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    20 * time.Second,
			Transaction: 60 * time.Second,
		}

		timeout := cfg.GetTimeout("DELETE")
		assert.Equal(t, 20*time.Second, timeout)
	})

	t.Run("get transaction timeout", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 90 * time.Second,
		}

		timeout := cfg.GetTimeout("TRANSACTION")
		assert.Equal(t, 90*time.Second, timeout)
	})

	t.Run("get default timeout for unknown operation", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       45 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 60 * time.Second,
		}

		timeout := cfg.GetTimeout("unknown_operation")
		assert.Equal(t, 45*time.Second, timeout) // Default to Query timeout
	})
}

func TestTimeoutConfig_Helpers(t *testing.T) {
	t.Run("ConnectTimeout helper", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     10 * time.Second,
			Query:       30 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 60 * time.Second,
		}

		assert.Equal(t, 10*time.Second, cfg.ConnectTimeout())
	})

	t.Run("QueryTimeout helper", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       45 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 60 * time.Second,
		}

		assert.Equal(t, 45*time.Second, cfg.QueryTimeout())
	})

	t.Run("MutationTimeout helper", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    20 * time.Second,
			Transaction: 60 * time.Second,
		}

		assert.Equal(t, 20*time.Second, cfg.MutationTimeout())
	})

	t.Run("TransactionTimeout helper", func(t *testing.T) {
		cfg := &TimeoutConfig{
			Connect:     5 * time.Second,
			Query:       30 * time.Second,
			Mutation:    10 * time.Second,
			Transaction: 120 * time.Second,
		}

		assert.Equal(t, 120*time.Second, cfg.TransactionTimeout())
	})
}

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 5*time.Second, cfg.Connect)
	assert.Equal(t, 30*time.Second, cfg.Query)
	assert.Equal(t, 10*time.Second, cfg.Mutation)
	assert.Equal(t, 60*time.Second, cfg.Transaction)
}
