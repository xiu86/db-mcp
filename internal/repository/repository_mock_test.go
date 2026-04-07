package repository

import (
	"testing"

	"db-mcp/internal/detector"
	"db-mcp/tests/mocks"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryMock_Query_Basic(t *testing.T) {
	mockDB := mocks.NewMockDB()
	repo := New(mockDB.ToGormDB())

	// Add test data
	mockDB.AddTable("users", []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com", "status": "active"},
		{"id": 2, "name": "Bob", "email": "bob@example.com", "status": "inactive"},
	})

	req := &QueryRequest{
		Table:  "users",
		Fields: []string{"id", "name"},
		Limit:  10,
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "users", req.Table)
	assert.Len(t, req.Fields, 2)
}

func TestRepositoryMock_Query_WithOrder(t *testing.T) {
	_, repo := New(nil), setupMockDB()
	req := &QueryRequest{
		Table: "users",
		Order: []OrderBy{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "desc"},
		},
	}

	assert.NotNil(t, repo)
	assert.Len(t, req.Order, 2)
	assert.Equal(t, "name", req.Order[0].Field)
	assert.Equal(t, "asc", req.Order[0].Direction)
}

func setupMockDB() *mocks.MockDB {
	mockDB := mocks.NewMockDB()
	mockDB.AddTable("users", []map[string]interface{}{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
	})
	return mockDB
}

func TestRepositoryMock_Insert(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &InsertRequest{
		Table: "users",
		Data: map[string]interface{}{
			"name":  "David",
			"email": "david@example.com",
		},
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "users", req.Table)
	assert.Equal(t, "David", req.Data["name"])
}

func TestRepositoryMock_Update(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &UpdateRequest{
		Table: "users",
		Data: map[string]interface{}{
			"status": "inactive",
		},
		Where: map[string]interface{}{
			"id": 1,
		},
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "users", req.Table)
	assert.Equal(t, "inactive", req.Data["status"])
}

func TestRepositoryMock_LogicalDelete_WithDeleteField(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	deleteField := &detector.DeleteFieldInfo{
		TableName: "users",
		Fields: []detector.Field{
			{Name: "is_del", Type: "tinyint", TrueValue: "1"},
			{Name: "deleted_time", Type: "datetime", TrueValue: "0000-00-00 00:00:00"},
		},
	}

	req := &DeleteRequest{
		Table:       "users",
		Where:       map[string]interface{}{"id": 1},
		DeleteField: deleteField,
	}

	assert.NotNil(t, repo)
	assert.NotNil(t, req.DeleteField)
	assert.Len(t, req.DeleteField.Fields, 2)
}

