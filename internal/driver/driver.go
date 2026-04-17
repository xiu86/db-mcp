package driver

import (
	"context"

	"db-mcp/internal/detector"
)

// DatabaseDriver defines the database driver interface
type DatabaseDriver interface {
	// Connection management
	Ping(ctx context.Context) error
	Close() error

	// Basic operations
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)
	Insert(ctx context.Context, req *InsertRequest) (*MutationResult, error)
	Update(ctx context.Context, req *UpdateRequest) (*MutationResult, error)
	Delete(ctx context.Context, req *DeleteRequest) (*MutationResult, error)

	// Batch operations
	BatchInsert(ctx context.Context, req *BatchInsertRequest) (*BatchResult, error)
	BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchResult, error)
	BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchResult, error)

	// Join query (MySQL only, MongoDB returns error)
	JoinQuery(ctx context.Context, req *JoinRequest) (*QueryResult, error)

	// Schema operations
	GetTableSchema(tableName string) (*TableSchema, error)

	// Database switch (MySQL uses USE DATABASE, MongoDB switches database)
	UseDatabase(database string) error

	// Get current database name
	CurrentDatabase() string

	// Get driver type
	DriverType() DriverType
}

// DriverType represents the driver type
type DriverType string

const (
	DriverMySQL  DriverType = "mysql"
	DriverMongoDB DriverType = "mongodb"
)

// QueryRequest represents a query request
type QueryRequest struct {
	Table  string
	Fields []string
	Where  map[string]interface{}
	Order  []OrderBy
	Limit  int
	Offset int
}

// OrderBy represents an ordering specification
type OrderBy struct {
	Field     string
	Direction string
}

// InsertRequest represents an insert request
type InsertRequest struct {
	Table string
	Data  map[string]interface{}
}

// UpdateRequest represents an update request
type UpdateRequest struct {
	Table string
	Data  map[string]interface{}
	Where map[string]interface{}
}

// DeleteRequest represents a delete request
type DeleteRequest struct {
	Table       string
	Where       map[string]interface{}
	DeleteField *detector.DeleteFieldInfo
}

// BatchInsertRequest represents a batch insert request
type BatchInsertRequest struct {
	Table string
	Data  []map[string]interface{}
}

// BatchUpdateRequest represents a batch update request
type BatchUpdateRequest struct {
	Table    string
	Data     []map[string]interface{}
	KeyField string
}

// BatchDeleteRequest represents a batch delete request
type BatchDeleteRequest struct {
	Table       string
	IDs         []string
	IDField     string
	DeleteField *detector.DeleteFieldInfo
}

// JoinRequest represents a join query request
type JoinRequest struct {
	Tables []TableRef
	Joins  []JoinClause
	Fields []string
	Where  map[string]interface{}
	Order  []OrderBy
	Limit  int
}

// TableRef represents a table reference
type TableRef struct {
	Name  string
	Alias string
}

// JoinClause represents a join clause
type JoinClause struct {
	Type      string
	FromTable string
	FromField string
	ToTable   string
	ToField   string
}

// QueryResult represents a query result
type QueryResult struct {
	Rows    []map[string]interface{}
	Total   int64
	Message string
}

// MutationResult represents a mutation result
type MutationResult struct {
	AffectedRows int64
	LastInsertID int64
	Message      string
}

// BatchResult represents a batch operation result
type BatchResult struct {
	SuccessCount int64
	FailedCount  int64
	Errors       []BatchError
}

// BatchError represents a batch error
type BatchError struct {
	Index   int
	Message string
}

// TableSchema represents a table schema
type TableSchema struct {
	TableName string
	Columns   []ColumnInfo
}

// ColumnInfo represents column information
type ColumnInfo struct {
	Name          string
	DataType      string
	IsNullable    string
	ColumnKey     string
	Extra         string
	ColumnDefault *string
	Comment       string
}
