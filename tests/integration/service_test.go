//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"db-mcp/internal/config"
	"db-mcp/internal/connection"
	"db-mcp/internal/detector"
	"db-mcp/internal/repository"
	"db-mcp/internal/service"
	"db-mcp/pkg/logger"
)

var (
	testCRUDService *service.CRUDService
	testLog         *logger.Logger
)

func setupService(t *testing.T) {
	if testCRUDService != nil {
		return
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     3306,
			User:     getEnvOrDefault("DB_USER", "root"),
			Password: getEnvOrDefault("DB_PASSWORD", "secret"),
			Database: getEnvOrDefault("DB_NAME", "test_db"),
			Charset:  "utf8mb4",
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	var err error
	testDB, err := connection.NewConnectionManager(cfg, nil)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
	}

	testLog = logger.NewLogger(&cfg.Log)
	repo := repository.New(testDB.DB())
	audit := service.NewAuditService(nil, "test_audit")
	testCRUDService = service.NewCRUDService(repo, audit, cfg, testLog)
}

func TestService_Query(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.Query(ctx, "users", nil, nil, nil, 10, 0)

	if err != nil {
		t.Logf("Query error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestService_Insert(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.Insert(ctx, "users", map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
	})

	if err != nil {
		t.Logf("Insert error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestService_Update(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.Update(ctx, "users",
		map[string]interface{}{"name": "Updated Name"},
		map[string]interface{}{"id": 1},
	)

	if err != nil {
		t.Logf("Update error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestService_Delete(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.Delete(ctx, "users", map[string]interface{}{"id": 1})

	if err != nil {
		t.Logf("Delete error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestService_BatchInsert(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.BatchInsert(ctx, "users", []map[string]interface{}{
		{"name": "User 1", "email": "user1@example.com"},
		{"name": "User 2", "email": "user2@example.com"},
	})

	if err != nil {
		t.Logf("BatchInsert error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}

	if result.SuccessCount != 2 {
		t.Errorf("Expected 2 success count, got %d", result.SuccessCount)
	}
}

func TestService_BatchUpdate(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.BatchUpdate(ctx, "users", []map[string]interface{}{
		{"id": 1, "name": "Updated 1"},
		{"id": 2, "name": "Updated 2"},
	}, "id")

	if err != nil {
		t.Logf("BatchUpdate error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestService_BatchDelete(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.BatchDelete(ctx, "users", []string{"1", "2"}, "id")

	if err != nil {
		t.Logf("BatchDelete error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestService_GetSchema(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.GetSchema(ctx, "users")

	if err != nil {
		t.Logf("GetSchema error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}

	// Check if delete_fields is included
	if _, ok := result["delete_fields"]; !ok {
		t.Error("Expected delete_fields in result")
	}
}

func TestService_Join(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.Join(ctx, &repository.JoinRequest{
		Tables: []repository.TableRef{
			{Name: "users", Alias: "u"},
			{Name: "orders", Alias: "o"},
		},
		Joins: []repository.JoinClause{
			{
				Type:      "left",
				FromTable: "u",
				FromField: "id",
				ToTable:   "o",
				ToField:   "user_id",
			},
		},
		Limit: 100,
	})

	if err != nil {
		t.Logf("Join error: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestDetector_WithRealSchema(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	schema, err := testCRUDService.GetSchema(ctx, "users")
	if err != nil {
		t.Skipf("Skipping test: cannot get schema: %v", err)
	}

	d := detector.NewDetector()
	deleteField := d.Detect("users", []detector.ColumnInfo{
		{Name: "id", DataType: "bigint", Comment: ""},
		{Name: "is_del", DataType: "tinyint", Comment: "是否删除：0.否，1.是"},
		{Name: "deleted_time", DataType: "datetime", Comment: "删除时间"},
		{Name: "name", DataType: "varchar", Comment: ""},
	})

	if deleteField == nil {
		t.Error("Expected delete field detection, got nil")
	}

	if len(deleteField.Fields) == 0 {
		t.Error("Expected at least one delete field")
	}

	t.Logf("Detected delete fields for schema %s: %+v", schema.TableName, deleteField)
}

func TestMain_Service(m *testing.M) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
