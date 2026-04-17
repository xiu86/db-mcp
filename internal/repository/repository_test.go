package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/detector"
)

func TestNew(t *testing.T) {
	repo := New(nil)
	assert.NotNil(t, repo)
}

func TestQueryRequest_Fields(t *testing.T) {
	req := &QueryRequest{
		Table:  "users",
		Fields: []string{"id", "name", "email"},
		Where:  map[string]interface{}{"status": "active"},
		Limit:  10,
		Offset: 0,
	}

	assert.Equal(t, "users", req.Table)
	assert.Len(t, req.Fields, 3)
	assert.Equal(t, 10, req.Limit)
	assert.Equal(t, 0, req.Offset)
}

func TestQueryRequest_Order(t *testing.T) {
	req := &QueryRequest{
		Table: "users",
		Order: []OrderBy{
			{Field: "created_at", Direction: "desc"},
			{Field: "id", Direction: "asc"},
		},
	}

	assert.Len(t, req.Order, 2)
	assert.Equal(t, "created_at", req.Order[0].Field)
	assert.Equal(t, "desc", req.Order[0].Direction)
}

func TestInsertRequest(t *testing.T) {
	req := &InsertRequest{
		Table: "users",
		Data: map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		},
	}

	assert.Equal(t, "users", req.Table)
	assert.Equal(t, "John", req.Data["name"])
}

func TestUpdateRequest(t *testing.T) {
	req := &UpdateRequest{
		Table: "users",
		Data:  map[string]interface{}{"name": "Jane"},
		Where: map[string]interface{}{"id": 1},
	}

	assert.Equal(t, "users", req.Table)
	assert.Equal(t, "Jane", req.Data["name"])
	assert.Equal(t, 1, req.Where["id"])
}

func TestDeleteRequest(t *testing.T) {
	deleteInfo := &detector.DeleteFieldInfo{
		TableName: "users",
		Fields: []detector.Field{
			{Name: "is_del", Type: "tinyint", TrueValue: "1"},
		},
	}

	req := &DeleteRequest{
		Table:       "users",
		Where:       map[string]interface{}{"id": 1},
		DeleteField: deleteInfo,
	}

	assert.Equal(t, "users", req.Table)
	assert.NotNil(t, req.DeleteField)
	assert.Equal(t, "is_del", req.DeleteField.Fields[0].Name)
}

func TestBatchInsertRequest(t *testing.T) {
	req := &BatchInsertRequest{
		Table: "users",
		Data: []map[string]interface{}{
			{"name": "User 1"},
			{"name": "User 2"},
			{"name": "User 3"},
		},
	}

	assert.Equal(t, "users", req.Table)
	assert.Len(t, req.Data, 3)
}

func TestBatchUpdateRequest(t *testing.T) {
	req := &BatchUpdateRequest{
		Table:    "users",
		KeyField: "id",
		Data: []map[string]interface{}{
			{"id": 1, "name": "Updated 1"},
			{"id": 2, "name": "Updated 2"},
		},
	}

	assert.Equal(t, "users", req.Table)
	assert.Equal(t, "id", req.KeyField)
	assert.Len(t, req.Data, 2)
}

func TestBatchDeleteRequest(t *testing.T) {
	deleteInfo := &detector.DeleteFieldInfo{
		TableName: "users",
		Fields: []detector.Field{
			{Name: "is_del", TrueValue: "1"},
		},
	}

	req := &BatchDeleteRequest{
		Table:       "users",
		IDs:         []string{"1", "2", "3"},
		IDField:     "id",
		DeleteField: deleteInfo,
	}

	assert.Equal(t, "users", req.Table)
	assert.Len(t, req.IDs, 3)
	assert.Equal(t, "id", req.IDField)
}

func TestJoinRequest(t *testing.T) {
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
		Fields: []string{"u.id", "u.name", "o.order_id"},
		Where:  map[string]interface{}{"u.status": "active"},
		Order:  []OrderBy{{Field: "u.created_at", Direction: "desc"}},
		Limit:  100,
	}

	assert.Len(t, req.Tables, 2)
	assert.Len(t, req.Joins, 1)
	assert.Equal(t, "left", req.Joins[0].Type)
	assert.Equal(t, 100, req.Limit)
}

