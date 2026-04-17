package service

import (
	"context"
	"errors"
	"testing"

	"db-mcp/internal/config"
	"db-mcp/internal/driver"
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

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()

	service := NewCRUDService(mockRepo, audit, cfg, log)

	assert.NotNil(t, service)
	assert.NotNil(t, service.detector)
}

func TestCRUDService_Query(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedResult := &driver.QueryResult{
			Rows: []map[string]interface{}{
				{"id": 1, "name": "test"},
			},
		}

		mockRepo.EXPECT().Query(ctx, gomock.Any()).Return(expectedResult, nil)

		result, err := service.Query(ctx, "users", []string{"id", "name"},
			map[string]interface{}{"status": 1},
			[]driver.OrderBy{{Field: "id", Direction: "DESC"}}, 10, 0)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().Query(ctx, gomock.Any()).Return(nil, errors.New("db error"))

		result, err := service.Query(ctx, "users", nil, nil, nil, 0, 0)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, result)
	})
}

func TestCRUDService_Insert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := map[string]interface{}{"name": "test", "email": "test@example.com"}

	t.Run("success", func(t *testing.T) {
		expectedResult := &driver.MutationResult{AffectedRows: 1}

		mockRepo.EXPECT().Insert(ctx, gomock.Any()).Return(expectedResult, nil)

		result, err := service.Insert(ctx, "users", data)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().Insert(ctx, gomock.Any()).Return(nil, errors.New("insert failed"))

		result, err := service.Insert(ctx, "users", data)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := map[string]interface{}{"name": "updated"}
	where := map[string]interface{}{"id": 1}

	t.Run("success", func(t *testing.T) {
		beforeResult := &driver.QueryResult{
			Rows: []map[string]interface{}{{"id": 1, "name": "old"}},
		}
		updateResult := &driver.MutationResult{AffectedRows: 1}

		// Before query for audit
		mockRepo.EXPECT().Query(ctx, gomock.Any()).Return(beforeResult, nil)

		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(updateResult, nil)

		result, err := service.Update(ctx, "users", data, where)

		assert.NoError(t, err)
		assert.Equal(t, updateResult, result)
	})

	t.Run("update error", func(t *testing.T) {
		mockRepo.EXPECT().Query(ctx, gomock.Any()).Return(nil, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil, errors.New("update failed"))

		result, err := service.Update(ctx, "users", data, where)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_Delete(t *testing.T) {
	t.Run("success with delete field detected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := driver.NewMockDatabaseDriver(ctrl)
		audit := NewAuditService("")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		where := map[string]interface{}{"id": 1}

		beforeResult := &driver.QueryResult{
			Rows: []map[string]interface{}{{"id": 1, "name": "test"}},
		}
		deleteResult := &driver.MutationResult{AffectedRows: 1}

		// Schema query for delete field detection
		schema := &driver.TableSchema{
			TableName: "users",
			Columns: []driver.ColumnInfo{
				{Name: "id", DataType: "bigint"},
				{Name: "is_del", DataType: "tinyint", Comment: "是否删除"},
			},
		}
		mockRepo.EXPECT().GetTableSchema("users").Return(schema, nil)

		// Before query for audit
		mockRepo.EXPECT().Query(ctx, gomock.Any()).Return(beforeResult, nil)

		// Delete is called
		mockRepo.EXPECT().Delete(ctx, gomock.Any()).Return(deleteResult, nil)

		result, err := service.Delete(ctx, "users", where)

		assert.NoError(t, err)
		assert.Equal(t, deleteResult, result)
	})

	t.Run("error in delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := driver.NewMockDatabaseDriver(ctrl)
		audit := NewAuditService("")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		where := map[string]interface{}{"id": 1}

		// GetTableSchema fails, but Query is still called with empty columns returned
		mockRepo.EXPECT().GetTableSchema("users").Return(nil, errors.New("schema error"))

		// Query is still called for audit before data (ignores error)
		mockRepo.EXPECT().Query(ctx, gomock.Any()).Return(nil, nil)

		// Delete is called
		mockRepo.EXPECT().Delete(ctx, gomock.Any()).Return(nil, errors.New("delete error"))

		result, err := service.Delete(ctx, "users", where)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_BatchInsert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := []map[string]interface{}{
		{"name": "user1"},
		{"name": "user2"},
	}

	t.Run("success", func(t *testing.T) {
		expectedResult := &driver.BatchResult{SuccessCount: 2}

		mockRepo.EXPECT().BatchInsert(ctx, gomock.Any()).Return(expectedResult, nil)

		result, err := service.BatchInsert(ctx, "users", data)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult.SuccessCount, result.SuccessCount)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().BatchInsert(ctx, gomock.Any()).Return(nil, errors.New("batch insert failed"))

		result, err := service.BatchInsert(ctx, "users", data)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_BatchUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	data := []map[string]interface{}{
		{"id": 1, "name": "user1"},
		{"id": 2, "name": "user2"},
	}

	t.Run("success", func(t *testing.T) {
		expectedResult := &driver.BatchResult{SuccessCount: 2}

		mockRepo.EXPECT().BatchUpdate(ctx, gomock.Any()).Return(expectedResult, nil)

		result, err := service.BatchUpdate(ctx, "users", data, "id")

		assert.NoError(t, err)
		assert.Equal(t, expectedResult.SuccessCount, result.SuccessCount)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().BatchUpdate(ctx, gomock.Any()).Return(nil, errors.New("batch update failed"))

		result, err := service.BatchUpdate(ctx, "users", data, "id")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_BatchDelete(t *testing.T) {
	t.Run("success with delete field detected", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := driver.NewMockDatabaseDriver(ctrl)
		audit := NewAuditService("")
		cfg := &config.Config{}
		log := newTestLogger()
		service := NewCRUDService(mockRepo, audit, cfg, log)

		ctx := context.Background()
		ids := []string{"1", "2"}

		// Schema query for delete field detection
		schema := &driver.TableSchema{
			TableName: "users",
			Columns: []driver.ColumnInfo{
				{Name: "id", DataType: "bigint"},
				{Name: "is_del", DataType: "tinyint", Comment: "是否删除"},
			},
		}
		mockRepo.EXPECT().GetTableSchema("users").Return(schema, nil)

		expectedResult := &driver.BatchResult{SuccessCount: 2}

		mockRepo.EXPECT().BatchDelete(ctx, gomock.Any()).Return(expectedResult, nil)

		result, err := service.BatchDelete(ctx, "users", ids, "id")

		assert.NoError(t, err)
		assert.Equal(t, expectedResult.SuccessCount, result.SuccessCount)
	})
}

func TestCRUDService_Join(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	ctx := context.Background()
	req := &driver.JoinRequest{
		Tables: []driver.TableRef{{Name: "users", Alias: "u"}},
		Joins: []driver.JoinClause{
			{Type: "left", FromTable: "u", FromField: "id", ToTable: "orders", ToField: "user_id"},
		},
		Fields: []string{"u.id", "u.name", "orders.total"},
		Limit: 10,
	}

	t.Run("success", func(t *testing.T) {
		expectedResult := &driver.QueryResult{
			Rows: []map[string]interface{}{
				{"id": 1, "name": "test", "total": 100},
			},
		}

		mockRepo.EXPECT().JoinQuery(ctx, req).Return(expectedResult, nil)

		result, err := service.Join(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.EXPECT().JoinQuery(ctx, gomock.Any()).Return(nil, errors.New("join failed"))

		result, err := service.Join(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCRUDService_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := driver.NewMockDatabaseDriver(ctrl)
	audit := NewAuditService("")
	cfg := &config.Config{}
	log := newTestLogger()
	service := NewCRUDService(mockRepo, audit, cfg, log)

	// Should not panic
	service.Close()
}
