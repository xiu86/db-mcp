package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_DefaultPath(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  host: localhost
  port: 3306
  user: test_user
  password: test_pass
  database: test_db
  charset: utf8mb4

pool:
  maxIdleConns: 5
  maxOpenConns: 20
  connMaxLifetime: 1h
  connMaxIdleTime: 10m

log:
  level: debug
  format: json
  output: stdout

rateLimit:
  enabled: true
  requests: 50
  burst: 100
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test loading
	cfg, err := Load(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify database config
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 3306, cfg.Database.Port)
	assert.Equal(t, "test_user", cfg.Database.User)
	assert.Equal(t, "test_pass", cfg.Database.Password)
	assert.Equal(t, "test_db", cfg.Database.Database)

	// Verify pool config (from YAML, not default)
	assert.Equal(t, 5, cfg.Pool.MaxIdleConns)
	assert.Equal(t, 20, cfg.Pool.MaxOpenConns)
	assert.Equal(t, time.Hour, cfg.Pool.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.Pool.ConnMaxIdleTime)

	// Verify log config
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)

	// Verify rate limit config (from YAML)
	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 50, cfg.RateLimit.Requests)
	assert.Equal(t, 100, cfg.RateLimit.Burst)
}

func TestLoad_InvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	assert.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)
	assert.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
}

func TestLoadFromMCP(t *testing.T) {
	params := map[string]interface{}{
		"host":     "remote-host",
		"port":     float64(3307),
		"user":     "mcp_user",
		"password": "mcp_pass",
		"database": "mcp_db",
	}

	cfg := LoadFromMCP(params)

	assert.Equal(t, "remote-host", cfg.Database.Host)
	assert.Equal(t, 3307, cfg.Database.Port)
	assert.Equal(t, "mcp_user", cfg.Database.User)
	assert.Equal(t, "mcp_pass", cfg.Database.Password)
	assert.Equal(t, "mcp_db", cfg.Database.Database)
}

func TestLoadFromMCP_PartialParams(t *testing.T) {
	params := map[string]interface{}{
		"host": "partial-host",
	}

	cfg := LoadFromMCP(params)

	assert.Equal(t, "partial-host", cfg.Database.Host)
	// Other fields should use defaults
	assert.Equal(t, 3306, cfg.Database.Port)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 3306, cfg.Database.Port)
	assert.Equal(t, "utf8mb4", cfg.Database.Charset)

	assert.Equal(t, 10, cfg.Pool.MaxIdleConns)
	assert.Equal(t, 100, cfg.Pool.MaxOpenConns)

	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "stdout", cfg.Log.Output)

	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 100, cfg.RateLimit.Requests)
	assert.Equal(t, 200, cfg.RateLimit.Burst)
}

func TestLoad_EnvironmentOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  host: config-host
  port: 3306
  user: config_user
  password: config_pass
  database: config_db
  charset: utf8mb4
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Set environment variables
	os.Setenv("DB_HOST", "env-host")
	os.Setenv("DB_PORT", "3307")
	os.Setenv("DB_USER", "env_user")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
	}()

	cfg, err := Load(configPath)
	assert.NoError(t, err)

	// Environment should override config file
	assert.Equal(t, "env-host", cfg.Database.Host)
	assert.Equal(t, 3307, cfg.Database.Port)
	assert.Equal(t, "env_user", cfg.Database.User)
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Missing user
	configContent := `
database:
  host: localhost
  port: 3306
  password: secret
  database: test_db
  charset: utf8mb4
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}
