//go:build integration
// +build integration

package integration

import (
	"os"
	"testing"

	"db-mcp/internal/service"
)

func TestAuditService_Operations(t *testing.T) {
	audit := service.NewAuditService(nil, "test_audit")

	// Test Start
	ctx := audit.Start("query", "users", "1")
	if ctx == nil {
		t.Error("Expected audit context, got nil")
	}

	if ctx.Operation != "query" {
		t.Errorf("Expected operation 'query', got '%s'", ctx.Operation)
	}

	if ctx.Table != "users" {
		t.Errorf("Expected table 'users', got '%s'", ctx.Table)
	}

	// Test Success
	audit.Success(ctx, map[string]interface{}{"id": 1}, map[string]interface{}{"id": 1, "name": "Test"}, 1)

	// Test Fail
	ctx2 := audit.Start("update", "users", "1")
	audit.Fail(ctx2, "connection timeout")
}

func TestMain_Audit(m *testing.M) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}
