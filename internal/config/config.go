package config

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Database  DatabaseConfig  `yaml:"database" json:"database"`
    MCP       MCPConfig       `yaml:"mcp" json:"mcp"`
    Log       LogConfig       `yaml:"log" json:"log"`
    RateLimit RateLimitConfig `yaml:"rateLimit" json:"rateLimit"`
    Pool      PoolConfig      `yaml:"pool" json:"pool"`
}

type DatabaseConfig struct {
    Host     string `yaml:"host" json:"host"`
    Port     int    `yaml:"port" json:"port"`
    User     string `yaml:"user" json:"user"`
    Password string `yaml:"password" json:"password"`
    Database string `yaml:"database" json:"database"`
    Charset  string `yaml:"charset" json:"charset"`
}

type MCPConfig struct {
    Host string `yaml:"host" json:"host"`
    Port int    `yaml:"port" json:"port"`
}

type LogConfig struct {
    Level      string `yaml:"level" json:"level"`
    Format     string `yaml:"format" json:"format"`
    Output     string `yaml:"output" json:"output"`
    AuditTable string `yaml:"auditTable" json:"auditTable"`
}

type RateLimitConfig struct {
    Enabled  bool `yaml:"enabled" json:"enabled"`
    Requests int  `yaml:"requests" json:"requests"`
    Burst    int  `yaml:"burst" json:"burst"`
}

type PoolConfig struct {
    MaxIdleConns    int           `yaml:"maxIdleConns" json:"maxIdleConns"`
    MaxOpenConns    int           `yaml:"maxOpenConns" json:"maxOpenConns"`
    ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"connMaxLifetime"`
    ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" json:"connMaxIdleTime"`
}

func DefaultConfig() *Config {
    return &Config{
        Database: DatabaseConfig{
            Host:    "localhost",
            Port:    3306,
            Charset: "utf8mb4",
        },
        Pool: PoolConfig{
            MaxIdleConns:    10,
            MaxOpenConns:    100,
            ConnMaxLifetime: time.Hour,
            ConnMaxIdleTime: 10 * time.Minute,
        },
        Log: LogConfig{
            Level:      "info",
            Format:     "json",
            Output:     "stdout",
            AuditTable: "_audit_logs",
        },
        RateLimit: RateLimitConfig{
            Enabled:  true,
            Requests: 100,
            Burst:    200,
        },
    }
}

func Load(configPath string) (*Config, error) {
    cfg := DefaultConfig()

    if configPath != "" {
        data, err := os.ReadFile(configPath)
        if err != nil {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }

        if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
            if err := yaml.Unmarshal(data, cfg); err != nil {
                return nil, fmt.Errorf("failed to parse yaml: %w", err)
            }
        } else {
            if err := yaml.Unmarshal(data, cfg); err != nil {
                return nil, fmt.Errorf("failed to parse config: %w", err)
            }
        }
    }

    // 环境变量覆盖
    if v := os.Getenv("DB_HOST"); v != "" {
        cfg.Database.Host = v
    }
    if v := os.Getenv("DB_PORT"); v != "" {
        if port, err := strconv.Atoi(v); err == nil {
            cfg.Database.Port = port
        }
    }
    if v := os.Getenv("DB_USER"); v != "" {
        cfg.Database.User = v
    }
    if v := os.Getenv("DB_PASSWORD"); v != "" {
        cfg.Database.Password = v
    }
    if v := os.Getenv("DB_NAME"); v != "" {
        cfg.Database.Database = v
    }

    // 必填校验
    if cfg.Database.User == "" || cfg.Database.Password == "" || cfg.Database.Database == "" {
        return nil, fmt.Errorf("database user, password and database are required")
    }

    return cfg, nil
}

func LoadFromMCP(params map[string]interface{}) *Config {
    cfg := DefaultConfig()

    if v, ok := params["host"].(string); ok {
        cfg.Database.Host = v
    }
    if v, ok := params["port"].(float64); ok {
        cfg.Database.Port = int(v)
    }
    if v, ok := params["user"].(string); ok {
        cfg.Database.User = v
    }
    if v, ok := params["password"].(string); ok {
        cfg.Database.Password = v
    }
    if v, ok := params["database"].(string); ok {
        cfg.Database.Database = v
    }

    return cfg
}
