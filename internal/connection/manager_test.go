package connection

import (
	"testing"

	"db-mcp/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestConnectionManager_EmptyConfig(t *testing.T) {
	emptyConfig := &config.Config{
		Database: &config.DatabaseConfig{
			User:     "root",
			Password: "password",
			Database: "test",
		},
	}

	manager, err := NewConnectionManager(emptyConfig, nil)
	// Should fail since we can't connect without valid DB
	assert.Error(t, err)
	assert.Nil(t, manager)
}