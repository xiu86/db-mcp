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

	// Verify database config (backward compatibility: database field converted to databases)
	assert.Len(t, cfg.Databases, 1)
	assert.Equal(t, "localhost", cfg.Databases[0].Host)
	assert.Equal(t, 3306, cfg.Databases[0].Port)
	assert.Equal(t, "test_user", cfg.Databases[0].User)
	assert.Equal(t, "test_pass", cfg.Databases[0].Password)
	assert.Equal(t, "test_db", cfg.Databases[0].Database)
	assert.Equal(t, "default", cfg.Default)

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
	// Non-existent path now returns default config, not an error
	cfg, err := Load("/nonexistent/path/config.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "default", cfg.Default)
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

	assert.Equal(t, "remote-host", cfg.Databases[0].Host)
	assert.Equal(t, 3307, cfg.Databases[0].Port)
	assert.Equal(t, "mcp_user", cfg.Databases[0].User)
	assert.Equal(t, "mcp_pass", cfg.Databases[0].Password)
	assert.Equal(t, "mcp_db", cfg.Databases[0].Database)
	assert.Equal(t, "default", cfg.Default)
}

func TestLoadFromMCP_PartialParams(t *testing.T) {
	params := map[string]interface{}{
		"host": "partial-host",
	}

	cfg := LoadFromMCP(params)

	assert.Equal(t, "partial-host", cfg.Databases[0].Host)
	// Other fields should use defaults
	assert.Equal(t, 3306, cfg.Databases[0].Port)
	assert.Equal(t, "default", cfg.Default)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Databases, 1)
	assert.Equal(t, "localhost", cfg.Databases[0].Host)
	assert.Equal(t, 3306, cfg.Databases[0].Port)
	assert.Equal(t, "utf8mb4", cfg.Databases[0].Charset)
	assert.Equal(t, "default", cfg.Default)

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
	assert.Equal(t, "env-host", cfg.Databases[0].Host)
	assert.Equal(t, 3307, cfg.Databases[0].Port)
	assert.Equal(t, "env_user", cfg.Databases[0].User)
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// The new design doesn't validate required fields at config level
	// Fields are validated at connection time instead
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

	cfg, err := Load(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// User field is empty but config loads successfully
	assert.Len(t, cfg.Databases, 1)
	assert.Equal(t, "", cfg.Databases[0].User)
	assert.Equal(t, "secret", cfg.Databases[0].Password)
}

func TestLoad_MultiDatabaseConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
databases:
  - type: mysql
    name: primary
    host: primary-db
    port: 3306
    user: primary_user
    password: primary_pass
    database: primary_db
    charset: utf8mb4
  - type: mongodb
    name: mongo
    uri: mongodb://mongo-server:27017
    database: mongo_db
    maxPoolSize: 100
    minPoolSize: 10
default: primary
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := Load(configPath)
	assert.NoError(t, err)
	assert.Len(t, cfg.Databases, 2)
	assert.Equal(t, "primary", cfg.Default)

	// Check primary MySQL instance
	primary := cfg.Databases[0]
	assert.Equal(t, "mysql", primary.Type)
	assert.Equal(t, "primary", primary.Name)
	assert.Equal(t, "primary-db", primary.Host)
	assert.Equal(t, 3306, primary.Port)
	assert.Equal(t, "primary_user", primary.User)
	assert.Equal(t, "primary_pass", primary.Password)
	assert.Equal(t, "primary_db", primary.Database)

	// Check MongoDB instance
	mongo := cfg.Databases[1]
	assert.Equal(t, "mongodb", mongo.Type)
	assert.Equal(t, "mongo", mongo.Name)
	assert.Equal(t, "mongodb://mongo-server:27017", mongo.URI)
	assert.Equal(t, "mongo_db", mongo.Database)
	assert.Equal(t, uint64(100), mongo.MaxPoolSize)
	assert.Equal(t, uint64(10), mongo.MinPoolSize)
}

func TestLoad_BackwardCompatibility_SingleDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  host: old-db
  port: 3307
  user: old_user
  password: old_pass
  database: old_db
  charset: utf8mb4
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := Load(configPath)
	assert.NoError(t, err)
	assert.Len(t, cfg.Databases, 1)
	assert.Equal(t, "default", cfg.Default)

	db := cfg.Databases[0]
	assert.Equal(t, "mysql", db.Type)
	assert.Equal(t, "default", db.Name)
	assert.Equal(t, "old-db", db.Host)
	assert.Equal(t, 3307, db.Port)
	assert.Equal(t, "old_user", db.User)
	assert.Equal(t, "old_pass", db.Password)
	assert.Equal(t, "old_db", db.Database)
	assert.Equal(t, "utf8mb4", db.Charset)
}

func TestLoad_BackwardCompatibility_MongoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  host: mysql-db
  port: 3306
  user: mysql_user
  password: mysql_pass
  database: mysql_db
