package mcp

import (
	"context"
	"errors"
	"testing"

	"db-mcp/internal/config"
	"db-mcp/internal/repository"
	"db-mcp/internal/service"
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// testMCPServer wraps MCPServer with mockable services for handler testing.
type testMCPServer struct {
	*MCPServer
	ctrl     *gomock.Controller
	mockRepo *repository.MockRepositoryInterface
}

func newTestMCPServer(t *testing.T) *testMCPServer {
	ctrl := gomock.NewController(t)
	mockRepo := repository.NewMockRepositoryInterface(ctrl)

	cfg := &config.Config{}
	log := logger.NewLogger(&config.LogConfig{Level: "debug", Format: "text", Output: "stdout"})

	audit := service.NewAuditService("")
	crud := service.NewCRUDService(mockRepo, audit, cfg, log)

	srv := &MCPServer{
		crud:      crud,
		txService: nil,
		config:    cfg,
		logger:    log,
	}

	return &testMCPServer{
		MCPServer: srv,
		ctrl:      ctrl,
		mockRepo:  mockRepo,
	}
}

// getResultText extracts text content from CallToolResult.
func getResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// Test handleQuery success
func TestHandleQuery_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expectedResult := &repository.QueryResult{
		Rows:    []map[string]interface{}{{"id": float64(1), "name": "Alice"}},
		Total:   1,
		Message: "success",
	}

	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(expectedResult, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table":  "users",
		"fields": []interface{}{"id", "name"},
		"where":  map[string]interface{}{"status": float64(1)},
		"order":  []interface{}{map[string]interface{}{"field": "id", "direction": "desc"}},
		"limit":  float64(10),
		"offset": float64(0),
	}

	result, err := ts.handleQuery(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "Alice")
}

// Test handleQuery error
func TestHandleQuery_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(nil, errors.New("db error"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
	}

	result, err := ts.handleQuery(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "db error")
}

// Test handleInsert success
func TestHandleInsert_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expectedResult := &repository.MutationResult{
		AffectedRows: 1,
		LastInsertID: 100,
		Message:      "Insert successful",
	}

	ts.mockRepo.EXPECT().Insert(gomock.Any()).Return(expectedResult, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data": map[string]interface{}{
			"name":  "Bob",
			"email": "bob@example.com",
		},
	}

	result, err := ts.handleInsert(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "100")
}

// Test handleInsert error
func TestHandleInsert_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().Insert(gomock.Any()).Return(nil, errors.New("insert failed"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  map[string]interface{}{"name": "Bob"},
	}

	result, err := ts.handleInsert(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "insert failed")
}

// Test handleUpdate success
func TestHandleUpdate_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}, nil)
	ts.mockRepo.EXPECT().Update(gomock.Any()).Return(&repository.MutationResult{AffectedRows: 1}, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  map[string]interface{}{"name": "Updated"},
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "1")
}

// Test handleUpdate error
func TestHandleUpdate_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(nil, nil)
	ts.mockRepo.EXPECT().Update(gomock.Any()).Return(nil, errors.New("update failed"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  map[string]interface{}{"name": "Updated"},
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "update failed")
}

// Test handleDelete success
func TestHandleDelete_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	ts.mockRepo.EXPECT().GetTableSchema("users").Return(&repository.TableSchema{
		TableName: "users",
		Columns:   []repository.ColumnInfo{{Name: "id", DataType: "bigint"}, {Name: "is_del", DataType: "tinyint", Comment: "删除"}},
	}, nil)
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}, nil)
	ts.mockRepo.EXPECT().LogicalDelete(gomock.Any()).Return(&repository.MutationResult{AffectedRows: 1}, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "1")
}

// Test handleDelete error from GetTableSchema (but code continues with empty delete field)
func TestHandleDelete_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	// GetTableSchema fails -> getTableColumns returns empty -> deleteField has no fields
	// Then Query is called for audit (with empty delete field, LogicalDelete still proceeds)
	ts.mockRepo.EXPECT().GetTableSchema("users").Return(nil, errors.New("table not found"))
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: nil}, nil)
	ts.mockRepo.EXPECT().LogicalDelete(gomock.Any()).Return(nil, errors.New("no delete field detected"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "no delete field detected")
}

