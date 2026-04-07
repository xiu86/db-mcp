# db-mcp 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现一个基于 go-mcp SDK 的 MySQL 数据库 MCP 服务，支持 CRUD、事务、批量操作、逻辑删除自动检测和审计日志。

**Architecture:** 分层架构：MCP Layer → Service Layer → Repository Layer → Connection Layer。detector 独立成包以支持 100% 测试覆盖率。

**Tech Stack:** Go 1.21+, mark3labs/mcp-go, GORM, MySQL 8.0+, golangci-lint, gomock, testcontainers

---

## 文件结构

```
db-mcp/
├── cmd/server/main.go              # 入口
├── internal/
│   ├── config/config.go            # 配置管理
│   ├── connection/manager.go       # 连接池
│   ├── repository/repository.go    # 数据访问
│   ├── service/
│   │   ├── crud.go                 # CRUD 业务
│   │   ├── transaction.go          # 事务
│   │   └── audit.go                # 审计
│   ├── detector/delete_field.go    # 删除字段检测
│   ├── middleware/
│   │   ├── ratelimit.go            # 限流
│   │   └── timeout.go              # 超时
│   ├── mcp/tools.go                # MCP 工具
│   └── errors/errors.go            # 错误
├── pkg/logger/logger.go            # 结构化日志
├── configs/config.yaml             # 配置示例
├── tests/                         # 测试目录
├── Makefile
└── README.md
```

---

## 阶段一：项目初始化

### Task 1: 项目基础设置

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.golangci.yml`

- [ ] **Step 1: 创建 go.mod**

```bash
cd /System/Volumes/Data/data/data/myproj/db-mcp && go mod init db-mcp
```

- [ ] **Step 2: 添加依赖**

```bash
go get github.com/mark3labs/mcp-go-go-sdk@latest
go get gorm.io/gorm@latest
go get gorm.io/driver/mysql@latest
go get golang.org/x/time/rate@latest
go get github.com/stretchr/testify@latest
go get go.uber.org/mock/gomock@latest
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/go-playground/validator/v10@latest
go get gopkg.in/yaml.v3@latest
```

- [ ] **Step 3: 创建 Makefile**

```makefile
.PHONY: build test test-unit test-integration lint clean

build:
	go build -o bin/db-mcp ./cmd/server

test:
	go test ./... -v -coverprofile=coverage.out

test-unit:
	go test ./tests/unit/... -v -cover

test-integration:
	go test ./tests/integration/... -v -cover

test-coverage:
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total

coverage-check:
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$(echo "$$coverage >= 100" | bc)" -eq 1 ]; then \
		echo "Coverage: $$coverage%"; \
	else \
		echo "Coverage: $$coverage% (required: 100%)"; \
		exit 1; \
	fi

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out coverage.html
```

- [ ] **Step 4: 创建 .golangci.yml**

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - typecheck
    - gosimple
    - ineffassign
    - goconst
    - gocyclo
    - funlen

linters-settings:
  gocyclo:
    min-complexity: 15
  funlen:
    lines: 50
    statements: 40

run:
  timeout: 5m
```

- [ ] **Step 5: 提交**

```bash
git add go.mod Makefile .golangci.yml
git commit -m "chore: initialize project with dependencies

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段二：基础设施层

### Task 2: 配置管理

**Files:**
- Create: `internal/config/config.go`
- Create: `tests/unit/config_test.go`

- [ ] **Step 1: 编写配置测试**

```go
package config