func TestRepositoryMock_LogicalDelete_NoDeleteField(t *testing.T) {
	repo := New(nil)

	req := &DeleteRequest{
		Table: "users",
		Where: map[string]interface{}{"id": 1},
	}

	_, err := repo.LogicalDelete(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no delete field detected")
}

func TestRepositoryMock_BatchInsert(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &BatchInsertRequest{
		Table: "users",
		Data: []map[string]interface{}{
			{"name": "User1", "email": "user1@example.com"},
			{"name": "User2", "email": "user2@example.com"},
			{"name": "User3", "email": "user3@example.com"},
		},
	}

	assert.NotNil(t, repo)
	assert.Len(t, req.Data, 3)
}

func TestRepositoryMock_BatchUpdate(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &BatchUpdateRequest{
		Table:    "users",
		KeyField: "id",
		Data: []map[string]interface{}{
			{"id": 1, "status": "active"},
			{"id": 2, "status": "inactive"},
		},
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "id", req.KeyField)
	assert.Len(t, req.Data, 2)
}

func TestRepositoryMock_BatchLogicalDelete_WithDeleteField(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	deleteField := &detector.DeleteFieldInfo{
		TableName: "users",
		Fields: []detector.Field{
			{Name: "is_del", Type: "tinyint", TrueValue: "1"},
		},
	}

	req := &BatchDeleteRequest{
		Table:       "users",
		IDs:         []string{"1", "2", "3"},
		IDField:     "id",
		DeleteField: deleteField,
	}

	assert.NotNil(t, repo)
	assert.Len(t, req.IDs, 3)
	assert.Equal(t, "id", req.IDField)
}

func TestRepositoryMock_BatchLogicalDelete_NoDeleteField(t *testing.T) {
	repo := New(nil)

	req := &BatchDeleteRequest{
		Table:   "users",
		IDs:     []string{"1", "2"},
		IDField: "id",
	}

	_, err := repo.BatchLogicalDelete(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no delete field detected")
}

func TestRepositoryMock_JoinQuery_TwoTables(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &JoinRequest{
		Tables: []TableRef{
			{Name: "users", Alias: "u"},
			{Name: "orders", Alias: "o"},
		},
		Joins: []JoinClause{
			{
				Type:      "inner",
				FromTable: "u",
				FromField: "id",
				ToTable:   "o",
				ToField:   "user_id",
			},
		},
		Fields: []string{"u.id", "u.name", "o.total"},
		Limit:  100,
	}

	assert.NotNil(t, repo)
	assert.Len(t, req.Tables, 2)
	assert.Len(t, req.Joins, 1)
}

func TestRepositoryMock_JoinQuery_LeftJoin(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &JoinRequest{
		Tables: []TableRef{
			{Name: "users", Alias: "u"},
			{Name: "orders", Alias: "o"},
		},
		Joins: []JoinClause{
			{
				Type:      "left",
				FromTable: "u",
				FromField: "id",
				ToTable:   "o",
				ToField:   "user_id",
			},
		},
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "left", req.Joins[0].Type)
}

func TestRepositoryMock_JoinQuery_RightJoin(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &JoinRequest{
		Tables: []TableRef{
			{Name: "users", Alias: "u"},
			{Name: "orders", Alias: "o"},
		},
		Joins: []JoinClause{
			{
				Type:      "right",
				FromTable: "u",
				FromField: "id",
				ToTable:   "o",
				ToField:   "user_id",
			},
		},
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "right", req.Joins[0].Type)
}

func TestRepositoryMock_JoinQuery_Invalid(t *testing.T) {
	repo := New(nil)

	req := &JoinRequest{
		Tables: []TableRef{
			{Name: "users", Alias: "u"},
		},
	}

	_, err := repo.JoinQuery(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 tables required")
}

func TestRepositoryMock_JoinQuery_WithWhere(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &JoinRequest{
		Tables: []TableRef{
			{Name: "users", Alias: "u"},
			{Name: "orders", Alias: "o"},
		},
		Joins: []JoinClause{
			{
				Type:      "inner",
				FromTable: "u",
				FromField: "id",
				ToTable:   "o",
				ToField:   "user_id",
			},
		},
		Where: map[string]interface{}{
			"u.status": "active",
		},
		Limit: 50,
	}

	assert.NotNil(t, repo)
	assert.Equal(t, "active", req.Where["u.status"])
	assert.Equal(t, 50, req.Limit)
}

func TestRepositoryMock_JoinQuery_WithOrder(t *testing.T) {
	_, repo := New(nil), setupMockDB()

	req := &JoinRequest{
		Tables: []TableRef{
			{Name: "users", Alias: "u"},
			{Name: "orders", Alias: "o"},
		},
		Joins: []JoinClause{
			{
				Type:      "inner",
				FromTable: "u",
				FromField: "id",
				ToTable:   "o",
				ToField:   "user_id",
			},
		},
		Order: []OrderBy{
			{Field: "u.created_at", Direction: "desc"},
			{Field: "o.id", Direction: "asc"},
		},
	}

	assert.NotNil(t, repo)
	assert.Len(t, req.Order, 2)
}

func TestRepositoryMock_ErrorHandling_LogicalDeleteNilField(t *testing.T) {
	repo := New(nil)

	req := &DeleteRequest{
		Table: "users",
		Where: map[string]interface{}{"id": 1},
	}

	_, err := repo.LogicalDelete(req)
	assert.Error(t, err)
}

func TestRepositoryMock_ErrorHandling_BatchLogicalDeleteNilField(t *testing.T) {
	repo := New(nil)

	req := &BatchDeleteRequest{
		Table:   "users",
		IDs:     []string{"1", "2"},
		IDField: "id",
	}

	_, err := repo.BatchLogicalDelete(req)
	assert.Error(t, err)
}

func TestRepositoryMock_ErrorHandling_JoinQueryInsufficientTables(t *testing.T) {
	repo := New(nil)

	req := &JoinRequest{
		Tables: []TableRef{{Name: "users", Alias: "u"}},
	}

	_, err := repo.JoinQuery(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 tables")
}

func TestRepositoryMock_MultipleDeleteFields(t *testing.T) {
	deleteField := &detector.DeleteFieldInfo{
		TableName: "users",
		Fields: []detector.Field{
			{Name: "is_del", Type: "tinyint", TrueValue: "1"},
			{Name: "deleted_at", Type: "datetime", TrueValue: "0000-00-00 00:00:00"},
		},
	}

	req := &DeleteRequest{
		Table:       "users",
		Where:       map[string]interface{}{"id": 1},
		DeleteField: deleteField,
	}

	assert.Len(t, req.DeleteField.Fields, 2)
	assert.Equal(t, "is_del", req.DeleteField.Fields[0].Name)
	assert.Equal(t, "deleted_at", req.DeleteField.Fields[1].Name)
}
