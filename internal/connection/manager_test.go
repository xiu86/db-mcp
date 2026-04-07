package connection

import (
	"testing"
	"time"

	"db-mcp/internal/config"

	"github.com/stretchr/testify/assert"
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

	dsn := BuildDSN(cfg)

	assert.Contains(t, dsn, "root:secret")
	assert.Contains(t, dsn, "tcp(localhost:3306)")
	assert.Contains(t, dsn, "testdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=True")
	assert.Contains(t, dsn, "loc=Local")
}

func TestBuildDSN_SpecialCharacters(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "db.example.com",
		Port:     3307,
		User:     "admin",
		Password: "p@ss!word",
		Database: "production_db",
		Charset:  "utf8",
	}

	dsn := BuildDSN(cfg)

	assert.Contains(t, dsn, "admin:p@ss!word")
	assert.Contains(t, dsn, "tcp(db.example.com:3307)")
	assert.Contains(t, dsn, "production_db")
}

func TestBuildDSN_DifferentPorts(t *testing.T) {
	testCases := []struct {
		port    int
		pattern string
	}{
		{3306, "tcp(localhost:3306)"},
		{3307, "tcp(localhost:3307)"},
		{5432, "tcp(localhost:5432)"},
	}

	for _, tc := range testCases {
		cfg := &config.DatabaseConfig{
			Host:     "localhost",
			Port:     tc.port,
			User:     "root",
			Password: "secret",
			Database: "testdb",
			Charset:  "utf8mb4",
		}

		dsn := BuildDSN(cfg)
		assert.Contains(t, dsn, tc.pattern, "Port %d should be in DSN", tc.port)
	}
}

func TestNewConnectionManager_InvalidDSN(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "invalid-host-that-does-not-exist",
			Port:     99999,
			User:     "root",
			Password: "secret",
			Database: "testdb",
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	// This should fail because the host is invalid
	_, err := NewConnectionManager(cfg, nil)
	assert.Error(t, err)
}

func TestConnectionManager_Close(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "secret",
			Database: "testdb",
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	mgr, err := NewConnectionManager(cfg, nil)
	if err != nil {
		// Expected to fail if database is not available
		t.Skipf("Skipping test: cannot connect to database: %v", err)
		return
	}

	err = mgr.Close()
	assert.NoError(t, err)
}
