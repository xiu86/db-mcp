package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, 5*time.Second, cfg.Connect)
	assert.Equal(t, 30*time.Second, cfg.Query)
	assert.Equal(t, 10*time.Second, cfg.Mutation)
	assert.Equal(t, 60*time.Second, cfg.Transaction)
}

func TestTimeoutConfig_GetTimeout(t *testing.T) {
	cfg := &TimeoutConfig{
		Connect:     5 * time.Second,
		Query:       30 * time.Second,
		Mutation:    10 * time.Second,
		Transaction: 60 * time.Second,
	}

	testCases := []struct {
		operation string
		expected  time.Duration
	}{
		{"INSERT", 10 * time.Second},
		{"UPDATE", 10 * time.Second},
		{"DELETE", 10 * time.Second},
		{"TRANSACTION", 60 * time.Second},
		{"SELECT", 30 * time.Second},
		{"QUERY", 30 * time.Second},
		{"UNKNOWN", 30 * time.Second},
		{"", 30 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.operation, func(t *testing.T) {
			got := cfg.GetTimeout(tc.operation)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestTimeoutConfig_ConnectTimeout(t *testing.T) {
	cfg := &TimeoutConfig{
		Connect: 5 * time.Second,
	}

	assert.Equal(t, 5*time.Second, cfg.ConnectTimeout())
}

func TestTimeoutConfig_QueryTimeout(t *testing.T) {
	cfg := &TimeoutConfig{
		Query: 30 * time.Second,
	}

	assert.Equal(t, 30*time.Second, cfg.QueryTimeout())
}

func TestTimeoutConfig_MutationTimeout(t *testing.T) {
	cfg := &TimeoutConfig{
		Mutation: 10 * time.Second,
	}

	assert.Equal(t, 10*time.Second, cfg.MutationTimeout())
}

func TestTimeoutConfig_TransactionTimeout(t *testing.T) {
	cfg := &TimeoutConfig{
		Transaction: 60 * time.Second,
	}

	assert.Equal(t, 60*time.Second, cfg.TransactionTimeout())
}

func TestTimeoutConfig_CustomValues(t *testing.T) {
	cfg := &TimeoutConfig{
		Connect:     3 * time.Second,
		Query:       15 * time.Second,
		Mutation:    5 * time.Second,
		Transaction: 120 * time.Second,
	}

	assert.Equal(t, 3*time.Second, cfg.Connect)
	assert.Equal(t, 15*time.Second, cfg.Query)
	assert.Equal(t, 5*time.Second, cfg.Mutation)
	assert.Equal(t, 120*time.Second, cfg.Transaction)
}

func TestTimeoutConfig_ZeroValues(t *testing.T) {
	cfg := &TimeoutConfig{
		Connect:     0,
		Query:       0,
		Mutation:    0,
		Transaction: 0,
	}

	assert.Equal(t, time.Duration(0), cfg.Connect)
	assert.Equal(t, time.Duration(0), cfg.Query)
	assert.Equal(t, time.Duration(0), cfg.Mutation)
	assert.Equal(t, time.Duration(0), cfg.Transaction)
}