func TestQueryResult(t *testing.T) {
	result := &QueryResult{
		Rows: []map[string]interface{}{
			{"id": 1, "name": "John"},
			{"id": 2, "name": "Jane"},
		},
		Total:   10,
		Message: "success",
	}

	assert.Len(t, result.Rows, 2)
	assert.Equal(t, int64(10), result.Total)
	assert.Equal(t, "success", result.Message)
}

func TestMutationResult(t *testing.T) {
	result := &MutationResult{
		AffectedRows: 1,
		LastInsertID: 5,
		Message:      "Insert successful",
	}

	assert.Equal(t, int64(1), result.AffectedRows)
	assert.Equal(t, int64(5), result.LastInsertID)
}

func TestBatchResult(t *testing.T) {
	result := &BatchResult{
		SuccessCount: 5,
		FailedCount:  2,
		Errors: []BatchError{
			{Index: 3, Message: "duplicate key"},
			{Index: 7, Message: "constraint violation"},
		},
	}

	assert.Equal(t, int64(5), result.SuccessCount)
	assert.Equal(t, int64(2), result.FailedCount)
	assert.Len(t, result.Errors, 2)
}

func TestBatchError(t *testing.T) {
	err := BatchError{
		Index:   5,
		Message: "test error",
	}

	assert.Equal(t, 5, err.Index)
	assert.Equal(t, "test error", err.Message)
}

func TestTableSchema(t *testing.T) {
	schema := &TableSchema{
		TableName: "users",
		Columns: []ColumnInfo{
			{
				Name:          "id",
				DataType:      "bigint",
				IsNullable:    "NO",
				ColumnKey:     "PRI",
				Extra:         "auto_increment",
				ColumnDefault: nil,
				Comment:       "主键ID",
			},
			{
				Name:          "name",
				DataType:      "varchar",
				IsNullable:    "YES",
				ColumnKey:     "",
				Extra:         "",
				ColumnDefault: strPtr("''"),
				Comment:       "用户名",
			},
		},
	}

	assert.Equal(t, "users", schema.TableName)
	assert.Len(t, schema.Columns, 2)
	assert.Equal(t, "bigint", schema.Columns[0].DataType)
	assert.Equal(t, "PRI", schema.Columns[0].ColumnKey)
}

func TestColumnInfo(t *testing.T) {
	col := ColumnInfo{
		Name:          "is_del",
		DataType:      "tinyint",
		IsNullable:    "NO",
		ColumnKey:     "MUL",
		Extra:         "",
		ColumnDefault: strPtr("0"),
		Comment:       "是否删除：0.否，1.是",
	}

	assert.Equal(t, "is_del", col.Name)
	assert.Equal(t, "tinyint", col.DataType)
	assert.Equal(t, "NO", col.IsNullable)
	assert.Equal(t, "MUL", col.ColumnKey)
	assert.Equal(t, "0", *col.ColumnDefault)
	assert.Equal(t, "是否删除：0.否，1.是", col.Comment)
}

func TestTableRef(t *testing.T) {
	ref := TableRef{
		Name:  "users",
		Alias: "u",
	}

	assert.Equal(t, "users", ref.Name)
	assert.Equal(t, "u", ref.Alias)
}

func TestJoinClause(t *testing.T) {
	clause := JoinClause{
		Type:      "inner",
		FromTable: "users",
		FromField: "id",
		ToTable:   "orders",
		ToField:   "user_id",
	}

	assert.Equal(t, "inner", clause.Type)
	assert.Equal(t, "users", clause.FromTable)
	assert.Equal(t, "id", clause.FromField)
	assert.Equal(t, "orders", clause.ToTable)
	assert.Equal(t, "user_id", clause.ToField)
}

func strPtr(s string) *string {
	return &s
}
