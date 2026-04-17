package connection

import (
	"testing"

	"db-mcp/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestBuildDSN(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "p@ssw0rd",
		Database: "testdb",
		Charset:  "utf8mb4",
	}

	dsn := BuildDSN(cfg)
	assert.Contains(t, dsn, "root:p%40ssw0rd@tcp(localhost:3306)/testdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
}

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