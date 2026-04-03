# db-mcp 设计文档

## 1. 概述

### 1.1 项目背景

db-mcp 是一个基于 go-mcp SDK 的 MySQL 数据库 MCP 服务，旨在为 AI 助手（如 Claude）提供数据库操作能力。

### 1.2 技术栈

- **SDK**: mark3labs/mcp-go
- **ORM**: GORM
- **数据库**: MySQL 8.0+
- **语言**: Go 1.21+

---

## 2. 架构设计

### 2.1 目录结构

```
db-mcp/
├── cmd/
│   └── server/
│       └── main.go              # 入口文件
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── connection/
│   │   └── manager.go           # 连接池管理
│   ├── repository/
│   │   └── repository.go        # GORM 数据访问层
│   ├── service/
│   │   ├── crud.go              # CRUD 业务逻辑
│   │   ├── transaction.go       # 事务管理
│   │   └── audit.go            # 审计日志
│   ├── detector/
│   │   └── delete_field.go      # 删除字段自动检测
│   ├── middleware/
│   │   ├── ratelimit.go         # 限流
│   │   └── timeout.go          # 超时控制
│   ├── mcp/
│   │   └── tools.go             # MCP 工具定义
│   └── errors/
│       └── errors.go            # 统一错误处理
├── pkg/
│   └── logger/
│       └── logger.go            # 结构化日志
├── configs/
│   └── config.yaml              # 默认配置文件
├── tests/
│   ├── unit/                    # 单元测试
│   ├── integration/              # 集成测试
│   └── e2e/                     # 端到端测试
├── docs/
│   └── specs/                   # 设计文档
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 2.2 分层职责

| 层级 | 职责 | 关键组件 |
|------|------|----------|
| **MCP Layer** | 暴露工具给 AI，参数验证 | `internal/mcp/tools.go` |
| **Service Layer** | 业务逻辑、审计、限流 | `internal/service/` |
| **Repository Layer** | GORM 封装、SQL 生成 | `internal/repository/` |
| **Connection Layer** | 连接池、配置管理 | `internal/connection/` |
| **Infrastructure** | 日志、配置、检测器、错误处理 | `pkg/logger`, `internal/detector/`, `internal/errors/` |

### 2.3 分层设计说明

**detector 独立设计的原因：**

1. **单一职责**：detector 只负责字段检测，职责清晰
2. **独立测试**：可以单独写测试，更容易达到 100% 覆盖率
3. **可复用**：如果其他层需要（如 connection 层初始化时检测），可以直接调用
4. **松耦合**：service 不需要知道检测的具体实现

---

## 3. 数据模型设计

### 3.1 配置模型

```go
type Config struct {
    Database  DatabaseConfig  `yaml:"database" json:"database"`
    MCP       MCPConfig       `yaml:"mcp" json:"mcp"`
    Log       LogConfig       `yaml:"log" json:"log"`
    RateLimit RateLimitConfig `yaml:"rateLimit" json:"rateLimit"`
    Pool      PoolConfig      `yaml:"pool" json:"pool"`
}
```

#### 配置默认值

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| Database.Host | localhost | 数据库主机 |
| Database.Port | 3306 | MySQL 默认端口 |
| Database.Charset | utf8mb4 | 支持完整 Unicode |
| Pool.MaxIdleConns | 10 | 空闲连接数 |
| Pool.MaxOpenConns | 100 | 最大打开连接数 |
| Pool.ConnMaxLifetime | 1h | 连接最大存活时间 |
| Pool.ConnMaxIdleTime | 10m | 空闲连接最大存活时间 |
| Log.Level | info | 日志级别 |
| Log.Format | json | 结构化日志格式 |
| Log.Output | stdout | 标准输出 |
| Log.AuditTable | _audit_logs | 审计日志表名 |
| RateLimit.Enabled | true | 默认开启限流 |
| RateLimit.Requests | 100 | 每秒请求数 |
| RateLimit.Burst | 200 | 突发容量 |

#### 必填字段

以下字段必须在配置文件或 MCP 参数中提供：
- `Database.User`
- `Database.Password`
- `Database.Database`

### 3.2 审计日志模型

```go
type AuditLog struct {
    ID          uint      `gorm:"primaryKey"`
    Timestamp   time.Time `gorm:"index"`
    Operation   string    `gorm:"size:20"`   // SELECT, INSERT, UPDATE, DELETE
    Table       string    `gorm:"size:100"`
    RecordID    string    `gorm:"size:100"`  // 操作的记录 ID
    Actor       string    `gorm:"size:100"`  // 操作者（AI 助手标识）
    RequestID   string    `gorm:"size:50"`   // 请求追踪 ID
    BeforeData  string    `gorm:"type:text"` // JSON: 操作前数据
    AfterData   string    `gorm:"type:text"` // JSON: 操作后数据
    Duration    int64     // 耗时（毫秒）
    Status      string    `gorm:"size:20"`   // success, failed
    ErrorMsg    string    `gorm:"type:text"`
}
```

### 3.3 通用响应模型

```go
type QueryResult struct {
    Rows    []map[string]interface{} `json:"rows"`
    Total   int64                    `json:"total"`
    Message string                   `json:"message,omitempty"`
}

