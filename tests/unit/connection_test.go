package connection_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/config"
	"db-mcp/internal/connection"
)

func TestBuildDSN(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "testdb",
		Charset:  "utf8mb4",
	}

	dsn := connection.BuildDSN(cfg)
	assert.Contains(t, dsn, "root:secret")
	assert.Contains(t, dsn, "tcp(localhost:3306)")
	assert.Contains(t, dsn, "testdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
}
