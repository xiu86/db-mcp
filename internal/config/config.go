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
    Databases []InstanceConfig       `yaml:"databases" json:"databases"`
    Default   string                 `yaml:"default" json:"default"`       // 默认实例名
    MCP       MCPConfig              `yaml:"mcp" json:"mcp"`
    Log       LogConfig              `yaml:"log" json:"log"`
    RateLimit RateLimitConfig        `yaml:"rateLimit" json:"rateLimit"`
    Pool      PoolConfig             `yaml:"pool" json:"pool"`
    Timeout   TimeoutConfigLoadable  `yaml:"timeout" json:"timeout"`

    // 向后兼容字段
    Database *DatabaseConfig `yaml:"database,omitempty" json:"database,omitempty"`
    Mongo    *MongoConfig    `yaml:"mongo,omitempty" json:"mongo,omitempty"`
}

type TimeoutConfigLoadable struct {
    Connect     int `yaml:"connect" json:"connect"`         // seconds
    Query       int `yaml:"query" json:"query"`             // seconds
    Mutation    int `yaml:"mutation" json:"mutation"`       // seconds
    Transaction int `yaml:"transaction" json:"transaction"`   // seconds
}

type InstanceConfig struct {
    Type     string `yaml:"type" json:"type"`           // "mysql" | "mongodb"
    Name     string `yaml:"name" json:"name"`           // 实例名称
    Host     string `yaml:"host" json:"host"`
    Port     int    `yaml:"port" json:"port"`
    User     string `yaml:"user" json:"user"`
    Password string `yaml:"password" json:"password"`
    Database string `yaml:"database" json:"database"`
    Charset  string `yaml:"charset" json:"charset"`
    // MongoDB专用
    URI         string `yaml:"uri" json:"uri"`
    MaxPoolSize uint64 `yaml:"maxPoolSize" json:"maxPoolSize"`
    MinPoolSize uint64 `yaml:"minPoolSize" json:"minPoolSize"`
}

type DatabaseConfig struct {
    Host     string `yaml:"host" json:"host"`
    Port     int    `yaml:"port" json:"port"`
    User     string `yaml:"user" json:"user"`
    Password string `yaml:"password" json:"password"`
    Database string `yaml:"database" json:"database"`
    Charset  string `yaml:"charset" json:"charset"`
}

type MongoConfig struct {
    Host        string `yaml:"host" json:"host"`
    Port        int    `yaml:"port" json:"port"`
    URI         string `yaml:"uri" json:"uri"`
    Database    string `yaml:"database" json:"database"`
    Username    string `yaml:"username" json:"username"`
    Password    string `yaml:"password" json:"password"`
    AuthSource  string `yaml:"authSource" json:"authSource"`
    MaxPoolSize uint64 `yaml:"maxPoolSize" json:"maxPoolSize"`
    MinPoolSize uint64 `yaml:"minPoolSize" json:"minPoolSize"`
}

type MCPConfig struct {
    Transport    string   `yaml:"transport" json:"transport"`       // "stdio" | "http" | "sse", default "stdio"
    Host         string   `yaml:"host" json:"host"`                 // HTTP listen address, default "0.0.0.0"
    Port         int      `yaml:"port" json:"port"`                 // HTTP listen port, default 8080
    EndpointPath string   `yaml:"endpointPath" json:"endpointPath"` // HTTP endpoint path, default "/mcp"
    Tokens       []string `yaml:"tokens" json:"tokens"`             // Auth tokens
}

type LogConfig struct {
    Level      string `yaml:"level" json:"level"`
    Format     string `yaml:"format" json:"format"`
    Output     string `yaml:"output" json:"output"`
    AuditTable string `yaml:"auditTable" json:"auditTable"`
    AuditFile  string `yaml:"auditFile" json:"auditFile"`
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
        Databases: []InstanceConfig{
            {
                Type:     "mysql",
                Name:     "default",
                Host:     "localhost",
                Port:     3306,
                Charset:  "utf8mb4",
                Database: "",
                User:     "",
                Password: "",
            },
        },
        Default: "default",
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
            AuditFile:  "audit.log",
        },
        RateLimit: RateLimitConfig{
            Enabled:  true,
            Requests: 100,
            Burst:    200,
        },
        Timeout: TimeoutConfigLoadable{
            Connect:     5,
            Query:       30,
            Mutation:    10,
            Transaction: 60,
        },
        MCP: MCPConfig{
            Transport:    "stdio",
            Host:         "0.0.0.0",
            Port:         8080,
            EndpointPath: "/mcp",
            Tokens:       []string{},
        },
    }
}