// Test handleBatchInsert success
func TestHandleBatchInsert_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.BatchResult{SuccessCount: 3, FailedCount: 0}
	ts.mockRepo.EXPECT().BatchInsert(gomock.Any()).Return(expected, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data": []interface{}{
			map[string]interface{}{"name": "User1"},
			map[string]interface{}{"name": "User2"},
			map[string]interface{}{"name": "User3"},
		},
	}

	result, err := ts.handleBatchInsert(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "3")
}

// Test handleBatchInsert error
func TestHandleBatchInsert_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().BatchInsert(gomock.Any()).Return(nil, errors.New("batch error"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  []interface{}{map[string]interface{}{"name": "User1"}},
	}

	result, err := ts.handleBatchInsert(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "batch error")
}

// Test handleBatchUpdate success
func TestHandleBatchUpdate_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.BatchResult{SuccessCount: 2}
	ts.mockRepo.EXPECT().BatchUpdate(gomock.Any()).Return(expected, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table":    "users",
		"data":     []interface{}{map[string]interface{}{"id": float64(1), "status": "active"}},
		"key_field": "id",
	}

	result, err := ts.handleBatchUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "2")
}

// Test handleBatchUpdate default key_field
func TestHandleBatchUpdate_DefaultKeyField(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.BatchResult{SuccessCount: 1}
	ts.mockRepo.EXPECT().BatchUpdate(gomock.Any()).DoAndReturn(
		func(req *repository.BatchUpdateRequest) (*repository.BatchResult, error) {
			assert.Equal(t, "id", req.KeyField) // Default key_field
			return expected, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  []interface{}{map[string]interface{}{"id": float64(1), "status": "active"}},
		// key_field omitted -> should default to "id"
	}

	result, err := ts.handleBatchUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleBatchUpdate error
func TestHandleBatchUpdate_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().BatchUpdate(gomock.Any()).Return(nil, errors.New("batch update error"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table":    "users",
		"data":     []interface{}{map[string]interface{}{"id": float64(1)}},
		"key_field": "id",
	}

	result, err := ts.handleBatchUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "batch update error")
}

// Test handleBatchDelete success
func TestHandleBatchDelete_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	ts.mockRepo.EXPECT().GetTableSchema("users").Return(&repository.TableSchema{
		TableName: "users",
		Columns:   []repository.ColumnInfo{{Name: "id", DataType: "bigint"}, {Name: "deleted_at", DataType: "datetime", Comment: "删除时间"}},
	}, nil)
	ts.mockRepo.EXPECT().BatchLogicalDelete(gomock.Any()).Return(&repository.BatchResult{SuccessCount: 3}, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table":    "users",
		"ids":      []interface{}{"1", "2", "3"},
		"id_field": "id",
	}

	result, err := ts.handleBatchDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "3")
}

// Test handleBatchDelete default id_field
func TestHandleBatchDelete_DefaultIDField(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	ts.mockRepo.EXPECT().GetTableSchema("users").Return(&repository.TableSchema{
		TableName: "users",
		Columns:   []repository.ColumnInfo{{Name: "id", DataType: "bigint"}},
	}, nil)
	ts.mockRepo.EXPECT().BatchLogicalDelete(gomock.Any()).DoAndReturn(
		func(req *repository.BatchDeleteRequest) (*repository.BatchResult, error) {
			assert.Equal(t, "id", req.IDField) // Default id_field
			return &repository.BatchResult{SuccessCount: 2}, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"ids":   []interface{}{"1", "2"},
		// id_field omitted -> should default to "id"
	}

	result, err := ts.handleBatchDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleBatchDelete error when GetTableSchema fails (results in "no delete field detected")
func TestHandleBatchDelete_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	// GetTableSchema fails -> getTableColumns returns empty -> deleteField has no fields
	// BatchLogicalDelete returns error for missing delete field
	ts.mockRepo.EXPECT().GetTableSchema("users").Return(nil, errors.New("schema error"))
	ts.mockRepo.EXPECT().BatchLogicalDelete(gomock.Any()).Return(nil, errors.New("no delete field detected"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table":    "users",
		"ids":      []interface{}{"1", "2"},
		"id_field": "id",
	}

	result, err := ts.handleBatchDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "no delete field detected")
}

// Test handleJoin success
func TestHandleJoin_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.QueryResult{
		Rows:  []map[string]interface{}{{"u.id": float64(1), "u.name": "Alice", "o.total": float64(100)}},
		Total: 1,
	}
	ts.mockRepo.EXPECT().JoinQuery(gomock.Any()).Return(expected, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		},
		"joins": []interface{}{
			map[string]interface{}{
				"type":        "left",
				"from_table":  "u",
				"from_field":  "id",
				"to_table":    "o",
				"to_field":    "user_id",
			},
		},
		"fields": []interface{}{"u.id", "u.name", "o.total"},
		"where":  map[string]interface{}{"u.status": "active"},
		"limit":  float64(100),
	}

	result, err := ts.handleJoin(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "Alice")
}