type MutationResult struct {
    AffectedRows int64  `json:"affectedRows"`
    LastInsertID int64  `json:"lastInsertId,omitempty"`
    Message      string `json:"message,omitempty"`
}

type BatchResult struct {
    SuccessCount int64            `json:"successCount"`
    FailedCount  int64            `json:"failedCount"`
    Errors       []BatchError     `json:"errors,omitempty"`
}
```

---

## 4. MCP 工具接口设计

### 4.1 工具列表

| 工具名称 | 功能 | 参数 |
|----------|------|------|
| `db_query` | 单表查询 | table, fields, where, order, limit, offset |
| `db_insert` | 插入数据 | table, data |
| `db_update` | 更新数据 | table, data, where |
| `db_delete` | 逻辑删除 | table, where |
| `db_batch_insert` | 批量插入 | table, data[] |
| `db_batch_update` | 批量更新 | table, data[], key_field |
| `db_batch_delete` | 批量逻辑删除 | table, ids |
| `db_join` | 多表关联查询 | tables, joins, fields, where, order, limit |
| `db_transaction` | 事务操作 | operations[] |
| `db_describe` | 获取表结构 | table |

### 4.2 工具详细定义

#### db_query - 单表查询

```json
{
  "name": "db_query",
  "description": "查询单表数据，支持条件过滤、排序、分页",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string", "description": "表名" },
      "fields": { "type": "array", "items": { "type": "string" }, "default": ["*"] },
      "where": { "type": "object" },
      "order": { "type": "array" },
      "limit": { "type": "integer", "default": 100, "maximum": 1000 },
      "offset": { "type": "integer", "default": 0 }
    },
    "required": ["table"]
  }
}
```

#### db_insert - 插入数据

```json
{
  "name": "db_insert",
  "description": "向表中插入一条数据",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" },
      "data": { "type": "object" }
    },
    "required": ["table", "data"]
  }
}
```

#### db_update - 更新数据

```json
{
  "name": "db_update",
  "description": "更新表中符合条件的数据",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" },
      "data": { "type": "object" },
      "where": { "type": "object" }
    },
    "required": ["table", "data", "where"]
  }
}
```

#### db_delete - 逻辑删除

```json
{
  "name": "db_delete",
  "description": "逻辑删除表中符合条件的数据（自动检测删除标识字段）",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" },
      "where": { "type": "object" }
    },
    "required": ["table", "where"]
  }
}
```

#### db_batch_insert - 批量插入

```json
{
  "name": "db_batch_insert",
  "description": "批量插入数据，支持事务",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" },
      "data": { "type": "array", "items": { "type": "object" }, "maxItems": 1000 }
    },
    "required": ["table", "data"]
  }
}
```

#### db_batch_update - 批量更新

```json
{
  "name": "db_batch_update",
  "description": "批量更新数据，根据 key_field 匹配记录",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" },
      "data": { "type": "array", "items": { "type": "object" }, "maxItems": 1000 },
      "key_field": { "type": "string", "default": "id" }
    },
    "required": ["table", "data"]
  }
}
```

#### db_batch_delete - 批量逻辑删除

```json
{
  "name": "db_batch_delete",
  "description": "批量逻辑删除，根据 ID 列表",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" },
      "ids": { "type": "array", "items": { "type": "string" }, "maxItems": 1000 },
      "id_field": { "type": "string", "default": "id" }
    },
    "required": ["table", "ids"]
  }
}
```

#### db_join - 多表关联查询

```json
{
  "name": "db_join",
  "description": "多表关联查询，支持 INNER/LEFT/RIGHT JOIN",
  "inputSchema": {
    "type": "object",
    "properties": {
      "tables": { "type": "array", "minItems": 2, "maxItems": 5 },
      "joins": { "type": "array" },
      "fields": { "type": "array" },
      "where": { "type": "object" },
      "order": { "type": "array" },
      "limit": { "type": "integer", "default": 100 }
    },
    "required": ["tables", "joins"]
  }
}
```

#### db_transaction - 事务操作

```json
{
  "name": "db_transaction",
  "description": "在事务中执行多个操作，全部成功则提交，任一失败则回滚",
  "inputSchema": {
    "type": "object",
    "properties": {
      "operations": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "action": { "type": "string", "enum": ["insert", "update", "delete"] },
            "table": { "type": "string" },
            "data": { "type": "object" },
            "where": { "type": "object" }
          }
        },
        "minItems": 2,
        "maxItems": 50
      }
    },
    "required": ["operations"]
  }
}
```

#### db_describe - 获取表结构

```json
{
  "name": "db_describe",
  "description": "获取表结构信息，包括字段、类型、注释等",
  "inputSchema": {
    "type": "object",
    "properties": {
      "table": { "type": "string" }
    },
    "required": ["table"]
  }
}
```

---

## 5. 核心功能设计

### 5.1 逻辑删除字段自动检测

#### 数据结构

```go
// DeleteFieldInfo 删除字段检测结果
type DeleteFieldInfo struct {
    TableName   string   // 表名
    Fields      []Field  // 检测到的删除字段列表（1-2个字段）
    DeleteValue string   // 删除标记值
}