mongo:
  uri: mongodb://mongo:27017
  database: mongo_db
  maxPoolSize: 50
  minPoolSize: 5
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := Load(configPath)
	assert.NoError(t, err)
	assert.Len(t, cfg.Databases, 2)
	assert.Equal(t, "mongo", cfg.Default) // MongoDB takes precedence as default

	// MySQL instance
	mysql := cfg.Databases[0]
	assert.Equal(t, "mysql", mysql.Type)
	assert.Equal(t, "default", mysql.Name)

	// MongoDB instance
	mongo := cfg.Databases[1]
	assert.Equal(t, "mongodb", mongo.Type)
	assert.Equal(t, "mongo", mongo.Name)
	assert.Equal(t, "mongodb://mongo:27017", mongo.URI)
	assert.Equal(t, uint64(50), mongo.MaxPoolSize)
	assert.Equal(t, uint64(5), mongo.MinPoolSize)
}

func TestLoad_MultiDatabase_EnvironmentVariables(t *testing.T) {
	os.Setenv("DB_INSTANCES", "primary,secondary")
	os.Setenv("DB_PRIMARY_HOST", "primary-env")
	os.Setenv("DB_PRIMARY_DATABASE", "primary_env_db")
	os.Setenv("DB_SECONDARY_HOST", "secondary-env")
	os.Setenv("DB_SECONDARY_PORT", "3307")
	defer func() {
		os.Unsetenv("DB_INSTANCES")
		os.Unsetenv("DB_PRIMARY_HOST")
		os.Unsetenv("DB_PRIMARY_DATABASE")
		os.Unsetenv("DB_SECONDARY_HOST")
		os.Unsetenv("DB_SECONDARY_PORT")
	}()

	cfg, err := Load("")
	assert.NoError(t, err)
	assert.Len(t, cfg.Databases, 2)
	assert.Equal(t, "primary", cfg.Default) // Defaults to first instance

	// Check primary instance (created from env)
	primary := cfg.Databases[0]
	assert.Equal(t, "mysql", primary.Type)
	assert.Equal(t, "primary", primary.Name)
	assert.Equal(t, "primary-env", primary.Host)
	assert.Equal(t, "primary_env_db", primary.Database)

	// Check secondary instance
	secondary := cfg.Databases[1]
	assert.Equal(t, "mysql", secondary.Type)
	assert.Equal(t, "secondary", secondary.Name)
	assert.Equal(t, "secondary-env", secondary.Host)
	assert.Equal(t, 3307, secondary.Port)
}

func TestLoadFromMCP_MultiDatabase(t *testing.T) {
	params := map[string]interface{}{
		"instance": "test",
		"type":     "mysql",
		"host":     "mcp-host",
		"port":     float64(3308),
		"user":     "mcp_user",
		"password": "mcp_pass",
		"database": "mcp_db",
	}

	cfg := LoadFromMCP(params)
	assert.Equal(t, "test", cfg.Default)
	assert.Len(t, cfg.Databases, 1)

	db := cfg.Databases[0]
	assert.Equal(t, "mysql", db.Type)
	assert.Equal(t, "test", db.Name)
	assert.Equal(t, "mcp-host", db.Host)
	assert.Equal(t, 3308, db.Port)
	assert.Equal(t, "mcp_user", db.User)
	assert.Equal(t, "mcp_pass", db.Password)
	assert.Equal(t, "mcp_db", db.Database)
}

func TestLoadFromMCP_CreateNewInstance(t *testing.T) {
	// Test creating an instance that doesn't exist
	params := map[string]interface{}{
		"instance": "new-instance",
		"host":     "new-host",
	}

	cfg := LoadFromMCP(params)
	assert.Equal(t, "new-instance", cfg.Default)
	assert.Len(t, cfg.Databases, 1)

	db := cfg.Databases[0]
	assert.Equal(t, "mysql", db.Type)
	assert.Equal(t, "new-instance", db.Name)
	assert.Equal(t, "new-host", db.Host)
	assert.Equal(t, 3306, db.Port) // Default port
}

func TestLoad_MultiDatabaseValidation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Multi-database config with missing default
	configContent := `
databases:
  - type: mysql
    name: primary
    host: primary-db
    port: 3306
    user: primary_user
    password: primary_pass
    database: primary_db
  - type: mysql
    name: secondary
    host: secondary-db
    port: 3307
    user: secondary_user
    password: secondary_pass
    database: secondary_db
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := Load(configPath)
	assert.NoError(t, err)
	assert.Len(t, cfg.Databases, 2)
	assert.Equal(t, "primary", cfg.Default) // Should default to first instance
}

func TestLoad_MissingDefaultInstance(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Invalid config: default instance doesn't exist
	configContent := `
databases:
  - type: mysql
    name: primary
    host: primary-db
    port: 3306
    user: primary_user
    password: primary_pass
    database: primary_db
default: nonexistent
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	cfg, err := Load(configPath)
	assert.NoError(t, err)
	// Auto-correct to first instance when default doesn't exist
	assert.Equal(t, "primary", cfg.Default)
}
