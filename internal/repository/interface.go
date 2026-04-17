package repository

import (
	"db-mcp/internal/driver"
)

// RepositoryInterface is an alias for DatabaseDriver
type RepositoryInterface = driver.DatabaseDriver

// Type aliases - export driver package types for external use
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
