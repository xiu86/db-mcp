package repository

import (
	"db-mcp/internal/driver"
)

// RepositoryInterface 是 DatabaseDriver 的别名
type RepositoryInterface = driver.DatabaseDriver

// 类型别名 - 导出 driver 包的类型供外部使用
type (
	QueryRequest       = driver.QueryRequest
	InsertRequest      = driver.InsertRequest
	UpdateRequest      = driver.UpdateRequest
	DeleteRequest      = driver.DeleteRequest
	BatchInsertRequest = driver.BatchInsertRequest
	BatchUpdateRequest = driver.BatchUpdateRequest
	BatchDeleteRequest = driver.BatchDeleteRequest
	JoinRequest        = driver.JoinRequest
	OrderBy            = driver.OrderBy
	QueryResult        = driver.QueryResult
	MutationResult     = driver.MutationResult
	BatchResult        = driver.BatchResult
	BatchError         = driver.BatchError
	TableSchema        = driver.TableSchema
	ColumnInfo         = driver.ColumnInfo
	TableRef           = driver.TableRef
	JoinClause         = driver.JoinClause
)
