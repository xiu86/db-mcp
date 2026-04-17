# db-mcp

一个基于 Go 的 MCP (Model Context Protocol) Server，支持 MySQL 和 MongoDB，用于通过 Claude Code 等 MCP Client 以结构化工具方式安全访问数据库。

## Features

- 支持 MySQL 和 MongoDB 多数据库实例管理
- 支持常见数据库工具：`db_query`、`db_insert`、`db_update`、`db_delete`
- 支持批量操作：`db_batch_insert`、`db_batch_update`、`db_batch_delete`
- 支持多表 JOIN 查询：`db_join`（MySQL 原生，MongoDB 通过 `$lookup` 模拟）
- 支持事务执行：`db_transaction`
- 支持表结构查看：`db_describe`
- 支持数据库实例切换：`db_switch`、`db_list_instances`
- 自动检测逻辑删除字段
- 内置审计日志输出
- 支持限流、查询超时和连接池配置
- 默认使用 stdio，可直接接入 Claude Code MCP Server

## Tech Stack

- Go 1.26+
- GORM（MySQL）
- mongo-go-driver（MongoDB）
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) (MCP server with StreamableHTTPServer/SSEServer support)

## Repository Structure

```text
db-mcp/
├── cmd/server/            # 服务入口
├── internal/config/       # 配置加载
├── internal/connection/   # 数据库连接管理
├── internal/detector/     # 逻辑删除字段检测
├── internal/driver/       # 数据库驱动抽象（MySQL / MongoDB）
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
- MySQL 5.7+/8.0+ 或 MongoDB 4.0+
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

### 多数据库实例配置（v0.0.2+）

```yaml
databases:
  - type: mysql
    name: default
    host: localhost
    port: 3306
    user: root
    password: "your-password"
    database: "your-database"
    charset: utf8mb4
  - type: mongodb
    name: mongo
    host: localhost
    port: 27017
    user: "your-mongo-user"
    password: "your-mongo-password"
    database: "your-mongo-database"
    uri: "mongodb://your-mongo-user:your-mongo-password@localhost:27017/your-mongo-database?authSource=your-mongo-database"

default: default

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

> `databases` 数组中的 `name` 为实例名，`type` 支持 `mysql` 和 `mongodb`。`default` 指定默认使用的实例。

### 单实例环境变量覆盖

```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your-password
export DB_NAME=your-database
```

### 多实例环境变量配置

```bash
export DB_INSTANCES=primary,secondary
export DB_PRIMARY_HOST=db1.example.com
export DB_PRIMARY_PORT=3306
export DB_SECONDARY_HOST=db2.example.com
export DB_SECONDARY_PORT=3306
```

> `user`、`password`、`database` 为必填项；若未提供，程序会在启动时失败。

### 向后兼容

v0.0.1 的单 `database` 配置格式仍然兼容，会自动转换为 `databases` 数组。

### 3. Run the server

```bash
# 使用默认配置文件 config.yaml
./bin/db-mcp

# 指定配置文件
./bin/db-mcp -config /absolute/path/to/config.yaml
```

The server entry point is in `cmd/server/main.go`.

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

## HTTP/SSE Transport Mode

db-mcp supports HTTP and SSE (Server-Sent Events) transport besides stdio.

### Configuration

```yaml
mcp:
  transport: http          # "stdio" | "http" | "sse"
  host: "0.0.0.0"          # Listen address
  port: 8080               # Listen port
  endpointPath: "/mcp"     # HTTP endpoint
  tokens:                  # Bearer tokens for auth
    - "your-token"
```

### Environment Variables

```bash
export MCP_TRANSPORT=http
export MCP_HOST=0.0.0.0
export MCP_PORT=8080
export MCP_ENDPOINT_PATH=/mcp
export MCP_TOKEN=your-token,another-token
```

### Usage with HTTP

```bash
# With auth
curl -H "Authorization: Bearer <token>" http://localhost:8080/mcp \
  -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# SSE endpoint
curl -H "Authorization: Bearer <token>" http://localhost:8080/sse
```

### Claude Code HTTP Mode

```bash
# HTTP 传输（推荐）
claude mcp add --transport http db-mcp http://localhost:8080/mcp

# 带 Token 认证
claude mcp add --transport http db-mcp http://localhost:8080/mcp \
  --header "Authorization: Bearer your-token"

# SSE 传输
claude mcp add --transport sse db-mcp http://localhost:8080/sse \
  --header "Authorization: Bearer your-token"
```

或在项目根目录创建 `.mcp.json`：

```json
{
  "mcpServers": {
    "db-mcp": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer ${DB_MCP_TOKEN}"
      }
    }
  }
}
```

> 启动 db-mcp 后再连接 Claude Code。确保 `mcp.transport` 配置为 `http` 或 `sse`。

## Available MCP Tools

> All tools are available via both stdio and HTTP/SSE transports.

### `db_query`

查询表数据。支持通过 `instance` 参数指定目标数据库实例。

```json
{
  "table": "users",
  "instance": "default",
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
  "instance": "default",
  "data": {"name": "John", "email": "john@example.com"}
}
```

### `db_update`

更新数据。

```json
{
  "table": "users",
  "instance": "default",
  "data": {"name": "John Doe"},
  "where": {"id": 1}
}
```

### `db_delete`

执行逻辑删除。

```json
{
  "table": "users",
  "instance": "default",
  "where": {"id": 1}
}
```

### `db_batch_insert`

批量插入数据。

```json
{
  "table": "users",
  "instance": "default",
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
  "instance": "default",
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
  "instance": "default",
  "ids": ["1", "2", "3"],
  "id_field": "id"
}
```

### `db_join`

执行多表 JOIN 查询（MySQL 原生 JOIN；MongoDB 通过 `$lookup` 聚合模拟）。

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
  "instance": "default",
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
  "instance": "default",
  "table": "users"
}
```

### `db_switch`

切换当前默认数据库实例或数据库。

```json
{
  "instance": "mongo",
  "database": "openim_v3"
}
```

### `db_list_instances`

列出所有可用的数据库实例。

```json
{}
```

See `internal/mcp/tools.go` for tool registration.

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

See `Makefile` for command definitions.

## Security Notes

- 请使用最小权限数据库账号，避免直接使用高权限生产账号
- `config.yaml` 可能包含敏感凭据，不应提交真实生产密码
- 建议将审计日志目录加入部署运维规范，防止日志丢失

## Roadmap Ideas

- [x] 多数据库实例管理
- [x] MongoDB 支持
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
