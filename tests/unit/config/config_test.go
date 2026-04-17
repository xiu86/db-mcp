package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// Check databases array
	assert.NotEmpty(t, cfg.Databases)
	assert.Equal(t, "default", cfg.Databases[0].Name)
	assert.Equal(t, "mysql", cfg.Databases[0].Type)
	assert.Equal(t, "localhost", cfg.Databases[0].Host)
	assert.Equal(t, 3306, cfg.Databases[0].Port)
	assert.Equal(t, "utf8mb4", cfg.Databases[0].Charset)
	assert.Equal(t, "default", cfg.Default)

	// Check pool settings
	assert.Equal(t, 10, cfg.Pool.MaxIdleConns)
	assert.Equal(t, 100, cfg.Pool.MaxOpenConns)
	assert.Equal(t, time.Hour, cfg.Pool.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.Pool.ConnMaxIdleTime)

	// Check log settings
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, "_audit_logs", cfg.Log.AuditTable)

	// Check rate limit settings
	assert.True(t, cfg.RateLimit.Enabled)
	assert.Equal(t, 100, cfg.RateLimit.Requests)
	assert.Equal(t, 200, cfg.RateLimit.Burst)

	// Check MCP settings
	assert.Equal(t, "stdio", cfg.MCP.Transport)
	assert.Equal(t, "0.0.0.0", cfg.MCP.Host)
	assert.Equal(t, 8080, cfg.MCP.Port)
	assert.Equal(t, "/mcp", cfg.MCP.EndpointPath)
	assert.Empty(t, cfg.MCP.Tokens)
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yaml")

	// Should return default config with empty database credentials
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Databases)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "3307")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
	}()

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.Equal(t, "testhost", cfg.Databases[0].Host)
	assert.Equal(t, 3307, cfg.Databases[0].Port)
	assert.Equal(t, "testuser", cfg.Databases[0].User)
	assert.Equal(t, "testpass", cfg.Databases[0].Password)
	assert.Equal(t, "testdb", cfg.Databases[0].Database)
}

func TestLoadConfig_MultiInstance(t *testing.T) {
	// Set multi-instance environment variables
	os.Setenv("DB_INSTANCES", "primary,secondary")
	os.Setenv("DB_PRIMARY_HOST", "primary-host")
	os.Setenv("DB_PRIMARY_PORT", "3306")
	os.Setenv("DB_SECONDARY_HOST", "secondary-host")
	os.Setenv("DB_SECONDARY_PORT", "3307")
	defer func() {
		os.Unsetenv("DB_INSTANCES")
		os.Unsetenv("DB_PRIMARY_HOST")
		os.Unsetenv("DB_PRIMARY_PORT")
		os.Unsetenv("DB_SECONDARY_HOST")
		os.Unsetenv("DB_SECONDARY_PORT")
	}()

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(cfg.Databases))
	assert.Contains(t, []string{"primary", "secondary"}, cfg.Default)
}

func TestLoadConfig_BackwardCompatibility(t *testing.T) {
	// Test that old config format with Database field works
	yamlContent := `
database:
  host: oldhost
  port: 3306
  user: olduser
  password: oldpass
  database: olddb
  charset: utf8
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlContent)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "oldhost", cfg.Databases[0].Host)
	assert.Equal(t, "olduser", cfg.Databases[0].User)
}

func TestLoadConfig_MCPEnvOverride(t *testing.T) {
	// Set MCP environment variables
	os.Setenv("MCP_TRANSPORT", "http")
	os.Setenv("MCP_HOST", "127.0.0.1")
	os.Setenv("MCP_PORT", "9090")
	os.Setenv("MCP_ENDPOINT_PATH", "/api/mcp")
	os.Setenv("MCP_TOKEN", "token1,token2,token3")
	defer func() {
		os.Unsetenv("MCP_TRANSPORT")
		os.Unsetenv("MCP_HOST")
		os.Unsetenv("MCP_PORT")
		os.Unsetenv("MCP_ENDPOINT_PATH")
		os.Unsetenv("MCP_TOKEN")
	}()

	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.Equal(t, "http", cfg.MCP.Transport)
	assert.Equal(t, "127.0.0.1", cfg.MCP.Host)
	assert.Equal(t, 9090, cfg.MCP.Port)
	assert.Equal(t, "/api/mcp", cfg.MCP.EndpointPath)
	assert.Equal(t, []string{"token1", "token2", "token3"}, cfg.MCP.Tokens)
}

func TestLoadConfig_MCPYAMLConfig(t *testing.T) {
	// Test MCP config from YAML
	yamlContent := `
mcp:
  transport: sse
  host: localhost
  port: 3000
  endpointPath: /sse
  tokens:
    - secret1
    - secret2
`
	tmpFile, err := os.CreateTemp("", "config-mcp-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlContent)
	assert.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "sse", cfg.MCP.Transport)
	assert.Equal(t, "localhost", cfg.MCP.Host)
	assert.Equal(t, 3000, cfg.MCP.Port)
	assert.Equal(t, "/sse", cfg.MCP.EndpointPath)
	assert.Equal(t, []string{"secret1", "secret2"}, cfg.MCP.Tokens)
}
