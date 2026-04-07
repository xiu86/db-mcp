# db-mcp

MySQL 数据库 MCP (Model Context Protocol) 服务，基于 mark3labs/mcp-go SDK 和 GORM 实现。

## 功能特性

- **基础 CRUD 操作**: 查询、插入、更新、逻辑删除
- **高级操作**: JOIN 查询、批量操作、事务支持
- **逻辑删除自动检测**: 支持字段名模式、COMMENT 语义、类型值映射
- **详细审计日志**: 完整操作记录，支持数据恢复扩展
- **限流/超时控制**: 保护数据库资源
- **连接池管理**: 可配置的连接池参数

## 安装

```bash
go get db-mcp
```

## 配置

### 配置文件 (config.yaml)

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: secret
  database: mydb
  charset: utf8mb4

pool:
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 1h
  conn_max_idle_time: 10m

log:
  level: info
  format: json
  output: stdout

rate_limit:
  enabled: true
  requests_per_second: 100
  burst: 200

timeout:
  query: 30s
  write: 60s
  transaction: 120s
```

### 环境变量覆盖

所有配置项都支持环境变量覆盖，格式为 `DB_MCP_<SECTION>_<KEY>`：

```bash
export DB_MCP_DATABASE_HOST=localhost
export DB_MCP_DATABASE_PORT=3306
export DB_MCP_DATABASE_USER=root
export DB_MCP_DATABASE_PASSWORD=secret
export DB_MCP_DATABASE_DATABASE=mydb
```

## 运行

```bash
# 使用默认配置文件 config.yaml
./db-mcp

# 指定配置文件
./db-mcp -config /path/to/config.yaml
```

## MCP 工具

### db_query

查询数据

```json
{
  "table": "users",
  "fields": ["id", "name", "email"],
  "where": {"status": "active"},
  "order": [{"field": "created_at", "direction": "desc"}],
  "limit": 10,
  "offset": 0
}
```

### db_insert

插入数据

```json
{
  "table": "users",
  "data": {"name": "John", "email": "john@example.com"}
}
```

### db_update

更新数据

```json
{
  "table": "users",
  "data": {"name": "John Doe"},
  "where": {"id": 1}
}
```

### db_delete

逻辑删除（自动检测删除字段）

```json
{
  "table": "users",
  "where": {"id": 1}
}
```

### db_batch_insert

批量插入

```json
{
  "table": "users",
  "data": [
    {"name": "User 1", "email": "user1@example.com"},
    {"name": "User 2", "email": "user2@example.com"}
  ]
}
```

### db_batch_update

批量更新

```json
{
  "table": "users",
  "data": [
    {"id": 1, "name": "Updated 1"},
    {"id": 2, "name": "Updated 2"}
  ],
  "key_field": "id"
}
```

### db_batch_delete

批量逻辑删除

```json
{
  "table": "users",
  "ids": ["1", "2", "3"],
  "id_field": "id"
}
```

### db_join

JOIN 查询

```json
{
  "tables": [
    {"name": "users", "alias": "u"},
    {"name": "orders", "alias": "o"}
  ],
  "joins": [
    {
      "type": "left",
      "from_table": "u",
      "from_field": "id",
      "to_table": "o",
      "to_field": "user_id"
    }
  ],
  "fields": ["u.id", "u.name", "o.order_id"],
  "where": {"u.status": "active"},
  "limit": 100
}
```

### db_transaction

事务操作

```json
{
  "operations": [
    {"type": "insert", "table": "orders", "data": {"user_id": 1, "amount": 100}},
    {"type": "update", "table": "users", "data": {"balance": 900}, "where": {"id": 1}}
  ]
}
```

## 逻辑删除字段检测

系统自动检测以下模式的删除字段：

### 字段名模式
- `deleted_at`, `deleted_time`, `delete_time`
- `is_deleted`, `is_del`, `deleted`
- `del_flag`, `delete_flag`

### COMMENT 关键字
- 包含 "删除"、"逻辑删除"、"软删除"
- 包含 "是否删除" 格式（如 "是否删除：0.否，1.是"）

### 示例

```sql
CREATE TABLE users (
  id BIGINT PRIMARY KEY,
  name VARCHAR(100),
  is_del TINYINT DEFAULT 0 COMMENT '是否删除：0.否，1.是',
  deleted_time DATETIME DEFAULT '0000-00-00 00:00:00' COMMENT '删除时间'
);
```

系统将自动识别 `is_del` 和 `deleted_time` 为删除字段。

## 审计日志

所有操作自动记录审计日志：

```json
{
  "id": 1,
  "timestamp": "2024-01-01T00:00:00Z",
  "operation": "update",
  "table": "users",
  "record_id": "1",
  "actor": "system",
  "request_id": "20240101000000-abc12345",
  "before_data": "{\"name\":\"John\"}",
  "after_data": "{\"name\":\"John Doe\"}",
  "duration": 15,
  "status": "success"
}
```

## 测试

```bash
# 运行所有测试
go test ./...

# 运行单元测试
go test ./tests/unit/...

# 运行集成测试
go test ./tests/integration/... -tags=integration

# 测试覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 项目结构

```
db-mcp/
├── cmd/
│   └── server/
│       └── main.go          # 服务入口
├── internal/
│   ├── config/              # 配置管理
│   ├── connection/          # 数据库连接
│   ├── detector/            # 删除字段检测
│   ├── errors/              # 错误处理
│   ├── mcp/                 # MCP 工具定义
│   ├── middleware/          # 限流/超时中间件
│   ├── repository/          # 数据访问层
│   └── service/             # 业务逻辑层
├── pkg/
│   └── logger/              # 日志组件
├── tests/
│   ├── unit/                # 单元测试
│   └── integration/         # 集成测试
├── docs/
│   └── superpowers/
│       ├── specs/           # 设计文档
│       └── plans/           # 实现计划
├── config.yaml              # 默认配置
├── go.mod
└── README.md
```

## 验收标准

### 功能验收
- [x] 支持基础 CRUD 操作
- [x] 支持 JOIN 查询
- [x] 支持批量操作
- [x] 支持事务
- [x] 逻辑删除自动检测
- [x] 审计日志记录

### 非功能验收
- [x] 限流控制
- [x] 超时控制
- [x] 连接池管理
- [x] 结构化日志

### 质量验收
- [x] 编译通过
- [x] 代码规范
- [x] 文档完整

## 许可证

MIT License
