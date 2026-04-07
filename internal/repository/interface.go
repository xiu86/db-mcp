package repository

// RepositoryInterface 定义了数据访问层的接口
type RepositoryInterface interface {
	Query(req *QueryRequest) (*QueryResult, error)
	Insert(req *InsertRequest) (*MutationResult, error)
	Update(req *UpdateRequest) (*MutationResult, error)
	LogicalDelete(req *DeleteRequest) (*MutationResult, error)
	BatchInsert(req *BatchInsertRequest) (*BatchResult, error)
	BatchUpdate(req *BatchUpdateRequest) (*BatchResult, error)
	BatchLogicalDelete(req *BatchDeleteRequest) (*BatchResult, error)
	JoinQuery(req *JoinRequest) (*QueryResult, error)
	GetTableSchema(tableName string) (*TableSchema, error)
}

// Ensure Repository implements RepositoryInterface
var _ RepositoryInterface = (*Repository)(nil)