func Load(configPath string) (*Config, error) {
    cfg := DefaultConfig()

    if configPath != "" {
        // #nosec G304 - configPath is user-provided but intentionally used for config file loading
        data, err := os.ReadFile(configPath)
        if err != nil {
            if !os.IsNotExist(err) {
                return nil, fmt.Errorf("failed to read config file: %w", err)
            }
            // Config file does not exist, use defaults + env vars
        } else {
            if err := yaml.Unmarshal(data, cfg); err != nil {
                return nil, fmt.Errorf("failed to parse yaml: %w", err)
            }
        }
    }

    // 向后兼容：如果使用旧的database字段，转换为databases
    if cfg.Database != nil {
        // Check if the databases array only has default values (from DefaultConfig)
        hasOnlyDefaults := len(cfg.Databases) == 1 &&
            cfg.Databases[0].Name == "default" &&
            cfg.Databases[0].User == "" &&
            cfg.Databases[0].Password == "" &&
            cfg.Databases[0].Database == ""

        if hasOnlyDefaults || len(cfg.Databases) == 0 {
            // Replace the default with the converted database config
            cfg.Databases = []InstanceConfig{
                {
                    Type:     "mysql",
                    Name:     "default",
                    Host:     cfg.Database.Host,
                    Port:     cfg.Database.Port,
                    User:     cfg.Database.User,
                    Password: cfg.Database.Password,
                    Database: cfg.Database.Database,
                    Charset:  cfg.Database.Charset,
                },
            }
        }
    }

    // 向后兼容：如果使用旧的mongo字段，转换为databases
    if cfg.Mongo != nil {
        mongoInstance := InstanceConfig{
            Type:        "mongodb",
            Name:        "mongo",
            URI:         cfg.Mongo.URI,
            Database:    cfg.Mongo.Database,
            MaxPoolSize: cfg.Mongo.MaxPoolSize,
            MinPoolSize: cfg.Mongo.MinPoolSize,
        }
        cfg.Databases = append(cfg.Databases, mongoInstance)
        // Set default to mongo when mongo config is present
        cfg.Default = "mongo"
    }

    // 环境变量覆盖 - 支持单实例(向后兼容)和多实例
    if instances := os.Getenv("DB_INSTANCES"); instances != "" {
        // 多实例模式：DB_INSTANCES=primary,secondary
        // Clear default databases and use only those from env
        cfg.Databases = []InstanceConfig{}
        instanceNames := parseInstances(instances)
        for _, name := range instanceNames {
            // Convert to uppercase for environment variable lookup
            prefix := fmt.Sprintf("DB_%s_", strings.ToUpper(name))
            instance := getInstanceConfig(name, prefix)
            cfg.Databases = append(cfg.Databases, *instance)
        }
        // Clear default since we're using multi-instance mode
        if cfg.Default == "default" {
            cfg.Default = ""
        }
    } else {
        // 单实例模式(向后兼容)
        if len(cfg.Databases) > 0 {
            if v := os.Getenv("DB_HOST"); v != "" {
                cfg.Databases[0].Host = v
            }
            if v := os.Getenv("DB_PORT"); v != "" {
                if port, err := strconv.Atoi(v); err == nil {
                    cfg.Databases[0].Port = port
                }
            }
            if v := os.Getenv("DB_USER"); v != "" {
                cfg.Databases[0].User = v
            }
            if v := os.Getenv("DB_PASSWORD"); v != "" {
                cfg.Databases[0].Password = v
            }
            if v := os.Getenv("DB_NAME"); v != "" {
                cfg.Databases[0].Database = v
            }
        }
    }

    // 至少需要一个数据库实例
    if len(cfg.Databases) == 0 {
        return nil, fmt.Errorf("at least one database instance is required")
    }

    // 默认实例校验
    if cfg.Default == "" || !instanceExists(cfg.Databases, cfg.Default) {
        // Default to first instance if none specified or doesn't exist
        cfg.Default = cfg.Databases[0].Name
    }

    // MCP环境变量覆盖
    if v := os.Getenv("MCP_TRANSPORT"); v != "" {
        cfg.MCP.Transport = v
    }
    if v := os.Getenv("MCP_HOST"); v != "" {
        cfg.MCP.Host = v
    }
    if v := os.Getenv("MCP_PORT"); v != "" {
        if port, err := strconv.Atoi(v); err == nil {
            cfg.MCP.Port = port
        }
    }
    if v := os.Getenv("MCP_ENDPOINT_PATH"); v != "" {
        cfg.MCP.EndpointPath = v
    }
    if v := os.Getenv("MCP_TOKEN"); v != "" {
        cfg.MCP.Tokens = splitByComma(v)
    }

    return cfg, nil
}

