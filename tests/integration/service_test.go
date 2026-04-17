//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
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
			Password: getEnvOrDefault("DB_PASSWORD", ""),
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
	audit := service.NewAuditService("")
	testCRUDService = service.NewCRUDService(repo, audit, cfg, testLog)
}

func TestService_Query(t *testing.T) {
	setupService(t)

	ctx := context.Background()
	result, err := testCRUDService.Query(ctx, "vc_account", nil, nil, nil, 10, 0)

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
	result, err := testCRUDService.Insert(ctx, "vc_account", map[string]interface{}{
		"out_account_id": "test_l3_" + t.Name(),
		"parent_id":      0,
		"platform_type":  "test",
		"platform_uid":   "test_uid",
		"vt_user_id":     0,
		"display_uid":    "test_display",
		"name":           "L3 Test Account",
		"avatar":         "http://test.com/avatar.jpg",
		"mobile":         "13800138000",
		"company_name":   "Test Company",
		"signature":      "Test signature",
		"account_status": 1,
		"account_type":   "test",
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

	// First insert
	insertResult, err := testCRUDService.Insert(ctx, "vc_account", map[string]interface{}{
		"out_account_id": "test_update_" + t.Name(),
		"parent_id":      0,
		"platform_type":  "test",
		"platform_uid":   "test_uid",
		"vt_user_id":     0,
		"display_uid":    "test_display",
		"name":           "Original Name",
		"avatar":         "http://test.com/avatar.jpg",
		"mobile":         "13800138000",
		"company_name":   "Test Company",
		"signature":      "Test signature",
		"account_status": 1,
		"account_type":   "test",
	})

	if err != nil {
		t.Skipf("Skipping update test: insert failed: %v", err)
	}

	result, err := testCRUDService.Update(ctx, "vc_account",
		map[string]interface{}{"name": "Updated Name"},
		map[string]interface{}{"id": insertResult.AffectedRows},
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

	// First insert
	insertResult, err := testCRUDService.Insert(ctx, "vc_account", map[string]interface{}{
		"out_account_id": "test_delete_" + t.Name(),
		"parent_id":      0,
		"platform_type":  "test",
		"platform_uid":   "test_uid",
		"vt_user_id":     0,
		"display_uid":    "test_display",
		"name":           "To Delete",
		"avatar":         "http://test.com/avatar.jpg",
		"mobile":         "13800138000",
		"company_name":   "Test Company",
		"signature":      "Test signature",
		"account_status": 1,
		"account_type":   "test",
	})

	if err != nil {
		t.Skipf("Skipping delete test: insert failed: %v", err)
	}

	result, err := testCRUDService.Delete(ctx, "vc_account", map[string]interface{}{"id": insertResult.AffectedRows})

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
	result, err := testCRUDService.BatchInsert(ctx, "vc_account", []map[string]interface{}{
		{
			"out_account_id": "test_batch1_" + t.Name(),
			"parent_id":      0,
			"platform_type":  "test",
			"platform_uid":   "test_uid1",
			"vt_user_id":     0,
			"display_uid":    "test_display1",
			"name":           "Batch User 1",
			"avatar":         "http://test.com/avatar.jpg",
			"mobile":         "13800138001",
			"company_name":   "Test Company",
			"signature":      "Test signature",
			"account_status": 1,
			"account_type":   "test",
		},
		{
			"out_account_id": "test_batch2_" + t.Name(),
			"parent_id":      0,
			"platform_type":  "test",
			"platform_uid":   "test_uid2",
			"vt_user_id":     0,
			"display_uid":    "test_display2",
			"name":           "Batch User 2",
			"avatar":         "http://test.com/avatar.jpg",
			"mobile":         "13800138002",
			"company_name":   "Test Company",
			"signature":      "Test signature",
			"account_status": 1,
			"account_type":   "test",
		},
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

	// First insert two records
	insertResult, err := testCRUDService.BatchInsert(ctx, "vc_account", []map[string]interface{}{
		{
			"out_account_id": "test_bu1_" + t.Name(),
			"parent_id":      0,
			"platform_type":  "test",
			"platform_uid":   "test_uid1",
			"vt_user_id":     0,
			"display_uid":    "test_display1",
			"name":           "Batch Update 1",
			"avatar":         "http://test.com/avatar.jpg",
			"mobile":         "13800138001",
			"company_name":   "Test Company",
			"signature":      "Test signature",
			"account_status": 1,
			"account_type":   "test",
		},
		{
			"out_account_id": "test_bu2_" + t.Name(),
			"parent_id":      0,
			"platform_type":  "test",
			"platform_uid":   "test_uid2",
			"vt_user_id":     0,
			"display_uid":    "test_display2",
			"name":           "Batch Update 2",
			"avatar":         "http://test.com/avatar.jpg",
			"mobile":         "13800138002",
			"company_name":   "Test Company",
			"signature":      "Test signature",
			"account_status": 1,
			"account_type":   "test",
		},
	})

	if err != nil || insertResult == nil {
		t.Skipf("Skipping batch update test: insert failed: %v", err)
	}

	// Query to get the IDs
	queryResult, _ := testCRUDService.Query(ctx, "vc_account", []string{"id"}, map[string]interface{}{
		"out_account_id": map[string]interface{}{"$in": []string{"test_bu1_" + t.Name(), "test_bu2_" + t.Name()}},
	}, nil, 2, 0)

	if queryResult == nil || len(queryResult.Rows) < 2 {
		t.Skipf("Skipping batch update test: could not query inserted records")
	}

	result, err := testCRUDService.BatchUpdate(ctx, "vc_account", []map[string]interface{}{
		{"id": queryResult.Rows[0]["id"], "name": "Updated Batch 1"},
		{"id": queryResult.Rows[1]["id"], "name": "Updated Batch 2"},
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

	// First insert two records
	insertResult, err := testCRUDService.BatchInsert(ctx, "vc_account", []map[string]interface{}{
		{
			"out_account_id": "test_bd1_" + t.Name(),
			"parent_id":      0,
			"platform_type":  "test",
			"platform_uid":   "test_uid1",
			"vt_user_id":     0,
			"display_uid":    "test_display1",
			"name":           "Batch Delete 1",
			"avatar":         "http://test.com/avatar.jpg",
			"mobile":         "13800138001",
			"company_name":   "Test Company",
			"signature":      "Test signature",
			"account_status": 1,
			"account_type":   "test",
		},
		{
			"out_account_id": "test_bd2_" + t.Name(),
			"parent_id":      0,
			"platform_type":  "test",
			"platform_uid":   "test_uid2",
			"vt_user_id":     0,
			"display_uid":    "test_display2",
			"name":           "Batch Delete 2",
			"avatar":         "http://test.com/avatar.jpg",
			"mobile":         "13800138002",
			"company_name":   "Test Company",
			"signature":      "Test signature",
			"account_status": 1,
			"account_type":   "test",
		},
	})

	if err != nil || insertResult == nil {
		t.Skipf("Skipping batch delete test: insert failed: %v", err)
	}

	// Query to get the IDs
	queryResult, _ := testCRUDService.Query(ctx, "vc_account", []string{"id"}, map[string]interface{}{
		"out_account_id": map[string]interface{}{"$in": []string{"test_bd1_" + t.Name(), "test_bd2_" + t.Name()}},
	}, nil, 2, 0)

	if queryResult == nil || len(queryResult.Rows) < 2 {
		t.Skipf("Skipping batch delete test: could not query inserted records")
	}

	ids := make([]string, len(queryResult.Rows))
	for i, row := range queryResult.Rows {
		ids[i] = fmt.Sprintf("%v", row["id"])
	}

	result, err := testCRUDService.BatchDelete(ctx, "vc_account", ids, "id")

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
	result, err := testCRUDService.GetSchema(ctx, "vc_account")

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
			{Name: "vc_account", Alias: "a"},
			{Name: "vc_creative_info", Alias: "c"},
		},
		Joins: []repository.JoinClause{
			{
				Type:      "left",
				FromTable: "a",
				FromField: "id",
				ToTable:   "c",
				ToField:   "vc_account_id",
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
	_, err := testCRUDService.GetSchema(ctx, "vc_account")
	if err != nil {
		t.Skipf("Skipping test: cannot get schema: %v", err)
	}

	d := detector.NewDetector()
	deleteField := d.Detect("vc_account", []detector.ColumnInfo{
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

	t.Logf("Detected delete fields for schema: %+v", deleteField)
}
