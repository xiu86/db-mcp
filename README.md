# db-mcp

一个基于 Go、GORM 和 `mark3labs/mcp-go` 的 MySQL MCP (Model Context Protocol) Server，用于通过 Claude Code 等 MCP Client 以结构化工具方式安全访问数据库。

## Features

- 支持常见数据库工具：`db_query`、`db_insert`、`db_update`、`db_delete`
- 支持批量操作：`db_batch_insert`、`db_batch_update`、`db_batch_delete`
- 支持多表 JOIN 查询：`db_join`
- 支持事务执行：`db_transaction`
- 支持表结构查看：`db_describe`
- 自动检测逻辑删除字段
- 内置审计日志输出
- 支持限流、查询超时和连接池配置
- 默认使用 stdio，可直接接入 Claude Code MCP Server

## Tech Stack

- Go 1.26+
- GORM
- MySQL Driver
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)

## Repository Structure

```text
db-mcp/
├── cmd/server/            # 服务入口
├── internal/config/       # 配置加载
├── internal/connection/   # 数据库连接管理
├── internal/detector/     # 逻辑删除字段检测
├── internal/mcp/          # MCP 工具注册与处理
├── internal/middleware/   # 限流与超时控制
├── internal/repository/   # 数据访问层
├── internal/service/      # CRUD / 审计 / 事务服务
├── pkg/logger/            # 日志组件
├── tests/                 # 单元测试与集成测试
├── config.yaml            # 默认配置文件
└── README.md
```

## Requirements

- Go 1.26 或更高版本
- MySQL 5.7+/8.0+
- 可访问目标数据库的账号

## Quick Start

### 1. Clone and build

```bash
git clone <your-repo-url>
cd db-mcp
make build
```

生成的二进制位于：

```bash
./bin/db-mcp
```

### 2. Configure database connection

项目默认读取根目录 `config.yaml`。

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: "your-password"
  database: "your-database"
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
  auditFile: ./logs/audit.log

rateLimit:
  enabled: true
  requests: 100
  burst: 200

timeout:
  connect: 5
  query: 30
  mutation: 10
  transaction: 60
```

也可以通过环境变量覆盖：

```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your-password
export DB_NAME=your-database
```

> `user`、`password`、`database` 为必填项；若未提供，程序会在启动时失败。

### 3. Run the server

```bash
# 使用默认配置文件 config.yaml
./bin/db-mcp

# 指定配置文件
./bin/db-mcp -config /absolute/path/to/config.yaml
```

服务入口见 `cmd/server/main.go:26`。

## Use with Claude Code

`db-mcp` 使用 stdio 传输，可直接注册为 Claude Code 的 MCP Server。

### Option 1: Add via CLI

```bash
# 当前项目
claude mcp add db-mcp -- /absolute/path/to/db-mcp/bin/db-mcp

# 用户级配置
claude mcp add --scope user db-mcp -- /absolute/path/to/db-mcp/bin/db-mcp
```

如果配置文件不在项目根目录，可通过包装脚本传入：

```bash
#!/bin/bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your-password
export DB_NAME=your-database
exec /absolute/path/to/db-mcp/bin/db-mcp
```

然后注册脚本：

```bash
claude mcp add db-mcp -- /absolute/path/to/db-mcp-wrapper.sh
```

### Option 2: Configure in settings

在 `.claude/settings.json` 或全局配置中添加：

```json
{
  "mcpServers": {
    "db-mcp": {
      "command": "/absolute/path/to/db-mcp/bin/db-mcp",
      "args": []
    }
  }
}
```

### Verify

```bash
claude mcp list
claude mcp get db-mcp
```

## Available MCP Tools

### `db_query`

查询表数据。

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

### `db_insert`

插入单条数据。

```json
{
  "table": "users",
  "data": {"name": "John", "email": "john@example.com"}
}
```

### `db_update`

更新数据。

```json
{
  "table": "users",
  "data": {"name": "John Doe"},
  "where": {"id": 1}
}
```

### `db_delete`

执行逻辑删除。

```json
{
  "table": "users",
  "where": {"id": 1}
}
```

### `db_batch_insert`

批量插入数据。

```json
{
  "table": "users",
  "data": [
    {"name": "User 1", "email": "user1@example.com"},
    {"name": "User 2", "email": "user2@example.com"}
  ]
}
```

### `db_batch_update`

按主键字段批量更新。

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

### `db_batch_delete`

按 ID 批量逻辑删除。

```json
{
  "table": "users",
  "ids": ["1", "2", "3"],
  "id_field": "id"
}
```

### `db_join`

执行多表 JOIN 查询。

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

### `db_transaction`

在单个事务中执行多个操作。

```json
{
  "operations": [
    {"type": "insert", "table": "orders", "data": {"user_id": 1, "amount": 100}},
    {"type": "update", "table": "users", "data": {"balance": 900}, "where": {"id": 1}}
  ]
}
```

### `db_describe`

查看表结构和检测到的逻辑删除字段。

```json
{
  "table": "users"
}
```

工具注册位置见 `internal/mcp/tools.go:114`。

## Logical Delete Detection

系统会自动识别常见逻辑删除字段，例如：

- 字段名：`deleted_at`、`deleted_time`、`delete_time`
- 布尔/标记字段：`is_deleted`、`is_del`、`deleted`
- 标志字段：`del_flag`、`delete_flag`
- 字段注释包含：`删除`、`逻辑删除`、`软删除`、`是否删除`

示例：

```sql
CREATE TABLE users (
  id BIGINT PRIMARY KEY,
  name VARCHAR(100),
  is_del TINYINT DEFAULT 0 COMMENT '是否删除：0.否，1.是',
  deleted_time DATETIME DEFAULT '0000-00-00 00:00:00' COMMENT '删除时间'
);
```

## Audit Logging

所有数据库操作都会记录审计日志到文件，默认路径为 `audit.log`，可通过 `log.auditFile` 配置。

示例记录：

```json
{
  "timestamp": "2024-01-01T00:00:00+08:00",
  "operation": "update",
  "table": "users",
  "record_id": "1",
  "request_id": "20240101000000-abc12345",
  "sql": "UPDATE `users` SET `name` = ? WHERE (id = ?)",
  "before_data": "{\"name\":\"John\"}",
  "after_data": "{\"name\":\"John Doe\"}",
  "duration_ms": 15,
  "status": "success",
  "error_msg": ""
}
```

## Development

### Build

```bash
make build
```

### Test

```bash
# 全量测试
make test

# 单元测试
make test-unit

# 集成测试
make test-integration

# 覆盖率报告
make test
make test-coverage
```

对应命令定义见 `Makefile:3`。

## Security Notes

- 请使用最小权限数据库账号，避免直接使用高权限生产账号
- `config.yaml` 可能包含敏感凭据，不应提交真实生产密码
- 建议将审计日志目录加入部署运维规范，防止日志丢失

## Roadmap Ideas

- 支持更多数据库类型
- 提供 HTTP/SSE 传输模式
- 增强更细粒度的权限控制与审计检索能力

## License

如需开源发布，请在仓库中补充 License 文件并在此处声明。

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