type Field struct {
    Name      string // 字段名
    Type      string // 类型: timestamp, datetime, boolean, tinyint, int
    TrueValue string // 删除时的真值
}
```

#### 检测模式

**字段名检测模式（按优先级排序）：**

| 优先级 | 字段名 | 类型 | 删除值 |
|--------|--------|------|--------|
| 1 | deleted_at | timestamp | 0000-00-00 00:00:00 |
| 2 | delete_time | datetime | 0000-00-00 00:00:00 |
| 3 | delete_date | datetime | - |
| 4 | deleted_time | datetime | 0000-00-00 00:00:00 |
| 5 | is_del | tinyint | 1 |
| 6 | is_deleted | tinyint | 1 |
| 7 | deleted | tinyint | 1 |
| 8 | delete_flag | tinyint | 1 |
| 9 | del_flag | tinyint | 1 |
| 10 | del | tinyint | 1 |
| 11 | status | integer | 1 |

**COMMENT 语义关键词：**

```
删除, del, is_del, 逻辑删除, 软删除
```

**COMMENT 值映射检测：**

支持从 COMMENT 中提取删除值，例如：
- `"是否删除：0.否，1.是"` → 删除值: `1`
- `"状态:0正常,1删除"` → 删除值: `1`

#### 检测优先级

1. **字段名精确匹配**（最高优先级）
2. **COMMENT 包含删除关键词**
3. **COMMENT 中有值映射**

#### 检测示例

```sql
`deleted_time` datetime NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT '删除时间'
`is_del` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '是否删除：0.否，1.是'
```

检测结果：

| 字段 | 检测方式 | Type | TrueValue |
|------|----------|------|-----------|
| deleted_time | 字段名匹配 "delete_time" | datetime | 0000-00-00 00:00:00 |
| is_del | 字段名匹配 "is_del" + COMMENT 语义 + 值映射 "1" | tinyint | 1 |

返回结果：
```go
&DeleteFieldInfo{
    TableName: "users",
    Fields: []Field{
        {Name: "deleted_time", Type: "datetime", TrueValue: "0000-00-00 00:00:00"},
        {Name: "is_del", Type: "tinyint", TrueValue: "1"},
    },
    DeleteValue: "0000-00-00 00:00:00",
}
```

### 5.2 CRUD 服务

```go
type CRUDService struct {
    repo     *repository.Repository
    detector *detector.DeleteFieldDetector
    audit    *AuditService
    logger   *logger.Logger
}
```

核心方法：
- `Query(ctx, req)` - 单表查询
- `Insert(ctx, req)` - 插入数据
- `Update(ctx, req)` - 更新数据
- `Delete(ctx, req)` - 逻辑删除
- `BatchInsert(ctx, req)` - 批量插入
- `BatchUpdate(ctx, req)` - 批量更新
- `BatchDelete(ctx, req)` - 批量逻辑删除
- `Join(ctx, req)` - 多表关联查询

### 5.3 事务服务

```go
type TransactionService struct {
    repo   *repository.Repository
    audit  *AuditService
    logger *logger.Logger
}

