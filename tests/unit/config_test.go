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

    assert.Equal(t, "localhost", cfg.Database.Host)
    assert.Equal(t, 3306, cfg.Database.Port)
    assert.Equal(t, "utf8mb4", cfg.Database.Charset)
    assert.Equal(t, 10, cfg.Pool.MaxIdleConns)
    assert.Equal(t, 100, cfg.Pool.MaxOpenConns)
    assert.Equal(t, time.Hour, cfg.Pool.ConnMaxLifetime)
    assert.Equal(t, 10*time.Minute, cfg.Pool.ConnMaxIdleTime)
    assert.Equal(t, "info", cfg.Log.Level)
    assert.Equal(t, "json", cfg.Log.Format)
    assert.Equal(t, "_audit_logs", cfg.Log.AuditTable)
    assert.True(t, cfg.RateLimit.Enabled)
    assert.Equal(t, 100, cfg.RateLimit.Requests)
    assert.Equal(t, 200, cfg.RateLimit.Burst)
}

func TestLoadConfig_FileNotFound(t *testing.T) {
    _, err := config.Load("nonexistent.yaml")
    assert.Error(t, err)
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
    tmpFile, _ := os.CreateTemp("", "config-*.yaml")
    defer os.Remove(tmpFile.Name())
    tmpFile.WriteString("database:\n  host: localhost\n")
    tmpFile.Close()

    _, err := config.Load(tmpFile.Name())
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "required")
}

func TestLoadConfig_Success(t *testing.T) {
    tmpFile, _ := os.CreateTemp("", "config-*.yaml")
    defer os.Remove(tmpFile.Name())
    tmpFile.WriteString(`
database:
  host: localhost
  port: 3306
  user: root
  password: secret
  database: testdb
`)
    tmpFile.Close()

    cfg, err := config.Load(tmpFile.Name())
    assert.NoError(t, err)
    assert.Equal(t, "root", cfg.Database.User)
    assert.Equal(t, "secret", cfg.Database.Password)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
    os.Setenv("DB_HOST", "envhost")
    os.Setenv("DB_USER", "envuser")
    defer os.Unsetenv("DB_HOST")
    defer os.Unsetenv("DB_USER")

    tmpFile, _ := os.CreateTemp("", "config-*.yaml")
    defer os.Remove(tmpFile.Name())
    tmpFile.WriteString(`
database:
  host: filehost
  user: fileuser
  password: secret
  database: testdb
`)
    tmpFile.Close()

    cfg, err := config.Load(tmpFile.Name())
    assert.NoError(t, err)
    assert.Equal(t, "envhost", cfg.Database.Host)
    assert.Equal(t, "envuser", cfg.Database.User)
}

func TestLoadFromMCP(t *testing.T) {
    params := map[string]interface{}{
        "host":     "mcphost",
        "port":     float64(3307),
        "user":     "mcpuser",
        "password": "mcpsecret",
        "database": "mcpdb",
    }

    cfg := config.LoadFromMCP(params)
    assert.Equal(t, "mcphost", cfg.Database.Host)
    assert.Equal(t, 3307, cfg.Database.Port)
    assert.Equal(t, "mcpuser", cfg.Database.User)
}
