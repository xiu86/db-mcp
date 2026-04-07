package service

import (
	"context"
	"errors"
	"testing"

	"db-mcp/internal/config"
	"db-mcp/internal/repository"
	"db-mcp/pkg/logger"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestLogger() *logger.Logger {
	return logger.NewLogger(&config.LogConfig{Level: "debug", Output: "stdout"})
}

func TestNewCRUDService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()

	service := NewCRUDService(mockRepo, audit, cfg, log)

	assert.NotNil(t, service)
	assert.NotNil(t, service.detector)
}

func TestCRUDService_Query(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedResult := &repository.QueryResult{
			Rows: []map[string]interface{}{
				{"id": 1, "name": "test"},
			},
		}

		mockRepo.EXPECT().Query(&repository.QueryRequest{
			Table:  "users",
			Fields: []string{"id", "name"},
			Where:  map[string]interface{}{"status": 1},
			Order:  []repository.OrderBy{{Field: "id", Direction: "DESC"}},
			Limit:  10,
			Offset: 0,
		}).Return(expectedResult, nil)

		result, err := service.Query(ctx, "users", []string{"id", "name"},
			map[string]interface{}{"status": 1},
			[]repository.OrderBy{{Field: "id", Direction: "DESC"}}, 10, 0)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().Query(gomock.Any()).Return(nil, errors.New("db error"))

		result, err := service.Query(ctx, "users", nil, nil, nil, 0, 0)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, result)
	})
}

