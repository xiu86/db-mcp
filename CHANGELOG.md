# Changelog

## [0.0.3] - 2026-04-17
### Added
- HTTP/SSE transport mode with StreamableHTTPServer and SSEServer
- Bearer token authentication middleware
- Configurable transport via config.yaml and environment variables
- Graceful shutdown for HTTP mode

## [0.0.2] - 2026-04-09
### Added
- Multi-database instance management (MySQL + MongoDB)
- MongoDB driver with CRUD, batch, and $lookup support
- SQL injection prevention via sanitizer package
- Comprehensive test suite with mock coverage

## [0.0.1] - 2026-04-03
### Added
- Initial MCP server with stdio transport
- MySQL CRUD operations (query, insert, update, delete)
- Batch operations (batch insert, update, delete)
- Multi-table JOIN query support
- Transaction support
- Logical delete field detection
- Configuration via YAML and environment variables