// Test handleJoin error
func TestHandleJoin_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().JoinQuery(gomock.Any()).Return(nil, errors.New("join error"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		},
		"joins": []interface{}{
			map[string]interface{}{
				"type": "left", "from_table": "u", "from_field": "id", "to_table": "o", "to_field": "user_id",
			},
		},
	}

	result, err := ts.handleJoin(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "join error")
}

// Test handleSchema success
func TestHandleSchema_Success(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.TableSchema{
		TableName: "users",
		Columns: []repository.ColumnInfo{
			{Name: "id", DataType: "bigint", Comment: "主键"},
			{Name: "name", DataType: "varchar", Comment: "用户名"},
			{Name: "is_del", DataType: "tinyint", Comment: "是否删除"},
		},
	}
	ts.mockRepo.EXPECT().GetTableSchema("users").Return(expected, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
	}

	result, err := ts.handleSchema(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "users")
	assert.Contains(t, getResultText(result), "is_del")
}

// Test handleSchema error
func TestHandleSchema_Error(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().GetTableSchema("nonexistent").Return(nil, errors.New("table not found"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "nonexistent",
	}

	result, err := ts.handleSchema(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "table not found")
}

// Test handleQuery with nil/empty args
func TestHandleQuery_NilArgs(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expectedResult := &repository.QueryResult{Rows: []map[string]interface{}{}, Total: 0}
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(expectedResult, nil)

	req := mcp.CallToolRequest{}
	// Empty/nil arguments

	result, err := ts.handleQuery(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test toJSON error path (channel that can't marshal)
func TestToJSON_NonMarshable(t *testing.T) {
	ch := make(chan int)
	result := toJSON(ch)
	assert.Equal(t, "{}", result)
}

// Test handleJoin with no limit
func TestHandleJoin_NoLimit(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}
	ts.mockRepo.EXPECT().JoinQuery(gomock.Any()).DoAndReturn(
		func(req *repository.JoinRequest) (*repository.QueryResult, error) {
			assert.Equal(t, 0, req.Limit) // No limit set
			return expected, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		},
		"joins": []interface{}{
			map[string]interface{}{
				"type": "inner", "from_table": "u", "from_field": "id", "to_table": "o", "to_field": "user_id",
			},
		},
		// no limit
	}

	result, err := ts.handleJoin(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleQuery with nil where
func TestHandleQuery_NilWhere(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	expectedResult := &repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}
	ts.mockRepo.EXPECT().Query(gomock.Any()).DoAndReturn(
		func(req *repository.QueryRequest) (*repository.QueryResult, error) {
			assert.Nil(t, req.Where)
			return expectedResult, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"where": nil,
	}

	result, err := ts.handleQuery(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleUpdate with nil where
func TestHandleUpdate_NilWhere(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().Query(gomock.Any()).DoAndReturn(
		func(req *repository.QueryRequest) (*repository.QueryResult, error) {
			assert.Nil(t, req.Where)
			return &repository.QueryResult{Rows: []map[string]interface{}{}}, nil
		},
	)
	ts.mockRepo.EXPECT().Update(gomock.Any()).Return(&repository.MutationResult{AffectedRows: 0}, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  map[string]interface{}{"status": "active"},
		"where": nil,
	}

	result, err := ts.handleUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleDelete with nil where
func TestHandleDelete_NilWhere(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	ts.mockRepo.EXPECT().GetTableSchema("users").Return(&repository.TableSchema{
		TableName: "users",
		Columns:   []repository.ColumnInfo{{Name: "id", DataType: "bigint"}},
	}, nil)
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: []map[string]interface{}{}}, nil)
	ts.mockRepo.EXPECT().LogicalDelete(gomock.Any()).Return(&repository.MutationResult{AffectedRows: 0}, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"where": nil,
	}

	result, err := ts.handleDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleUpdate when before-query returns empty results (nil Rows).
// The service code accesses beforeResult.Rows, which must be safe.
func TestHandleUpdate_EmptyBeforeResult(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: nil}, nil)
	ts.mockRepo.EXPECT().Update(gomock.Any()).Return(&repository.MutationResult{AffectedRows: 1}, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  map[string]interface{}{"name": "Updated"},
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "1")
}

// Test handleUpdate error from Update itself
func TestHandleUpdate_UpdateError(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: nil}, nil)
	ts.mockRepo.EXPECT().Update(gomock.Any()).Return(nil, errors.New("update db error"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"data":  map[string]interface{}{"name": "Updated"},
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleUpdate(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "update db error")
}

// Test handleDelete error from GetTableSchema but then succeeds with no delete field
func TestHandleDelete_NoDeleteField(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	// Schema returns no delete field
	ts.mockRepo.EXPECT().GetTableSchema("users").Return(&repository.TableSchema{
		TableName: "users",
		Columns:   []repository.ColumnInfo{{Name: "id", DataType: "bigint"}, {Name: "name", DataType: "varchar"}},
	}, nil)
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}, nil)
	ts.mockRepo.EXPECT().LogicalDelete(gomock.Any()).DoAndReturn(
		func(req *repository.DeleteRequest) (*repository.MutationResult, error) {
			// DeleteField will be nil or have empty fields
			return &repository.MutationResult{AffectedRows: 1}, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleDelete error from LogicalDelete
func TestHandleDelete_LogicalDeleteError(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	ts.mockRepo.EXPECT().GetTableSchema("users").Return(&repository.TableSchema{
		TableName: "users",
		Columns:   []repository.ColumnInfo{{Name: "id", DataType: "bigint"}, {Name: "is_del", DataType: "tinyint"}},
	}, nil)
	ts.mockRepo.EXPECT().Query(gomock.Any()).Return(&repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}, nil)
	ts.mockRepo.EXPECT().LogicalDelete(gomock.Any()).Return(nil, errors.New("delete failed"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"table": "users",
		"where": map[string]interface{}{"id": float64(1)},
	}

	result, err := ts.handleDelete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "delete failed")
}

// Test handleJoin with nil tables (insufficient tables -> validation error)
func TestHandleJoin_NilTables(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	// JoinQuery validates len(Tables) < 2 and returns error
	ts.mockRepo.EXPECT().JoinQuery(gomock.Any()).Return(nil, errors.New("at least 2 tables required for join"))

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"tables": nil,
		"joins":  nil,
	}

	result, err := ts.handleJoin(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, getResultText(result), "at least 2 tables required")
}

// Test handleJoin with nil where
func TestHandleJoin_NilWhere(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.QueryResult{Rows: []map[string]interface{}{{"id": float64(1)}}}
	ts.mockRepo.EXPECT().JoinQuery(gomock.Any()).DoAndReturn(
		func(req *repository.JoinRequest) (*repository.QueryResult, error) {
			assert.Nil(t, req.Where)
			return expected, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		},
		"joins": []interface{}{
			map[string]interface{}{
				"type": "inner", "from_table": "u", "from_field": "id", "to_table": "o", "to_field": "user_id",
			},
		},
		"where": nil,
	}

	result, err := ts.handleJoin(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// Test handleJoin with nil fields (should default to *)
func TestHandleJoin_NilFields(t *testing.T) {
	ts := newTestMCPServer(t)
	defer ts.ctrl.Finish()

	ctx := context.Background()

	expected := &repository.QueryResult{Rows: []map[string]interface{}{}}
	ts.mockRepo.EXPECT().JoinQuery(gomock.Any()).DoAndReturn(
		func(req *repository.JoinRequest) (*repository.QueryResult, error) {
			assert.Empty(t, req.Fields) // nil fields
			return expected, nil
		},
	)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		},
		"joins": []interface{}{
			map[string]interface{}{
				"type": "inner", "from_table": "u", "from_field": "id", "to_table": "o", "to_field": "user_id",
			},
		},
		"fields": nil,
	}

	result, err := ts.handleJoin(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}