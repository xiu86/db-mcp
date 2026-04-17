package driver

import (
	"context"

	"db-mcp/internal/detector"
)

// DatabaseDriver 数据库驱动接口
type DatabaseDriver interface {
	// 连接管理
	Ping(ctx context.Context) error
	Close() error

	// 基础操作
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)
	Insert(ctx context.Context, req *InsertRequest) (*MutationResult, error)
	Update(ctx context.Context, req *UpdateRequest) (*MutationResult, error)
	Delete(ctx context.Context, req *DeleteRequest) (*MutationResult, error)

	// 批量操作
	BatchInsert(ctx context.Context, req *BatchInsertRequest) (*BatchResult, error)
	BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchResult, error)
	BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchResult, error)

	// Join查询 (仅MySQL支持，MongoDB返回错误)
	JoinQuery(ctx context.Context, req *JoinRequest) (*QueryResult, error)

	// Schema操作
	GetTableSchema(tableName string) (*TableSchema, error)

	// 库切换 (MySQL使用USE DATABASE, MongoDB切换database)
	UseDatabase(database string) error

	// 获取当前库名
	CurrentDatabase() string

	// 获取驱动类型
	DriverType() DriverType
}

// DriverType 驱动类型
type DriverType string

const (
	DriverMySQL  DriverType = "mysql"
	DriverMongoDB DriverType = "mongodb"
)

// QueryRequest 查询请求
type QueryRequest struct {
	Table  string
	Fields []string
	Where  map[string]interface{}
	Order  []OrderBy
	Limit  int
	Offset int
}

// OrderBy 排序
type OrderBy struct {
	Field     string
	Direction string
}

// InsertRequest 插入请求
type InsertRequest struct {
	Table string
	Data  map[string]interface{}
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	Table string
	Data  map[string]interface{}
	Where map[string]interface{}
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	Table       string
	Where       map[string]interface{}
	DeleteField *detector.DeleteFieldInfo
}

// BatchInsertRequest 批量插入请求
type BatchInsertRequest struct {
	Table string
	Data  []map[string]interface{}
}

// BatchUpdateRequest 批量更新请求
type BatchUpdateRequest struct {
	Table    string
	Data     []map[string]interface{}
	KeyField string
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	Table       string
	IDs         []string
	IDField     string
	DeleteField *detector.DeleteFieldInfo
}

// JoinRequest Join查询请求
type JoinRequest struct {
	Tables []TableRef
	Joins  []JoinClause
	Fields []string
	Where  map[string]interface{}
	Order  []OrderBy
	Limit  int
}

// TableRef 表引用
type TableRef struct {
	Name  string
	Alias string
}

// JoinClause Join子句
type JoinClause struct {
	Type      string
	FromTable string
	FromField string
	ToTable   string
	ToField   string
}

// QueryResult 查询结果
type QueryResult struct {
	Rows    []map[string]interface{}
	Total   int64
	Message string
}

// MutationResult 变更结果
type MutationResult struct {
	AffectedRows int64
	LastInsertID int64
	Message      string
}

// BatchResult 批量操作结果
type BatchResult struct {
	SuccessCount int64
	FailedCount  int64
	Errors       []BatchError
}

// BatchError 批量错误
type BatchError struct {
	Index   int
	Message string
}

// TableSchema 表结构
type TableSchema struct {
	TableName string
	Columns   []ColumnInfo
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name          string
	DataType      string
	IsNullable    string
	ColumnKey     string
	Extra         string
	ColumnDefault *string
	Comment       string
}
