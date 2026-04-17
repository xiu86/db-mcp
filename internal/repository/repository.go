package repository

import (
	"context"
	"db-mcp/internal/driver"
)

// Repository holds a driver and implements DatabaseDriver
type Repository struct {
	driver driver.DatabaseDriver
}

// New creates a Repository instance
func New(d driver.DatabaseDriver) *Repository {
	return &Repository{driver: d}
}

// Ensure Repository implements DatabaseDriver
var _ driver.DatabaseDriver = (*Repository)(nil)

// Ping delegates to driver
func (r *Repository) Ping(ctx context.Context) error {
	return r.driver.Ping(ctx)
}

// Close delegates to driver
func (r *Repository) Close() error {
	return r.driver.Close()
}

// DriverType delegates to driver
func (r *Repository) DriverType() driver.DriverType {
	return r.driver.DriverType()
}

// CurrentDatabase delegates to driver
func (r *Repository) CurrentDatabase() string {
	return r.driver.CurrentDatabase()
}

// UseDatabase delegates to driver
func (r *Repository) UseDatabase(database string) error {
	return r.driver.UseDatabase(database)
}

// Query delegates to driver
func (r *Repository) Query(ctx context.Context, req *driver.QueryRequest) (*driver.QueryResult, error) {
	return r.driver.Query(ctx, req)
}

// Insert delegates to driver
func (r *Repository) Insert(ctx context.Context, req *driver.InsertRequest) (*driver.MutationResult, error) {
	return r.driver.Insert(ctx, req)
}

// Update delegates to driver
func (r *Repository) Update(ctx context.Context, req *driver.UpdateRequest) (*driver.MutationResult, error) {
	return r.driver.Update(ctx, req)
}

// Delete delegates to driver
func (r *Repository) Delete(ctx context.Context, req *driver.DeleteRequest) (*driver.MutationResult, error) {
	return r.driver.Delete(ctx, req)
}

// BatchInsert delegates to driver
func (r *Repository) BatchInsert(ctx context.Context, req *driver.BatchInsertRequest) (*driver.BatchResult, error) {
	return r.driver.BatchInsert(ctx, req)
}

// BatchUpdate delegates to driver
func (r *Repository) BatchUpdate(ctx context.Context, req *driver.BatchUpdateRequest) (*driver.BatchResult, error) {
	return r.driver.BatchUpdate(ctx, req)
}

// BatchDelete delegates to driver
func (r *Repository) BatchDelete(ctx context.Context, req *driver.BatchDeleteRequest) (*driver.BatchResult, error) {
	return r.driver.BatchDelete(ctx, req)
}

// JoinQuery delegates to driver
func (r *Repository) JoinQuery(ctx context.Context, req *driver.JoinRequest) (*driver.QueryResult, error) {
	return r.driver.JoinQuery(ctx, req)
}

// GetTableSchema delegates to driver
func (r *Repository) GetTableSchema(tableName string) (*driver.TableSchema, error) {
	return r.driver.GetTableSchema(tableName)
}

// Old interface methods (backward compatibility) - use context.Background()

// LogicalDelete performs logical delete (old interface)
func (r *Repository) LogicalDelete(req *DeleteRequest) (*driver.MutationResult, error) {
	return r.driver.Delete(context.Background(), req)
}

// BatchLogicalDelete performs batch logical delete (old interface)
func (r *Repository) BatchLogicalDelete(req *driver.BatchDeleteRequest) (*driver.BatchResult, error) {
	return r.driver.BatchDelete(context.Background(), req)
}
