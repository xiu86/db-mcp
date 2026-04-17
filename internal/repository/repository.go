package repository

import (
	"context"
	"db-mcp/internal/driver"
)

// Repository 持有驱动并实现 DatabaseDriver
type Repository struct {
	driver driver.DatabaseDriver
}

// New 创建 Repository 实例
func New(d driver.DatabaseDriver) *Repository {
	return &Repository{driver: d}
}

// 确保 Repository 实现了 DatabaseDriver
var _ driver.DatabaseDriver = (*Repository)(nil)

// Ping 委托给驱动
func (r *Repository) Ping(ctx context.Context) error {
	return r.driver.Ping(ctx)
}

// Close 委托给驱动
func (r *Repository) Close() error {
	return r.driver.Close()
}

// DriverType 委托给驱动
func (r *Repository) DriverType() driver.DriverType {
	return r.driver.DriverType()
}

// CurrentDatabase 委托给驱动
func (r *Repository) CurrentDatabase() string {
	return r.driver.CurrentDatabase()
}

// UseDatabase 委托给驱动
func (r *Repository) UseDatabase(database string) error {
	return r.driver.UseDatabase(database)
}

// Query 委托给驱动
func (r *Repository) Query(ctx context.Context, req *driver.QueryRequest) (*driver.QueryResult, error) {
	return r.driver.Query(ctx, req)
}

// Insert 委托给驱动
func (r *Repository) Insert(ctx context.Context, req *driver.InsertRequest) (*driver.MutationResult, error) {
	return r.driver.Insert(ctx, req)
}

// Update 委托给驱动
func (r *Repository) Update(ctx context.Context, req *driver.UpdateRequest) (*driver.MutationResult, error) {
	return r.driver.Update(ctx, req)
}

// Delete 委托给驱动
func (r *Repository) Delete(ctx context.Context, req *driver.DeleteRequest) (*driver.MutationResult, error) {
	return r.driver.Delete(ctx, req)
}

// BatchInsert 委托给驱动
func (r *Repository) BatchInsert(ctx context.Context, req *driver.BatchInsertRequest) (*driver.BatchResult, error) {
	return r.driver.BatchInsert(ctx, req)
}

// BatchUpdate 委托给驱动
func (r *Repository) BatchUpdate(ctx context.Context, req *driver.BatchUpdateRequest) (*driver.BatchResult, error) {
	return r.driver.BatchUpdate(ctx, req)
}

// BatchDelete 委托给驱动
func (r *Repository) BatchDelete(ctx context.Context, req *driver.BatchDeleteRequest) (*driver.BatchResult, error) {
	return r.driver.BatchDelete(ctx, req)
}

// JoinQuery 委托给驱动
func (r *Repository) JoinQuery(ctx context.Context, req *driver.JoinRequest) (*driver.QueryResult, error) {
	return r.driver.JoinQuery(ctx, req)
}

// GetTableSchema 委托给驱动
func (r *Repository) GetTableSchema(tableName string) (*driver.TableSchema, error) {
	return r.driver.GetTableSchema(tableName)
}

// 旧接口方法(向后兼容) - 使用 context.Background()

// LogicalDelete 逻辑删除(旧接口)
func (r *Repository) LogicalDelete(req *DeleteRequest) (*driver.MutationResult, error) {
	return r.driver.Delete(context.Background(), req)
}

// BatchLogicalDelete 批量逻辑删除(旧接口)
func (r *Repository) BatchLogicalDelete(req *driver.BatchDeleteRequest) (*driver.BatchResult, error) {
	return r.driver.BatchDelete(context.Background(), req)
}

// joinFields 连接字段列表
func joinFields(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}