func (s *TransactionService) Execute(ctx, req) (*TransactionResult, error) {
    // 开始事务
    tx := s.repo.Begin()

    // 执行每个操作
    for _, op := range req.Operations {
        result, err := s.executeOperation(tx, &op)
        if err != nil {
            tx.Rollback()
            return nil, err
        }
    }

    // 全部成功，提交
    tx.Commit()
    return &TransactionResult{Success: true}, nil
}
```

### 5.4 审计日志服务

```go
type AuditService struct {
    repo   *repository.Repository
    table  string
    logger *logger.Logger
}

func (s *AuditService) Start(ctx, operation, table, recordID) *AuditContext
func (s *AuditService) Success(ctx, beforeData, affectedRows)
func (s *AuditService) Fail(ctx, err)
```

---

## 6. 非功能性设计

### 6.1 限流控制

```go
type RateLimiter struct {
    limiter  *rate.Limiter  // golang.org/x/time/rate
    enabled  bool
    requests int
    burst    int
}
```

配置：
- `Enabled`: 是否启用限流（默认 true）
- `Requests`: 每秒请求数（默认 100）
- `Burst`: 突发容量（默认 200）

### 6.2 超时控制

```go
type TimeoutConfig struct {
    Connect     time.Duration // 连接超时: 5s
    Query       time.Duration // 查询超时: 30s
    Mutation    time.Duration // 写操作超时: 10s
    Transaction time.Duration // 事务超时: 60s
}
```

### 6.3 连接池管理

```go
type ConnectionManager struct {
    db     *gorm.DB
    config *config.Config
}

func (m *ConnectionManager) DB() *gorm.DB
func (m *ConnectionManager) Close() error
func (m *ConnectionManager) HealthCheck() error
```

连接池配置：
- `MaxIdleConns`: 最大空闲连接数（默认 10）
- `MaxOpenConns`: 最大打开连接数（默认 100）
- `ConnMaxLifetime`: 连接最大存活时间（默认 1h）
- `ConnMaxIdleTime`: 空闲连接最大存活时间（默认 10m）

### 6.4 结构化日志

```go
type Logger struct {
    level  slog.Level
    format string  // json, text
    output string  // stdout, file
}
```

日志格式示例（JSON）：
```json
{
    "time": "2026-04-02T10:00:00Z",
    "level": "INFO",
    "msg": "audit",
    "operation": "SELECT",
    "table": "users",
    "status": "success",
    "duration_ms": 15
}
```

### 6.5 错误处理

```go
type ErrorCode string

const (
    ErrInvalidInput     ErrorCode = "INVALID_INPUT"
    ErrTableNotFound    ErrorCode = "TABLE_NOT_FOUND"
    ErrRecordNotFound   ErrorCode = "RECORD_NOT_FOUND"
    ErrDuplicateEntry   ErrorCode = "DUPLICATE_ENTRY"
    ErrForeignKey       ErrorCode = "FOREIGN_KEY_ERROR"
    ErrTimeout          ErrorCode = "TIMEOUT"
    ErrRateLimit        ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrInternal         ErrorCode = "INTERNAL_ERROR"
)
```

---

## 7. 测试策略

### 7.1 测试分层

```
┌─────────────────────────────────────────┐
│            E2E Tests (10%)              │  ← 完整流程测试
├───────────────────────────────���─────────┤
│        Integration Tests (30%)          │  ← 数据库交互测试
├─────────────────────────────────────────┤
│          Unit Tests (60%)               │  ← 各层独立测试
└─────────────────────────────────────────┘
```

### 7.2 测试覆盖率目标

```
测试覆盖率目标: 100%