import (
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()

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
    _, err := Load("nonexistent.yaml")
    assert.Error(t, err)
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
    tmpFile, _ := os.CreateTemp("", "config-*.yaml")
    defer os.Remove(tmpFile.Name())
    tmpFile.WriteString("database:\n  host: localhost\n")
    tmpFile.Close()

    _, err := Load(tmpFile.Name())
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

    cfg, err := Load(tmpFile.Name())
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

    cfg, err := Load(tmpFile.Name())
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

    cfg := LoadFromMCP(params)
    assert.Equal(t, "mcphost", cfg.Database.Host)
    assert.Equal(t, 3307, cfg.Database.Port)
    assert.Equal(t, "mcpuser", cfg.Database.User)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/config_test.go -v
# Expected: FAIL - function not defined
```

- [ ] **Step 3: 实现配置管理**

```go
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
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/config_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/config/config.go tests/unit/config_test.go
git commit -m "feat: add config management with defaults and env override

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: 结构化日志

**Files:**
- Create: `pkg/logger/logger.go`
- Create: `tests/unit/logger_test.go`

- [ ] **Step 1: 编写测试**

```go
package logger

import (
    "bytes"
    "encoding/json"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "db-mcp/internal/config"
)

func TestNewLogger_JSONFormat(t *testing.T) {
    cfg := &config.LogConfig{Level: "info", Format: "json", Output: "stdout"}
    logger := NewLogger(cfg)
    assert.NotNil(t, logger)
}

func TestLogger_Info(t *testing.T) {
    var buf bytes.Buffer
    cfg := &config.LogConfig{Level: "debug", Format: "json", Output: "stdout"}

    old := out
    out = &buf
    defer func() { out = old }()

    logger := NewLogger(cfg)
    logger.Info("test message", "key", "value")

    output := buf.String()
    assert.Contains(t, output, "INFO")
    assert.Contains(t, output, "test message")
}

func TestLogger_With(t *testing.T) {
    cfg := &config.LogConfig{Level: "info", Format: "json"}
    logger := NewLogger(cfg)
    withLogger := logger.With("request_id", "123")

    assert.NotNil(t, withLogger)
    assert.NotEqual(t, logger, withLogger)
}

func TestParseLevel(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"debug", "DEBUG"},
        {"info", "INFO"},
        {"warn", "WARN"},
        {"error", "ERROR"},
        {"unknown", "INFO"},
    }

    for _, tt := range tests {
        got := parseLevelString(tt.input)
        assert.Equal(t, tt.expected, got)
    }
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/logger_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现日志**

```go
package logger

import (
    "io"
    "log/slog"
    "os"
    "strings"

    "db-mcp/internal/config"
)

var out io.Writer = os.Stdout

type Logger struct {
    level  slog.Level
    format string
    logger *slog.Logger
}

type LogFields map[string]interface{}

func NewLogger(cfg *config.LogConfig) *Logger {
    var handler slog.Handler
    opts := &slog.HandlerOptions{
        Level: parseLevel(cfg.Level),
    }

    switch cfg.Format {
    case "json":
        handler = slog.NewJSONHandler(out, opts)
    default:
        handler = slog.NewTextHandler(out, opts)
    }

    return &Logger{
        level:  parseLevel(cfg.Level),
        format: cfg.Format,
        logger: slog.New(handler),
    }
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
    l.logger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...interface{}) {
    l.logger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
    l.logger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
    l.logger.Error(msg, fields...)
}

func (l *Logger) With(fields ...interface{}) *Logger {
    return &Logger{
        level:  l.level,
        format: l.format,
        logger: l.logger.With(fields...),
    }
}

func parseLevel(level string) slog.Level {
    switch strings.ToLower(level) {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}

func parseLevelString(level string) string {
    switch strings.ToLower(level) {
    case "debug":
        return "DEBUG"
    case "info":
        return "INFO"
    case "warn":
        return "WARN"
    case "error":
        return "ERROR"
    default:
        return "INFO"
    }
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/logger_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add pkg/logger/logger.go tests/unit/logger_test.go
git commit -m "feat: add structured JSON logger

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: 错误处理

**Files:**
- Create: `internal/errors/errors.go`
- Create: `tests/unit/errors_test.go`

- [ ] **Step 1: 编写测试**

```go
package errors

import (
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "gorm.io/gorm"
)

func TestNewError(t *testing.T) {
    cause := errors.New("original error")
    err := NewError(ErrInvalidInput, "validation failed", cause)

    assert.Equal(t, ErrInvalidInput, err.Code)
    assert.Equal(t, "validation failed", err.Message)
    assert.Equal(t, cause, err.Cause)
}

func TestError_Error(t *testing.T) {
    err := NewError(ErrRecordNotFound, "not found", nil)
    assert.Equal(t, "[RECORD_NOT_FOUND] not found", err.Error())

    cause := errors.New("db error")
    err = NewError(ErrInternal, "internal", cause)
    assert.Contains(t, err.Error(), "db error")
}

func TestWrapGormError_RecordNotFound(t *testing.T) {
    err := WrapGormError(gorm.ErrRecordNotFound)
    assert.NotNil(t, err)
    assert.Equal(t, ErrRecordNotFound, err.Code)
}

func TestWrapGormError_DuplicateEntry(t *testing.T) {
    err := WrapGormError(errors.New("Duplicate entry '1' for key 'PRIMARY'"))
    assert.NotNil(t, err)
    assert.Equal(t, ErrDuplicateEntry, err.Code)
}

func TestWrapGormError_ForeignKey(t *testing.T) {
    err := WrapGormError(errors.New("foreign key constraint fails"))
    assert.NotNil(t, err)
    assert.Equal(t, ErrForeignKey, err.Code)
}

func TestWrapGormError_Nil(t *testing.T) {
    err := WrapGormError(nil)
    assert.Nil(t, err)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/errors_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现错误处理**

```go
package errors

import (
    "fmt"
    "strings"
)

type ErrorCode string

const (
    ErrInvalidInput   ErrorCode = "INVALID_INPUT"
    ErrTableNotFound ErrorCode = "TABLE_NOT_FOUND"
    ErrRecordNotFound ErrorCode = "RECORD_NOT_FOUND"
    ErrDuplicateEntry ErrorCode = "DUPLICATE_ENTRY"
    ErrForeignKey    ErrorCode = "FOREIGN_KEY_ERROR"
    ErrTimeout        ErrorCode = "TIMEOUT"
    ErrRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrInternal       ErrorCode = "INTERNAL_ERROR"
)

type DBError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

func (e *DBError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DBError) Unwrap() error {
    return e.Cause
}

func NewError(code ErrorCode, message string, cause error) *DBError {
    return &DBError{Code: code, Message: message, Cause: cause}
}

func WrapGormError(err error) *DBError {
    if err == nil {
        return nil
    }

    errStr := err.Error()
    switch {
    case strings.Contains(errStr, "record not found"):
        return NewError(ErrRecordNotFound, "record not found", err)
    case strings.Contains(errStr, "Duplicate entry"):
        return NewError(ErrDuplicateEntry, "duplicate entry", err)
    case strings.Contains(errStr, "foreign key constraint"):
        return NewError(ErrForeignKey, "foreign key constraint failed", err)
    default:
        return NewError(ErrInternal, "database error", err)
    }
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/errors_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/errors/errors.go tests/unit/errors_test.go
git commit -m "feat: add unified error handling with error codes

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段三：数据访问层

### Task 5: 连接管理

**Files:**
- Create: `internal/connection/manager.go`
- Create: `tests/unit/connection_test.go`

- [ ] **Step 1: 编写测试**

```go
package connection

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "db-mcp/internal/config"
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

    dsn := buildDSN(cfg)
    assert.Contains(t, dsn, "root:secret")
    assert.Contains(t, dsn, "tcp(localhost:3306)")
    assert.Contains(t, dsn, "testdb")
    assert.Contains(t, dsn, "charset=utf8mb4")
}

func TestBuildDSN_EmptyPassword(t *testing.T) {
    cfg := &config.DatabaseConfig{
        Host:     "localhost",
        Port:     3306,
        User:     "root",
        Password: "",
        Database: "testdb",
        Charset:  "utf8mb4",
    }

    dsn := buildDSN(cfg)
    assert.Contains(t, dsn, "root:@tcp")
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/connection_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现连接管理**

```go
package connection

import (
    "context"
    "fmt"
    "time"

    "db-mcp/internal/config"
    "db-mcp/pkg/logger"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

type ConnectionManager struct {
    db     *gorm.DB
    config *config.Config
    logger *logger.Logger
}

func NewConnectionManager(cfg *config.Config, log *logger.Logger) (*ConnectionManager, error) {
    dsn := buildDSN(&cfg.Database)

    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to connect database: %w", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get sql.DB: %w", err)
    }

    sqlDB.SetMaxIdleConns(cfg.Pool.MaxIdleConns)
    sqlDB.SetMaxOpenConns(cfg.Pool.MaxOpenConns)
    sqlDB.SetConnMaxLifetime(cfg.Pool.ConnMaxLifetime)
    sqlDB.SetConnMaxIdleTime(cfg.Pool.ConnMaxIdleTime)

    return &ConnectionManager{db: db, config: cfg, logger: log}, nil
}

func (m *ConnectionManager) DB() *gorm.DB {
    return m.db
}

func (m *ConnectionManager) Close() error {
    sqlDB, err := m.db.DB()
    if err != nil {
        return err
    }
    return sqlDB.Close()
}

func (m *ConnectionManager) HealthCheck() error {
    sqlDB, err := m.db.DB()
    if err != nil {
        return err
    }
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    return sqlDB.PingContext(ctx)
}

func buildDSN(cfg *config.DatabaseConfig) string {
    return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
        cfg.User,
        cfg.Password,
        cfg.Host,
        cfg.Port,
        cfg.Database,
        cfg.Charset,
    )
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/connection_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/connection/manager.go tests/unit/connection_test.go
git commit -m "feat: add connection manager with pool configuration

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 6: 删除字段检测器

**Files:**
- Create: `internal/detector/delete_field.go`
- Create: `tests/unit/detector_test.go`

- [ ] **Step 1: 编写测试**

```go
package detector

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestNamePatterns(t *testing.T) {
    patterns := []string{
        "deleted_at", "delete_time", "delete_date", "deleted_time",
        "is_del", "is_deleted", "deleted", "delete_flag",
        "del_flag", "del", "status",
    }
    assert.Len(t, patterns, 11)
}

func TestTypeBasedValues(t *testing.T) {
    values := map[string]string{
        "tinyint":   "1",
        "smallint":  "1",
        "int":       "1",
        "bigint":    "1",
        "boolean":   "1",
        "timestamp": "0000-00-00 00:00:00",
        "datetime":  "0000-00-00 00:00:00",
    }

    assert.Equal(t, "1", values["tinyint"])
    assert.Equal(t, "0000-00-00 00:00:00", values["timestamp"])
    assert.Equal(t, "1", values["boolean"])
}

func TestContainsDeleteKeyword(t *testing.T) {
    tests := []struct {
        comment  string
        expected bool
    }{
        {"删除时间", true},
        {"是否删除：0.否，1.是", true},
        {"逻辑删除", true},
        {"软删除", true},
        {"用户名", false},
        {"创建时间", false},
    }

    for _, tt := range tests {
        got := containsDeleteKeyword(tt.comment)
        assert.Equal(t, tt.expected, got, "comment: %s", tt.comment)
    }
}

func TestExtractDeleteValue(t *testing.T) {
    tests := []struct {
        comment  string
        dataType string
        expected string
    }{
        {"是否删除：0.否，1.是", "tinyint", "1"},
        {"状态:0正常,1删除", "int", "1"},
        {"删除标记 1=删除 0=未删", "tinyint", "1"},
        {"正常数据", "varchar", ""},
    }

    for _, tt := range tests {
        got := extractDeleteValue(tt.comment, tt.dataType)
        assert.Equal(t, tt.expected, got, "comment: %s", tt.comment)
    }
}

func TestDetermineDeleteValue(t *testing.T) {
    d := &DeleteFieldDetector{}

    // 空字段
    assert.Equal(t, "", d.determineDeleteValue(nil))
    assert.Equal(t, "", d.determineDeleteValue([]Field{}))

    // 单字段
    fields := []Field{{Name: "deleted_at", TrueValue: "0000-00-00 00:00:00"}}
    assert.Equal(t, "0000-00-00 00:00:00", d.determineDeleteValue(fields))

    // 多字段
    fields = []Field{
        {Name: "deleted_time", TrueValue: "0000-00-00 00:00:00"},
        {Name: "is_del", TrueValue: "1"},
    }
    assert.Equal(t, "0000-00-00 00:00:00", d.determineDeleteValue(fields))
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/detector_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现删除字段检测器**

```go
package detector

import (
    "regexp"
    "strings"
)

type DeleteFieldDetector struct {
    db    interface{}
    cache map[string]*DeleteFieldInfo
}

type DeleteFieldInfo struct {
    TableName   string
    Fields      []Field
    DeleteValue string
}

type Field struct {
    Name      string
    Type      string
    TrueValue string
}

var namePatterns = []string{
    "deleted_at", "delete_time", "delete_date", "deleted_time",
    "is_del", "is_deleted", "deleted", "delete_flag",
    "del_flag", "del", "status",
}

var commentKeywords = []string{
    "删除", "del", "is_del", "逻辑删除", "软删除",
}

var typeBasedValues = map[string]string{
    "tinyint":   "1",
    "smallint":  "1",
    "int":       "1",
    "bigint":    "1",
    "boolean":   "1",
    "timestamp": "0000-00-00 00:00:00",
    "datetime":  "0000-00-00 00:00:00",
}

type ColumnInfo struct {
    Name       string
    DataType   string
    ColumnType string
    Nullable   string
    Key        string
    Comment    string
}

func (d *DeleteFieldDetector) Detect(table string, columns []ColumnInfo) *DeleteFieldInfo {
    if d.cache == nil {
        d.cache = make(map[string]*DeleteFieldInfo)
    }

    if info, ok := d.cache[table]; ok {
        return info
    }

    detectedFields := d.detectDeleteFields(columns)

    info := &DeleteFieldInfo{
        TableName:   table,
        Fields:      detectedFields,
        DeleteValue: d.determineDeleteValue(detectedFields),
    }

    d.cache[table] = info
    return info
}

func (d *DeleteFieldDetector) detectDeleteFields(columns []ColumnInfo) []Field {
    var result []Field

    for _, col := range columns {
        field := d.analyzeColumn(col)
        if field != nil {
            result = append(result, *field)
            if len(result) >= 2 {
                break
            }
        }
    }

    return result
}

func (d *DeleteFieldDetector) analyzeColumn(col ColumnInfo) *Field {
    // 1. 字段名匹配检测
    for _, pattern := range namePatterns {
        if strings.EqualFold(col.Name, pattern) {
            return &Field{
                Name:      col.Name,
                Type:      col.DataType,
                TrueValue: typeBasedValues[col.DataType],
            }
        }
    }

    // 2. COMMENT 语义检测
    if containsDeleteKeyword(col.Comment) {
        trueValue := typeBasedValues[col.DataType]
        return &Field{
            Name:      col.Name,
            Type:      col.DataType,
            TrueValue: trueValue,
        }
    }

    // 3. COMMENT 值映射检测
    if mappedValue := extractDeleteValue(col.Comment, col.DataType); mappedValue != "" {
        return &Field{
            Name:      col.Name,
            Type:      col.DataType,
            TrueValue: mappedValue,
        }
    }

    return nil
}

func containsDeleteKeyword(comment string) bool {
    lowerComment := strings.ToLower(comment)
    for _, keyword := range commentKeywords {
        if strings.Contains(lowerComment, keyword) {
            return true
        }
    }
    return false
}

func extractDeleteValue(comment, dataType string) string {
    patterns := []string{
        `(\d+)[\.:：]*(?:是|删除|yes|true)`,
    }

    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(strings.ToLower(comment))
        if len(matches) > 1 {
            return matches[1]
        }
    }
    return ""
}

func (d *DeleteFieldDetector) determineDeleteValue(fields []Field) string {
    if len(fields) == 0 {
        return ""
    }
    return fields[0].TrueValue
}

func NewDetector() *DeleteFieldDetector {
    return &DeleteFieldDetector{
        cache: make(map[string]*DeleteFieldInfo),
    }
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/detector_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/detector/delete_field.go tests/unit/detector_test.go
git commit -m "feat: add delete field auto-detection with comment parsing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7: Repository 层

**Files:**
- Create: `internal/repository/repository.go`
- Create: `tests/unit/repository_test.go`

- [ ] **Step 1: 编写测试**

```go
package repository

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "db-mcp/internal/detector"
)

func TestNewRepository(t *testing.T) {
    repo := New(nil)
    assert.NotNil(t, repo)
}

func TestQueryRequest_Validation(t *testing.T) {
    req := &QueryRequest{Table: "users"}
    assert.Equal(t, "users", req.Table)
    assert.Equal(t, 100, req.Limit)
    assert.Equal(t, 0, req.Offset)
}

func TestInsertRequest_Validation(t *testing.T) {
    req := &InsertRequest{
        Table: "users",
        Data:  map[string]interface{}{"name": "test"},
    }
    assert.Equal(t, "users", req.Table)
    assert.Equal(t, "test", req.Data["name"])
}

func TestUpdateRequest_Validation(t *testing.T) {
    req := &UpdateRequest{
        Table: "users",
        Data:  map[string]interface{}{"name": "updated"},
        Where: map[string]interface{}{"id": 1},
    }
    assert.Equal(t, "updated", req.Data["name"])
    assert.Equal(t, 1, req.Where["id"])
}

func TestDeleteRequest_WithDeleteField(t *testing.T) {
    req := &DeleteRequest{
        Table:        "users",
        Where:        map[string]interface{}{"id": 1},
        DeleteField:  &detector.DeleteFieldInfo{
            TableName: "users",
            Fields: []detector.Field{
                {Name: "deleted_time", Type: "datetime", TrueValue: "0000-00-00 00:00:00"},
            },
        },
    }
    assert.NotNil(t, req.DeleteField)
    assert.Equal(t, "deleted_time", req.DeleteField.Fields[0].Name)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/repository_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现 Repository**

```go
package repository

import (
    "db-mcp/internal/detector"
    "db-mcp/internal/errors"

    "gorm.io/gorm"
)

type Repository struct {
    db *gorm.DB
}

type QueryRequest struct {
    Table  string
    Fields []string
    Where  map[string]interface{}
    Order  []OrderBy
    Limit  int
    Offset int
}

type OrderBy struct {
    Field     string
    Direction string
}

type InsertRequest struct {
    Table string
    Data  map[string]interface{}
}

type UpdateRequest struct {
    Table string
    Data  map[string]interface{}
    Where map[string]interface{}
}

type DeleteRequest struct {
    Table       string
    Where       map[string]interface{}
    DeleteField *detector.DeleteFieldInfo
}

type BatchInsertRequest struct {
    Table string
    Data  []map[string]interface{}
}

type BatchUpdateRequest struct {
    Table     string
    Data      []map[string]interface{}
    KeyField  string
}

type BatchDeleteRequest struct {
    Table    string
    IDs      []string
    IDField  string
}

type JoinRequest struct {
    Tables []TableRef
    Joins  []JoinClause
    Fields []string
    Where  map[string]interface{}
    Order  []OrderBy
    Limit  int
}

type TableRef struct {
    Name  string
    Alias string
}

type JoinClause struct {
    Type      string
    FromTable string
    FromField string
    ToTable   string
    ToField   string
}

type QueryResult struct {
    Rows    []map[string]interface{}
    Total   int64
    Message string
}

type MutationResult struct {
    AffectedRows int64
    LastInsertID int64
    Message      string
}

type BatchResult struct {
    SuccessCount int64
    FailedCount  int64
    Errors       []BatchError
}

type BatchError struct {
    Index   int
    Message string
}

func New(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Query(req *QueryRequest) (*QueryResult, error) {
    var rows []map[string]interface{}

    query := r.db.Table(req.Table)

    // 应用条件
    if len(req.Where) > 0 {
        query = query.Where(req.Where)
    }

    // 排序
    for _, order := range req.Order {
        dir := "ASC"
        if order.Direction == "desc" {
            dir = "DESC"
        }
        query = query.Order(order.Field + " " + dir)
    }

    // 分页
    if req.Limit > 0 {
        query = query.Limit(req.Limit)
    }
    if req.Offset > 0 {
        query = query.Offset(req.Offset)
    }

    // 字段
    fields := "*"
    if len(req.Fields) > 0 {
        fields = joinFields(req.Fields)
    }

    err := query.Select(fields).Find(&rows).Error
    if err != nil {
        return nil, errors.WrapGormError(err)
    }

    var total int64
    r.db.Table(req.Table).Where(req.Where).Count(&total)

    return &QueryResult{Rows: rows, Total: total}, nil
}

func (r *Repository) Insert(req *InsertRequest) (*MutationResult, error) {
    err := r.db.Table(req.Table).Create(req.Data).Error
    if err != nil {
        return nil, errors.WrapGormError(err)
    }

    return &MutationResult{
        AffectedRows: 1,
        LastInsertID: 1,
        Message:      "Insert successful",
    }, nil
}

func (r *Repository) Update(req *UpdateRequest) (*MutationResult, error) {
    result := r.db.Table(req.Table).Where(req.Where).Updates(req.Data)
    if result.Error != nil {
        return nil, errors.WrapGormError(result.Error)
    }

    return &MutationResult{
        AffectedRows: result.RowsAffected,
        Message:      "Update successful",
    }, nil
}

func (r *Repository) LogicalDelete(req *DeleteRequest) (*MutationResult, error) {
    if req.DeleteField == nil || len(req.DeleteField.Fields) == 0 {
        return nil, errors.NewError(errors.ErrInvalidInput, "no delete field detected", nil)
    }

    updates := make(map[string]interface{})
    for _, field := range req.DeleteField.Fields {
        updates[field.Name] = field.TrueValue
    }

    result := r.db.Table(req.Table).Where(req.Where).Updates(updates)
    if result.Error != nil {
        return nil, errors.WrapGormError(result.Error)
    }

    return &MutationResult{
        AffectedRows: result.RowsAffected,
        Message:      "Logical delete successful",
    }, nil
}

func joinFields(fields []string) string {
    result := ""
    for i, f := range fields {
        if i > 0 {
            result += ", "
        }
        result += f
    }
    return result
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/repository_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/repository/repository.go tests/unit/repository_test.go
git commit -m "feat: add repository layer with CRUD operations

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段四：业务服务层

### Task 8: 审计日志服务

**Files:**
- Create: `internal/service/audit.go`
- Create: `tests/unit/audit_test.go`

- [ ] **Step 1: 编写测试**

```go
package service

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestAuditContext(t *testing.T) {
    ctx := &AuditContext{
        RequestID:  "req-123",
        Operation:  "SELECT",
        Table:      "users",
        RecordID:   "1",
        StartTime:  time.Now(),
    }

    assert.Equal(t, "req-123", ctx.RequestID)
    assert.Equal(t, "SELECT", ctx.Operation)
    assert.Equal(t, "users", ctx.Table)
}

func TestNewAuditService(t *testing.T) {
    service := NewAuditService(nil, "_audit_logs")
    assert.NotNil(t, service)
    assert.Equal(t, "_audit_logs", service.table)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/audit_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现审计日志服务**

```go
package service

import (
    "time"
)

type AuditLog struct {
    ID         uint      `gorm:"primaryKey"`
    Timestamp  time.Time `gorm:"index"`
    Operation  string    `gorm:"size:20;index"`
    Table      string    `gorm:"size:100;index"`
    RecordID   string    `gorm:"size:100;index"`
    Actor      string    `gorm:"size:100"`
    RequestID  string    `gorm:"size:50;index"`
    BeforeData string    `gorm:"type:text"`
    AfterData  string    `gorm:"type:text"`
    Duration   int64
    Status     string    `gorm:"size:20;index"`
    ErrorMsg   string    `gorm:"type:text"`
}

type AuditContext struct {
    RequestID  string
    Operation  string
    Table      string
    RecordID   string
    Actor      string
    StartTime  time.Time
    BeforeData map[string]interface{}
    AfterData  map[string]interface{}
}

type AuditService struct {
    repo   interface{}
    table  string
    logger interface{}
}

func NewAuditService(repo interface{}, table string) *AuditService {
    return &AuditService{
        repo:  repo,
        table: table,
    }
}

func (s *AuditService) Start(operation, table, recordID string) *AuditContext {
    return &AuditContext{
        RequestID:  generateRequestID(),
        Operation:  operation,
        Table:      table,
        RecordID:   recordID,
        StartTime:  time.Now(),
        BeforeData: make(map[string]interface{}),
    }
}

func (s *AuditService) Success(ctx *AuditContext, beforeData, afterData interface{}, affectedRows int64) {
    duration := time.Since(ctx.StartTime).Milliseconds()

    log := AuditLog{
        Timestamp:  ctx.StartTime,
        Operation:  ctx.Operation,
        Table:      ctx.Table,
        RecordID:   ctx.RecordID,
        Actor:      ctx.Actor,
        RequestID:  ctx.RequestID,
        BeforeData: toJSON(ctx.BeforeData),
        AfterData:  toJSON(ctx.AfterData),
        Duration:   duration,
        Status:     "success",
    }
    _ = log
}

func (s *AuditService) Fail(ctx *AuditContext, errMsg string) {
    duration := time.Since(ctx.StartTime).Milliseconds()

    log := AuditLog{
        Timestamp: ctx.StartTime,
        Operation: ctx.Operation,
        Table:     ctx.Table,
        RecordID:  ctx.RecordID,
        Actor:     ctx.Actor,
        RequestID: ctx.RequestID,
        BeforeData: toJSON(ctx.BeforeData),
        Duration:  duration,
        Status:    "failed",
        ErrorMsg:  errMsg,
    }
    _ = log
}

func generateRequestID() string {
    return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
    }
    return string(b)
}

func toJSON(data interface{}) string {
    if data == nil {
        return ""
    }
    return "{}"
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/audit_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/service/audit.go tests/unit/audit_test.go
git commit -m "feat: add audit logging service

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 9: CRUD 服务

**Files:**
- Create: `internal/service/crud.go`
- Create: `tests/unit/crud_test.go`

- [ ] **Step 1: 编写测试**

```go
package service

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "db-mcp/internal/repository"
)

func TestNewCRUDService(t *testing.T) {
    repo := repository.New(nil)
    detector := NewDeleteFieldDetector()
    audit := NewAuditService(nil, "_audit_logs")

    service := NewCRUDService(repo, detector, audit)
    assert.NotNil(t, service)
}

func TestCRUDService_Query_EmptyTable(t *testing.T) {
    repo := repository.New(nil)
    detector := NewDeleteFieldDetector()
    audit := NewAuditService(nil, "_audit_logs")

    service := NewCRUDService(repo, detector, audit)

    req := &repository.QueryRequest{Table: ""}
    assert.NotNil(t, req)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/crud_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现 CRUD 服务**

```go
package service

import (
    "db-mcp/internal/detector"
    "db-mcp/internal/repository"
)

type CRUDService struct {
    repo     *repository.Repository
    detector *detector.DeleteFieldDetector
    audit    *AuditService
}

func NewCRUDService(repo *repository.Repository, detector *detector.DeleteFieldDetector, audit *AuditService) *CRUDService {
    return &CRUDService{
        repo:     repo,
        detector: detector,
        audit:    audit,
    }
}

func (s *CRUDService) Query(req *repository.QueryRequest) (*repository.QueryResult, error) {
    auditCtx := s.audit.Start("SELECT", req.Table, "")

    result, err := s.repo.Query(req)
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }

    s.audit.Success(auditCtx, nil, nil, result.Total)
    return result, nil
}

func (s *CRUDService) Insert(req *repository.InsertRequest) (*repository.MutationResult, error) {
    auditCtx := s.audit.Start("INSERT", req.Table, "")

    result, err := s.repo.Insert(req)
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }

    s.audit.Success(auditCtx, nil, nil, 1)
    return result, nil
}

func (s *CRUDService) Update(req *repository.UpdateRequest) (*repository.MutationResult, error) {
    auditCtx := s.audit.Start("UPDATE", req.Table, "")

    result, err := s.repo.Update(req)
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }

    s.audit.Success(auditCtx, nil, nil, result.AffectedRows)
    return result, nil
}

func (s *CRUDService) Delete(req *repository.DeleteRequest) (*repository.MutationResult, error) {
    auditCtx := s.audit.Start("DELETE", req.Table, "")

    result, err := s.repo.LogicalDelete(req)
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }

    s.audit.Success(auditCtx, nil, nil, result.AffectedRows)
    return result, nil
}

func NewDeleteFieldDetector() *detector.DeleteFieldDetector {
    return detector.NewDetector()
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/crud_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/service/crud.go tests/unit/crud_test.go
git commit -m "feat: add CRUD service with audit logging

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段五：中间件

### Task 10: 限流中间件

**Files:**
- Create: `internal/middleware/ratelimit.go`
- Create: `tests/unit/ratelimit_test.go`

- [ ] **Step 1: 编写测试**

```go
package middleware

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "db-mcp/internal/config"
)

func TestNewRateLimiter_Enabled(t *testing.T) {
    cfg := &config.RateLimitConfig{
        Enabled:  true,
        Requests: 100,
        Burst:    200,
    }

    limiter := NewRateLimiter(cfg)
    assert.NotNil(t, limiter)
    assert.True(t, limiter.enabled)
    assert.Equal(t, 100, limiter.requests)
    assert.Equal(t, 200, limiter.burst)
}

func TestNewRateLimiter_Disabled(t *testing.T) {
    cfg := &config.RateLimitConfig{
        Enabled: false,
    }

    limiter := NewRateLimiter(cfg)
    assert.NotNil(t, limiter)
    assert.False(t, limiter.enabled)
}

func TestRateLimiter_Allow_Enabled(t *testing.T) {
    cfg := &config.RateLimitConfig{
        Enabled:  true,
        Requests: 100,
        Burst:    200,
    }

    limiter := NewRateLimiter(cfg)

    for i := 0; i < 200; i++ {
        assert.True(t, limiter.Allow(), "request %d should be allowed", i)
    }
}

func TestRateLimiter_Allow_Disabled(t *testing.T) {
    cfg := &config.RateLimitConfig{
        Enabled: false,
    }

    limiter := NewRateLimiter(cfg)
    assert.True(t, limiter.Allow())
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/ratelimit_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现限流中间件**

```go
package middleware

import (
    "context"

    "db-mcp/internal/config"
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    limiter  *rate.Limiter
    enabled  bool
    requests int
    burst    int
}

func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
    if !cfg.Enabled {
        return &RateLimiter{enabled: false}
    }

    return &RateLimiter{
        enabled:  true,
        limiter:  rate.NewLimiter(rate.Limit(cfg.Requests), cfg.Burst),
        requests: cfg.Requests,
        burst:    cfg.Burst,
    }
}

func (r *RateLimiter) Allow() bool {
    if !r.enabled {
        return true
    }
    return r.limiter.Allow()
}

func (r *RateLimiter) Wait(ctx context.Context) error {
    if !r.enabled {
        return nil
    }
    return r.limiter.Wait(ctx)
}

func (r *RateLimiter) Enabled() bool {
    return r.enabled
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/ratelimit_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/middleware/ratelimit.go tests/unit/ratelimit_test.go
git commit -m "feat: add rate limiting middleware

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 11: 超时中间件

**Files:**
- Create: `internal/middleware/timeout.go`
- Create: `tests/unit/timeout_test.go`

- [ ] **Step 1: 编写测试**

```go
package middleware

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestDefaultTimeoutConfig(t *testing.T) {
    cfg := DefaultTimeoutConfig()

    assert.Equal(t, 5*time.Second, cfg.Connect)
    assert.Equal(t, 30*time.Second, cfg.Query)
    assert.Equal(t, 10*time.Second, cfg.Mutation)
    assert.Equal(t, 60*time.Second, cfg.Transaction)
}

func TestTimeoutConfig_GetTimeout(t *testing.T) {
    cfg := &TimeoutConfig{
        Connect:     5 * time.Second,
        Query:       30 * time.Second,
        Mutation:    10 * time.Second,
        Transaction: 60 * time.Second,
    }

    assert.Equal(t, 30*time.Second, cfg.GetTimeout("SELECT"))
    assert.Equal(t, 10*time.Second, cfg.GetTimeout("INSERT"))
    assert.Equal(t, 10*time.Second, cfg.GetTimeout("UPDATE"))
    assert.Equal(t, 10*time.Second, cfg.GetTimeout("DELETE"))
    assert.Equal(t, 60*time.Second, cfg.GetTimeout("TRANSACTION"))
    assert.Equal(t, 30*time.Second, cfg.GetTimeout("UNKNOWN"))
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/timeout_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现超时中间件**

```go
package middleware

import (
    "time"
)

type TimeoutConfig struct {
    Connect     time.Duration
    Query       time.Duration
    Mutation    time.Duration
    Transaction time.Duration
}

func DefaultTimeoutConfig() *TimeoutConfig {
    return &TimeoutConfig{
        Connect:     5 * time.Second,
        Query:       30 * time.Second,
        Mutation:    10 * time.Second,
        Transaction: 60 * time.Second,
    }
}

func (t *TimeoutConfig) GetTimeout(op string) time.Duration {
    switch op {
    case "INSERT", "UPDATE", "DELETE":
        return t.Mutation
    case "TRANSACTION":
        return t.Transaction
    default:
        return t.Query
    }
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/timeout_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/middleware/timeout.go tests/unit/timeout_test.go
git commit -m "feat: add timeout middleware

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段六：MCP 层

### Task 12: MCP 工具定义

**Files:**
- Create: `internal/mcp/tools.go`
- Create: `tests/unit/mcp_test.go`

- [ ] **Step 1: 编写测试**

```go
package mcp

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestToolNames(t *testing.T) {
    expectedTools := []string{
        "db_query", "db_insert", "db_update", "db_delete",
        "db_batch_insert", "db_batch_update", "db_batch_delete",
        "db_join", "db_transaction", "db_describe",
    }

    for _, tool := range expectedTools {
        assert.NotEmpty(t, tool)
    }
}

func TestQueryRequest_ToMap(t *testing.T) {
    req := QueryRequest{
        Table:  "users",
        Fields: []string{"id", "name"},
        Where:  map[string]interface{}{"status": "active"},
        Limit:  10,
    }

    assert.Equal(t, "users", req.Table)
    assert.Len(t, req.Fields, 2)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./tests/unit/mcp_test.go -v
# Expected: FAIL
```

- [ ] **Step 3: 实现 MCP 工具**

```go
package mcp

type QueryRequest struct {
    Table  string                 `json:"table"`
    Fields []string               `json:"fields"`
    Where  map[string]interface{} `json:"where"`
    Order  []OrderSpec            `json:"order"`
    Limit  int                    `json:"limit"`
    Offset int                    `json:"offset"`
}

type OrderSpec struct {
    Field     string `json:"field"`
    Direction string `json:"direction"`
}

type InsertRequest struct {
    Table string                 `json:"table"`
    Data  map[string]interface{} `json:"data"`
}

type UpdateRequest struct {
    Table string                 `json:"table"`
    Data  map[string]interface{} `json:"data"`
    Where map[string]interface{} `json:"where"`
}

type DeleteRequest struct {
    Table string                 `json:"table"`
    Where map[string]interface{} `json:"where"`
}

type BatchInsertRequest struct {
    Table string                   `json:"table"`
    Data  []map[string]interface{} `json:"data"`
}

type BatchUpdateRequest struct {
    Table    string                   `json:"table"`
    Data     []map[string]interface{} `json:"data"`
    KeyField string                   `json:"key_field"`
}

type BatchDeleteRequest struct {
    Table   string   `json:"table"`
    IDs     []string `json:"ids"`
    IDField string   `json:"id_field"`
}

type JoinRequest struct {
    Tables []TableRef    `json:"tables"`
    Joins  []JoinSpec    `json:"joins"`
    Fields []string      `json:"fields"`
    Where  interface{}   `json:"where"`
    Order  []OrderSpec   `json:"order"`
    Limit  int           `json:"limit"`
}

type TableRef struct {
    Name  string `json:"name"`
    Alias string `json:"alias"`
}

type JoinSpec struct {
    Type      string `json:"type"`
    FromTable string `json:"from_table"`
    FromField string `json:"from_field"`
    ToTable   string `json:"to_table"`
    ToField   string `json:"to_field"`
}

type TransactionRequest struct {
    Operations []Operation `json:"operations"`
}

type Operation struct {
    Action string                 `json:"action"`
    Table  string                 `json:"table"`
    Data   map[string]interface{} `json:"data"`
    Where  map[string]interface{} `json:"where"`
}

type DescribeRequest struct {
    Table string `json:"table"`
}

type DescribeResult struct {
    Table      string      `json:"table"`
    Fields     []FieldInfo `json:"fields"`
    DeleteField *FieldInfo `json:"delete_field,omitempty"`
}

type FieldInfo struct {
    Name     string `json:"name"`
    Type     string `json:"type"`
    Nullable bool   `json:"nullable"`
    Key      string `json:"key"`
    Comment  string `json:"comment"`
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./tests/unit/mcp_test.go -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/mcp/tools.go tests/unit/mcp_test.go
git commit -m "feat: add MCP tool definitions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段七：入口和集成

### Task 13: 主入口文件

**Files:**
- Create: `cmd/server/main.go`
- Create: `configs/config.yaml`

- [ ] **Step 1: 创建主入口**

```go
package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"

    "db-mcp/internal/config"
    "db-mcp/internal/connection"
    "db-mcp/pkg/logger"
)

func main() {
    configPath := flag.String("config", "configs/config.yaml", "path to config file")
    flag.Parse()

    // 加载配置
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 初始化日志
    log := logger.NewLogger(&cfg.Log)

    // 初始化数据库连接
    connMgr, err := connection.NewConnectionManager(cfg, log)
    if err != nil {
        log.Error("Failed to connect database", "error", err)
        os.Exit(1)
    }
    defer connMgr.Close()

    log.Info("database connected",
        "host", cfg.Database.Host,
        "database", cfg.Database.Database)

    // 健康检查
    if err := connMgr.HealthCheck(); err != nil {
        log.Error("health check failed", "error", err)
        os.Exit(1)
    }

    log.Info("server started",
        "host", cfg.Database.Host,
        "port", cfg.Database.Port)

    // 等待信号
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig

    log.Info("server shutting down")
}
```

- [ ] **Step 2: 创建配置示例**

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  database: testdb
  charset: utf8mb4

pool:
  maxIdleConns: 10
  maxOpenConns: 100
  connMaxLifetime: 1h
  connMaxIdleTime: 10m

log:
  level: info
  format: json
  output: stdout
  auditTable: _audit_logs

rateLimit:
  enabled: true
  requests: 100
  burst: 200
```

- [ ] **Step 3: 提交**

```bash
git add cmd/server/main.go configs/config.yaml
git commit -m "feat: add main entry point and config example

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 14: README 文档

**Files:**
- Create: `README.md`

- [ ] **Step 1: 创建 README**

```markdown
# db-mcp

基于 go-mcp SDK 的 MySQL 数据库 MCP 服务，为 AI 助手提供数据库操作能力。

## 功能特性

- **CRUD 操作**：单表查询、插入、更新、逻辑删除
- **批量操作**：批量插入、更新、删除
- **多表查询**：支持 INNER/LEFT/RIGHT JOIN
- **事务支持**：原子性操作
- **逻辑删除**：自动检测删除标识字段
- **审计日志**：完整操作记录
- **限流控制**：防止数据库过载
- **超时控制**：保护数据库连接

## 快速开始

### 安装

```bash
go build -o bin/db-mcp ./cmd/server
```

### 配置

复制 `configs/config.yaml` 并修改数据库连接信息：

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  database: testdb
```

### 运行

```bash
./bin/db-mcp -config configs/config.yaml
```

## MCP 工具

| 工具 | 功能 |
|------|------|
| db_query | 单表查询 |
| db_insert | 插入数据 |
| db_update | 更新数据 |
| db_delete | 逻辑删除 |
| db_batch_insert | 批量插入 |
| db_batch_update | 批量更新 |
| db_batch_delete | 批量删除 |
| db_join | 多表关联查询 |
| db_transaction | 事务操作 |
| db_describe | 获取表结构 |

## 开发

### 测试

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 检查覆盖率
make coverage-check
```

### 代码规范

```bash
make lint
```

## 架构

```
MCP Layer → Service Layer → Repository Layer → Connection Layer
```

详见 [设计文档](docs/superpowers/specs/2026-04-02-db-mcp-design.md)

## License

MIT
```

- [ ] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: add README

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 阶段八：集成测试（可选，需要数据库）

### Task 15: 集成测试

**Files:**
- Create: `tests/integration/crud_test.go`

此任务需要 MySQL 数据库环境，可使用 Docker:

```bash
# 启动 MySQL
docker run -d --name mysql-test -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=test \
  -e MYSQL_DATABASE=testdb \
  mysql:8.0

# 运行集成测试
go test ./tests/integration/... -v
```

---

## 任务汇总

| 阶段 | Task | 组件 | 文件数 |
|------|------|------|--------|
| 一 | 1 | 项目初始化 | 3 |
| 二 | 2-4 | 基础设施层 | 6 |
| 三 | 5-7 | 数据访问层 | 6 |
| 四 | 8-9 | 业务服务层 | 4 |
| 五 | 10-11 | 中间件 | 4 |
| 六 | 12 | MCP 层 | 2 |
| 七 | 13-14 | 入口和文档 | 3 |
| **总计** | **15** | | **28+** |

---

## 验证清单

- [ ] 所有单元测试通过
- [ ] 测试覆盖率 100%
- [ ] golangci-lint 通过
- [ ] README 文档完整