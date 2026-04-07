package middleware_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/middleware"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := middleware.DefaultTimeoutConfig()

	assert.Equal(t, 5*time.Second, cfg.ConnectTimeout())
	assert.Equal(t, 30*time.Second, cfg.QueryTimeout())
	assert.Equal(t, 10*time.Second, cfg.MutationTimeout())
	assert.Equal(t, 60*time.Second, cfg.TransactionTimeout())
}

func TestTimeoutConfig_GetTimeout(t *testing.T) {
	cfg := middleware.DefaultTimeoutConfig()

	assert.Equal(t, 30*time.Second, cfg.GetTimeout("SELECT"))
	assert.Equal(t, 10*time.Second, cfg.GetTimeout("INSERT"))
	assert.Equal(t, 10*time.Second, cfg.GetTimeout("UPDATE"))
	assert.Equal(t, 10*time.Second, cfg.GetTimeout("DELETE"))
	assert.Equal(t, 60*time.Second, cfg.GetTimeout("TRANSACTION"))
	assert.Equal(t, 30*time.Second, cfg.GetTimeout("UNKNOWN"))
}