分层覆盖率:
├── internal/config/      100%
├── internal/connection/  100%
├── internal/detector/     100%
├── internal/repository/   100%
├── internal/service/     100%
├── internal/middleware/  100%
├── internal/mcp/         100%
└── pkg/logger/           100%
```

### 7.3 测试工具

| 类型 | 工具 |
|------|------|
| Mock | gomock |
| 数据库（集成测试） | testcontainers |
| 断言 | testify/assert |

---

## 8. 质量标准

### 8.1 代码质量标准

| 指标 | 标准 | 验证方式 |
|------|------|----------|
| 测试覆盖率 | 100% | `go test -cover` |
| 代码规范 | 通过 golangci-lint | `golangci-lint run` |
| 安全扫描 | 无高危漏洞 | `gosec ./...` |
| 循环复杂度 | ≤ 15 | golangci-lint |
| 函数长度 | ≤ 50 行 | golangci-lint |

### 8.2 性能标准

| 指标 | 标准 | 说明 |
|------|------|------|
| 单次查询延迟 | < 100ms | 简单查询 |
| 批量操作延迟 | < 1s | 1000 条记录 |
| 并发能力 | 100 QPS | 限流默认值 |
| 内存占用 | < 512MB | 正常负载 |

---

## 9. 验收标准

### 9.1 功能性需求验收

| ID | 需求 | 验收标准 | 验证方法 |
|----|------|----------|----------|
| F01 | 单表查询 | 支持条件、排序、分页，返回正确结果 | 集成测试 |
| F02 | 插入数据 | 正确插入并返回 LastInsertID | 集成测试 |
| F03 | 更新数据 | 正确更新并返回 AffectedRows | 集成测试 |
| F04 | 逻辑删除 | 自动检测删除字段并执行逻辑删除 | 集成测试 |
| F05 | 批量插入 | 支持 ≤1000 条，返回成功/失败统计 | 集成测试 |
| F06 | 批量更新 | 支持 ≤1000 条，按 key_field 匹配 | 集成测试 |
| F07 | 批量删除 | 支持 ≤1000 条 ID，逻辑删除 | 集成测试 |
| F08 | 多表 JOIN | 支持 2-5 表关联查询 | 集成测试 |
| F09 | 事务操作 | 全部成功提交，任一失败回滚 | 集成测试 |
| F10 | 表结构查询 | 返回字段信息和检测到的删除字段 | 集成测试 |
| F11 | 删除字段检测 | 支持字段名、COMMENT 语义、值映射检测 | 单元测试 |
| F12 | 双删除字段 | 支持检测 1-2 个删除字段 | 单元测试 |
| F13 | 配置文件 | YAML/JSON 配置文件加载 | 单元测试 |
| F14 | MCP 参数 | MCP 配置参数覆盖 | 单元测试 |
| F15 | 审计日志 | 记录操作类型、表、记录、前后数据 | 集成测试 |

### 9.2 非功能性需求验收

| ID | 需求 | 验收标准 | 验证方法 |
|----|------|----------|----------|
| NF01 | 限流控制 | 超过限制返回错误 | 单元测试 |
| NF02 | 超时控制 | 查询/写操作/事务各自超时 | 单元测试 |
| NF03 | 连接池 | 支持配置连接池参数 | 集成测试 |
| NF04 | 结构化日志 | JSON 格式输出，支持级别配置 | 单元测试 |
| NF05 | 错误处理 | 统一错误码和错误信息 | 单元测试 |
| NF06 | 测试覆盖率 | 100% | `go test -cover` |
| NF07 | 代码规范 | golangci-lint 通过 | `golangci-lint run` |
| NF08 | 安全扫描 | 无高危漏洞 | `gosec ./...` |

### 9.3 文档验收

| ID | 文档 | 验收标准 |
|----|------|----------|
| D01 | README.md | 包含安装、配置、使用说明 |
| D02 | API 文档 | 所有 MCP 工具的参数和返回说明 |
| D03 | 配置说明 | 所有配置项的含义和默认值 |
| D04 | 架构设计 | 分层设计和组件职责 |
| D05 | 测试报告 | 覆盖率报告和测试结果 |

---

## 10. 产品验收清单

### 环境准备
- [ ] Go 1.21+ 已安装
- [ ] MySQL 8.0+ 已安装
- [ ] Docker 已安装（用于集成测试）

### 功能验收
- [ ] F01: 单表查询
- [ ] F02: 插入数据
- [ ] F03: 更新数据
- [ ] F04: 逻辑删除
- [ ] F05: 批量插入
- [ ] F06: 批量更新
- [ ] F07: 批量删除
- [ ] F08: 多表 JOIN
- [ ] F09: 事务操作
- [ ] F10: 表结构查询
- [ ] F11: 删除字段检测
- [ ] F12: 双删除字段
- [ ] F13: 配置文件加载
- [ ] F14: MCP 参数覆盖
- [ ] F15: 审计日志

### 非功能验收
- [ ] NF01: 限流控制
- [ ] NF02: 超时控制
- [ ] NF03: 连接池配置
- [ ] NF04: 结构化日志
- [ ] NF05: 错误处理
- [ ] NF06: 测试覆盖率 100%
- [ ] NF07: golangci-lint 通过
- [ ] NF08: 安全扫描通过

### 文档验收
- [ ] D01: README.md 完整
- [ ] D02: API 文档完整
- [ ] D03: 配置说明完整
- [ ] D04: 架构设计文档完整
- [ ] D05: 测试报告完整

---

## 11. 版本历史

| 版本 | 日期 | 描述 |
|------|------|------|
| 1.0 | 2026-04-02 | 初始版本 |