func parseInstances(instances string) []string {
    var names []string
    for _, name := range splitByComma(instances) {
        if name = strings.TrimSpace(name); name != "" {
            names = append(names, name)
        }
    }
    return names
}

func splitByComma(s string) []string {
    var parts []string
    start := 0
    for i, r := range s {
        if r == ',' {
            parts = append(parts, s[start:i])
            start = i + 1
        }
    }
    parts = append(parts, s[start:])
    return parts
}

func getInstanceConfig(name, prefix string) *InstanceConfig {
    instance := &InstanceConfig{
        Type:     "mysql",
        Name:     name,
        Host:     "localhost",
        Port:     3306,
        Charset:  "utf8mb4",
    }
    if v := os.Getenv(prefix + "TYPE"); v != "" {
        instance.Type = v
    }
    if v := os.Getenv(prefix + "HOST"); v != "" {
        instance.Host = v
    }
    if v := os.Getenv(prefix + "PORT"); v != "" {
        if port, err := strconv.Atoi(v); err == nil {
            instance.Port = port
        }
    }
    if v := os.Getenv(prefix + "USER"); v != "" {
        instance.User = v
    }
    if v := os.Getenv(prefix + "PASSWORD"); v != "" {
        instance.Password = v
    }
    if v := os.Getenv(prefix + "DATABASE"); v != "" {
        instance.Database = v
    }
    if v := os.Getenv(prefix + "CHARSET"); v != "" {
        instance.Charset = v
    }
    if v := os.Getenv(prefix + "URI"); v != "" {
        instance.URI = v
    }
    if v := os.Getenv(prefix + "MAX_POOL_SIZE"); v != "" {
        if size, err := strconv.ParseUint(v, 10, 64); err == nil {
            instance.MaxPoolSize = size
        }
    }
    if v := os.Getenv(prefix + "MIN_POOL_SIZE"); v != "" {
        if size, err := strconv.ParseUint(v, 10, 64); err == nil {
            instance.MinPoolSize = size
        }
    }
    return instance
}

func instanceExists(instances []InstanceConfig, name string) bool {
    for _, inst := range instances {
        if inst.Name == name {
            return true
        }
    }
    return false
}

func LoadFromMCP(params map[string]interface{}) *Config {
    cfg := DefaultConfig()

    // 支持instance参数指定目标实例
    instanceName := "default"
    if v, ok := params["instance"].(string); ok {
        instanceName = v
    }

    // 查找或创建指定实例
    var instance *InstanceConfig
    for i, inst := range cfg.Databases {
        if inst.Name == instanceName {
            instance = &cfg.Databases[i]
            break
        }
    }

    if instance == nil {
        // 创建新实例，替换默认实例
        cfg.Databases = []InstanceConfig{{
            Type: "mysql",
            Name: instanceName,
            Host: "localhost",
            Port: 3306,
        }}
        instance = &cfg.Databases[0]
    }

    // 从参数更新实例配置
    if v, ok := params["type"].(string); ok {
        instance.Type = v
    }
    if v, ok := params["host"].(string); ok {
        instance.Host = v
    }
    if v, ok := params["port"].(float64); ok {
        instance.Port = int(v)
    }
    if v, ok := params["user"].(string); ok {
        instance.User = v
    }
    if v, ok := params["password"].(string); ok {
        instance.Password = v
    }
    if v, ok := params["database"].(string); ok {
        instance.Database = v
    }
    if v, ok := params["charset"].(string); ok {
        instance.Charset = v
    }
    if v, ok := params["uri"].(string); ok {
        instance.URI = v
    }
    if v, ok := params["maxPoolSize"].(float64); ok {
        instance.MaxPoolSize = uint64(v)
    }
    if v, ok := params["minPoolSize"].(float64); ok {
        instance.MinPoolSize = uint64(v)
    }

    // 更新默认实例
    if instanceName != "" {
        cfg.Default = instanceName
    }

    return cfg
}