func TestCRUDService_Insert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := map[string]interface{}{"name": "test", "email": "test@example.com"}

	t.Run("success", func(t *testing.T) {
		expectedResult := &repository.MutationResult{AffectedRows: 1}

		mockRepo.EXPECT().Insert(&repository.InsertRequest{
			Table: "users",
			Data:  data,
		}).Return(expectedResult, nil)

		result, err := service.Insert(ctx, "users", data)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().Insert(gomock.Any()).Return(nil, errors.New("insert failed"))

		result, err := service.Insert(ctx, "users", data)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := map[string]interface{}{"name": "updated"}
	where := map[string]interface{}{"id": 1}

	t.Run("success", func(t *testing.T) {
		beforeResult := &repository.QueryResult{
			Rows: []map[string]interface{}{{"id": 1, "name": "old"}},
		}
		updateResult := &repository.MutationResult{AffectedRows: 1}

		// Before query for audit
		mockRepo.EXPECT().Query(&repository.QueryRequest{
			Table: "users",
			Where: where,
			Limit: 100,
		}).Return(beforeResult, nil)

		mockRepo.EXPECT().Update(&repository.UpdateRequest{
			Table: "users",
			Data:  data,
			Where: where,
		}).Return(updateResult, nil)

		result, err := service.Update(ctx, "users", data, where)

		assert.NoError(t, err)
		assert.Equal(t, updateResult, result)
	})

	t.Run("update error", func(t *testing.T) {
		mockRepo.EXPECT().Query(gomock.Any()).Return(nil, nil)
		mockRepo.EXPECT().Update(gomock.Any()).Return(nil, errors.New("update failed"))

		result, err := service.Update(ctx, "users", data, where)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_Delete(t *testing.T) {
	t.Run("success with delete field detected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockRepositoryInterface(ctrl)
		audit := NewAuditService(nil, "test_audit")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		where := map[string]interface{}{"id": 1}

		beforeResult := &repository.QueryResult{
			Rows: []map[string]interface{}{{"id": 1, "name": "test"}},
		}
		deleteResult := &repository.MutationResult{AffectedRows: 1}

		// Schema query for delete field detection
		schema := &repository.TableSchema{
			TableName: "users",
			Columns: []repository.ColumnInfo{
				{Name: "id", DataType: "bigint"},
				{Name: "is_del", DataType: "tinyint", Comment: "是否删除"},
			},
		}
		mockRepo.EXPECT().GetTableSchema("users").Return(schema, nil)

		// Before query for audit
		mockRepo.EXPECT().Query(&repository.QueryRequest{
			Table: "users",
			Where: where,
			Limit: 100,
		}).Return(beforeResult, nil)

		// Logical delete with detected field - verify DeleteField is not nil
		mockRepo.EXPECT().LogicalDelete(gomock.Any()).DoAndReturn(
			func(req *repository.DeleteRequest) (*repository.MutationResult, error) {
				assert.NotNil(t, req.DeleteField)
				assert.Equal(t, "users", req.Table)
				return deleteResult, nil
			},
		)

		result, err := service.Delete(ctx, "users", where)

		assert.NoError(t, err)
		assert.Equal(t, deleteResult, result)
	})

	t.Run("error in logical delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockRepositoryInterface(ctrl)
		audit := NewAuditService(nil, "test_audit")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		where := map[string]interface{}{"id": 1}

		// GetTableSchema fails, but Query is still called with empty columns returned
		mockRepo.EXPECT().GetTableSchema("users").Return(nil, errors.New("schema error"))

		// Query is still called for audit before data (ignores error)
		mockRepo.EXPECT().Query(gomock.Any()).Return(nil, nil)

		// LogicalDelete is called
		mockRepo.EXPECT().LogicalDelete(gomock.Any()).Return(nil, errors.New("delete error"))

		result, err := service.Delete(ctx, "users", where)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_BatchInsert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := []map[string]interface{}{
		{"name": "user1"},
		{"name": "user2"},
	}

	t.Run("success", func(t *testing.T) {
		expectedResult := &repository.BatchResult{SuccessCount: 2}

		mockRepo.EXPECT().BatchInsert(&repository.BatchInsertRequest{
			Table: "users",
			Data:  data,
		}).Return(expectedResult, nil)

		result, err := service.BatchInsert(ctx, "users", data)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().BatchInsert(gomock.Any()).Return(nil, errors.New("batch insert failed"))

		result, err := service.BatchInsert(ctx, "users", data)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_BatchUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := []map[string]interface{}{
		{"id": 1, "name": "user1"},
		{"id": 2, "name": "user2"},
	}

	t.Run("success", func(t *testing.T) {
		expectedResult := &repository.BatchResult{SuccessCount: 2}

		mockRepo.EXPECT().BatchUpdate(&repository.BatchUpdateRequest{
			Table:    "users",
			Data:     data,
			KeyField: "id",
		}).Return(expectedResult, nil)

		result, err := service.BatchUpdate(ctx, "users", data, "id")

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().BatchUpdate(gomock.Any()).Return(nil, errors.New("batch update failed"))

		result, err := service.BatchUpdate(ctx, "users", data, "id")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_BatchDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockRepositoryInterface(ctrl)
		audit := NewAuditService(nil, "test_audit")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		ids := []string{"1", "2", "3"}
		expectedResult := &repository.BatchResult{SuccessCount: 3}

		schema := &repository.TableSchema{
			TableName: "users",
			Columns: []repository.ColumnInfo{
				{Name: "id", DataType: "bigint"},
				{Name: "deleted_at", DataType: "datetime", Comment: "删除时间"},
			},
		}
		mockRepo.EXPECT().GetTableSchema("users").Return(schema, nil)

		mockRepo.EXPECT().BatchLogicalDelete(gomock.Any()).DoAndReturn(
			func(req *repository.BatchDeleteRequest) (*repository.BatchResult, error) {
				assert.NotNil(t, req.DeleteField)
				assert.Equal(t, "users", req.Table)
				assert.Equal(t, ids, req.IDs)
				assert.Equal(t, "id", req.IDField)
				return expectedResult, nil
			},
		)

		result, err := service.BatchDelete(ctx, "users", ids, "id")

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repository.NewMockRepositoryInterface(ctrl)
		audit := NewAuditService(nil, "test_audit")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		ids := []string{"1", "2", "3"}

		// GetTableSchema fails
		mockRepo.EXPECT().GetTableSchema("users").Return(nil, errors.New("schema error"))

		// BatchLogicalDelete is called with empty delete field
		mockRepo.EXPECT().BatchLogicalDelete(gomock.Any()).Return(nil, errors.New("batch delete error"))

		result, err := service.BatchDelete(ctx, "users", ids, "id")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_Join(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := &repository.JoinRequest{
			Tables: []repository.TableRef{
				{Name: "users", Alias: "u"},
			},
			Fields: []string{"u.id", "u.name"},
		}

		expectedResult := &repository.QueryResult{
			Rows: []map[string]interface{}{
				{"id": 1, "name": "user1"},
			},
		}

		mockRepo.EXPECT().JoinQuery(req).Return(expectedResult, nil)

		result, err := service.Join(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		req := &repository.JoinRequest{
			Tables: []repository.TableRef{{Name: "users", Alias: "u"}},
		}

		mockRepo.EXPECT().JoinQuery(req).Return(nil, errors.New("join failed"))

		result, err := service.Join(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_GetSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		schema := &repository.TableSchema{
			TableName: "users",
			Columns: []repository.ColumnInfo{
				{Name: "id", DataType: "bigint"},
				{Name: "is_del", DataType: "tinyint", Comment: "是否删除"},
			},
		}

		mockRepo.EXPECT().GetTableSchema("users").Return(schema, nil)

		result, err := service.GetSchema(ctx, "users")

		assert.NoError(t, err)
		assert.Equal(t, "users", result["table_name"])
		assert.NotNil(t, result["columns"])
		assert.NotNil(t, result["delete_fields"])
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().GetTableSchema("nonexistent").Return(nil, errors.New("table not found"))

		result, err := service.GetSchema(ctx, "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_getTableColumns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	t.Run("success", func(t *testing.T) {
		schema := &repository.TableSchema{
			TableName: "users",
			Columns: []repository.ColumnInfo{
				{Name: "id", DataType: "bigint"},
				{Name: "name", DataType: "varchar", Comment: "用户名"},
			},
		}

		mockRepo.EXPECT().GetTableSchema("users").Return(schema, nil)

		columns := service.getTableColumns("users")

		assert.Len(t, columns, 2)
		assert.Equal(t, "id", columns[0].Name)
		assert.Equal(t, "name", columns[1].Name)
	})

	t.Run("error returns empty", func(t *testing.T) {
		mockRepo.EXPECT().GetTableSchema("nonexistent").Return(nil, errors.New("not found"))

		columns := service.getTableColumns("nonexistent")

		assert.Empty(t, columns)
	})
}

func TestCRUDService_toDetectorColumns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)
	audit := NewAuditService(nil, "test_audit")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	columns := []repository.ColumnInfo{
		{Name: "id", DataType: "bigint"},
		{Name: "name", DataType: "varchar", Comment: "用户名"},
		{Name: "is_del", DataType: "tinyint", Comment: "是否删除"},
	}

	result := service.toDetectorColumns(columns)

	assert.Len(t, result, 3)
	assert.Equal(t, "id", result[0].Name)
	assert.Equal(t, "bigint", result[0].DataType)
	assert.Equal(t, "is_del", result[2].Name)
	assert.Equal(t, "是否删除", result[2].Comment)
}
