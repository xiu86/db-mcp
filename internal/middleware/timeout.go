package middleware

import (
	"time"
)

type TimeoutConfig struct {
	Connect     time.Duration
	Query       time.Duration
	Mutation    time.Duration
	Transaction time.Duration
}

func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Connect:     5 * time.Second,
		Query:       30 * time.Second,
		Mutation:    10 * time.Second,
		Transaction: 60 * time.Second,
	}
}

func (t *TimeoutConfig) GetTimeout(op string) time.Duration {
	switch op {
	case "INSERT", "UPDATE", "DELETE":
		return t.Mutation
	case "TRANSACTION":
		return t.Transaction
	default:
		return t.Query
	}
}

func (t *TimeoutConfig) ConnectTimeout() time.Duration {
	return t.Connect
}

func (t *TimeoutConfig) QueryTimeout() time.Duration {
	return t.Query
}

func (t *TimeoutConfig) MutationTimeout() time.Duration {
	return t.Mutation
}

func (t *TimeoutConfig) TransactionTimeout() time.Duration {
	return t.Transaction
}
