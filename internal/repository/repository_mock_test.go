package repository

import (
	"context"
	"testing"

	"db-mcp/internal/driver"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryTypes(t *testing.T) {
	t.Run("QueryRequest", func(t *testing.T) {
		req := &QueryRequest{
			Table:  "users",
			Fields: []string{"id", "name"},
			Where:  map[string]interface{}{"status": "active"},
			Limit:  10,
			Offset: 0,
		}

		assert.Equal(t, "users", req.Table)
		assert.Len(t, req.Fields, 2)
		assert.Equal(t, 10, req.Limit)
	})

	t.Run("InsertRequest", func(t *testing.T) {
		req := &InsertRequest{
			Table: "users",
			Data: map[string]interface{}{
				"name": "test",
			},
		}

		assert.Equal(t, "users", req.Table)
		assert.Equal(t, "test", req.Data["name"])
	})

	t.Run("UpdateRequest", func(t *testing.T) {
		req := &UpdateRequest{
			Table: "users",
			Data:  map[string]interface{}{"name": "updated"},
			Where: map[string]interface{}{"id": 1},
		}

		assert.Equal(t, "users", req.Table)
		assert.Equal(t, "updated", req.Data["name"])
		assert.Equal(t, 1, req.Where["id"])
	})

	t.Run("DeleteRequest", func(t *testing.T) {
		req := &DeleteRequest{
			Table: "users",
			Where: map[string]interface{}{"id": 1},
		}

		assert.Equal(t, "users", req.Table)
		assert.Equal(t, 1, req.Where["id"])
	})

	t.Run("BatchInsertRequest", func(t *testing.T) {
		req := &BatchInsertRequest{
			Table: "users",
			Data: []map[string]interface{}{
				{"name": "user1"},
				{"name": "user2"},
			},
		}

		assert.Equal(t, "users", req.Table)
		assert.Len(t, req.Data, 2)
	})

	t.Run("BatchUpdateRequest", func(t *testing.T) {
		req := &BatchUpdateRequest{
			Table:    "users",
			Data:     []map[string]interface{}{{"id": 1, "name": "updated"}},
			KeyField: "id",
		}

		assert.Equal(t, "users", req.Table)
		assert.Equal(t, "id", req.KeyField)
	})

	t.Run("BatchDeleteRequest", func(t *testing.T) {
		req := &BatchDeleteRequest{
			Table:   "users",
			IDs:     []string{"1", "2"},
			IDField: "id",
		}

		assert.Equal(t, "users", req.Table)
		assert.Len(t, req.IDs, 2)
	})

	t.Run("JoinRequest", func(t *testing.T) {
		req := &JoinRequest{
			Tables: []TableRef{{Name: "users", Alias: "u"}},
			Joins: []JoinClause{
				{Type: "left", FromTable: "u", FromField: "id", ToTable: "orders", ToField: "user_id"},
			},
			Fields: []string{"u.id", "u.name"},
			Limit: 10,
		}

		assert.Len(t, req.Tables, 1)
		assert.Len(t, req.Joins, 1)
	})

	t.Run("QueryResult", func(t *testing.T) {
		result := &QueryResult{
			Rows:    []map[string]interface{}{{"id": 1, "name": "test"}},
			Total:   1,
			Message: "success",
		}

		assert.Len(t, result.Rows, 1)
		assert.Equal(t, int64(1), result.Total)
	})

	t.Run("MutationResult", func(t *testing.T) {
		result := &MutationResult{
			AffectedRows: 5,
			LastInsertID: 100,
			Message:      "success",
		}

		assert.Equal(t, int64(5), result.AffectedRows)
		assert.Equal(t, int64(100), result.LastInsertID)
	})

	t.Run("BatchResult", func(t *testing.T) {
		result := &BatchResult{
			SuccessCount: 10,
			FailedCount: 2,
			Errors: []BatchError{
				{Index: 3, Message: "error1"},
				{Index: 7, Message: "error2"},
			},
		}

		assert.Equal(t, int64(10), result.SuccessCount)
		assert.Equal(t, int64(2), result.FailedCount)
		assert.Len(t, result.Errors, 2)
	})

	t.Run("TableSchema", func(t *testing.T) {
		schema := &TableSchema{
			TableName: "users",
			Columns: []ColumnInfo{
				{Name: "id", DataType: "bigint", IsNullable: "NO", ColumnKey: "PRI"},
				{Name: "name", DataType: "varchar", IsNullable: "YES", ColumnKey: ""},
			},
		}

		assert.Equal(t, "users", schema.TableName)
		assert.Len(t, schema.Columns, 2)
	})
}

// MockDatabaseDriverForTest is a minimal mock for testing
type MockDatabaseDriverForTest struct{}

func (m *MockDatabaseDriverForTest) Ping(ctx context.Context) error { return nil }
func (m *MockDatabaseDriverForTest) Close() error                { return nil }
func (m *MockDatabaseDriverForTest) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	return &QueryResult{Rows: []map[string]interface{}{}}, nil
}
func (m *MockDatabaseDriverForTest) Insert(ctx context.Context, req *InsertRequest) (*MutationResult, error) {
	return &MutationResult{AffectedRows: 1}, nil
}
func (m *MockDatabaseDriverForTest) Update(ctx context.Context, req *UpdateRequest) (*MutationResult, error) {
	return &MutationResult{AffectedRows: 1}, nil
}
func (m *MockDatabaseDriverForTest) Delete(ctx context.Context, req *DeleteRequest) (*MutationResult, error) {
	return &MutationResult{AffectedRows: 1}, nil
}
func (m *MockDatabaseDriverForTest) BatchInsert(ctx context.Context, req *BatchInsertRequest) (*BatchResult, error) {
	return &BatchResult{SuccessCount: int64(len(req.Data))}, nil
}
func (m *MockDatabaseDriverForTest) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchResult, error) {
	return &BatchResult{SuccessCount: int64(len(req.Data))}, nil
}
func (m *MockDatabaseDriverForTest) BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchResult, error) {
	return &BatchResult{SuccessCount: int64(len(req.IDs))}, nil
}
func (m *MockDatabaseDriverForTest) JoinQuery(ctx context.Context, req *JoinRequest) (*QueryResult, error) {
	return &QueryResult{Rows: []map[string]interface{}{}}, nil
}
func (m *MockDatabaseDriverForTest) GetTableSchema(tableName string) (*TableSchema, error) {
	return &TableSchema{TableName: tableName}, nil
}
func (m *MockDatabaseDriverForTest) UseDatabase(database string) error { return nil }
func (m *MockDatabaseDriverForTest) CurrentDatabase() string { return "test" }
func (m *MockDatabaseDriverForTest) DriverType() driver.DriverType { return driver.DriverMySQL }

func TestRepositoryWithMockDriver(t *testing.T) {
	mock := &MockDatabaseDriverForTest{}
	repo := New(mock)
	assert.NotNil(t, repo)

	// Test Query method
	result, err := repo.Query(context.Background(), &QueryRequest{Table: "users"})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test Insert method
	insertResult, err := repo.Insert(context.Background(), &InsertRequest{
		Table: "users",
		Data:  map[string]interface{}{"name": "test"},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), insertResult.AffectedRows)

	// Test Update method
	updateResult, err := repo.Update(context.Background(), &UpdateRequest{
		Table: "users",
		Data:  map[string]interface{}{"name": "updated"},
		Where: map[string]interface{}{"id": 1},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), updateResult.AffectedRows)

	// Test Delete method
	deleteResult, err := repo.Delete(context.Background(), &DeleteRequest{
		Table: "users",
		Where: map[string]interface{}{"id": 1},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleteResult.AffectedRows)

	// Test BatchInsert method
	batchResult, err := repo.BatchInsert(context.Background(), &BatchInsertRequest{
		Table: "users",
		Data: []map[string]interface{}{
			{"name": "user1"},
			{"name": "user2"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), batchResult.SuccessCount)

	// Test BatchUpdate method
	batchUpdateResult, err := repo.BatchUpdate(context.Background(), &BatchUpdateRequest{
		Table: "users",
		Data: []map[string]interface{}{
			{"id": 1, "name": "updated1"},
			{"id": 2, "name": "updated2"},
		},
		KeyField: "id",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), batchUpdateResult.SuccessCount)

	// Test BatchDelete method
	batchDeleteResult, err := repo.BatchDelete(context.Background(), &BatchDeleteRequest{
		Table:   "users",
		IDs:     []string{"1", "2"},
		IDField: "id",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), batchDeleteResult.SuccessCount)

	// Test JoinQuery method
	joinResult, err := repo.JoinQuery(context.Background(), &JoinRequest{
		Tables: []TableRef{{Name: "users", Alias: "u"}},
	})
	assert.NoError(t, err)
	assert.NotNil(t, joinResult)

	// Test GetTableSchema method
	schema, err := repo.GetTableSchema("users")
	assert.NoError(t, err)
	assert.Equal(t, "users", schema.TableName)
}
