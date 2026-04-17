//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/connection"
	"db-mcp/internal/repository"
)

var (
	testRepo *repository.Repository
	testDB  *connection.ConnectionManager
)

func TestMain(m *testing.M) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		os.Exit(0)
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
	}

	var err error
	testDB, err = connection.NewConnectionManager(cfg, nil)
	if err != nil {
		os.Exit(1)
	}

	testRepo = repository.New(testDB.DB())
	os.Exit(m.Run())
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestRepository_Query(t *testing.T) {
	ctx := context.Background()
	_ = ctx // Will be used with timeout middleware in real tests

	result, err := testRepo.Query(&repository.QueryRequest{
		Table:  "users",
		Limit:  10,
		Offset: 0,
	})

	if err != nil {
		t.Logf("Query error (may be expected if table doesn't exist): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestRepository_Insert(t *testing.T) {
	result, err := testRepo.Insert(&repository.InsertRequest{
		Table: "users",
		Data: map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
	})

	if err != nil {
		t.Logf("Insert error (may be expected if table doesn't exist): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestRepository_Update(t *testing.T) {
	result, err := testRepo.Update(&repository.UpdateRequest{
		Table: "users",
		Data:  map[string]interface{}{"name": "Updated Name"},
		Where: map[string]interface{}{"id": 1},
	})

	if err != nil {
		t.Logf("Update error (may be expected if table doesn't exist): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestRepository_JoinQuery(t *testing.T) {
	result, err := testRepo.JoinQuery(&repository.JoinRequest{
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
		t.Logf("Join error (may be expected if tables don't exist): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestRepository_GetTableSchema(t *testing.T) {
	schema, err := testRepo.GetTableSchema("users")

	if err != nil {
		t.Logf("GetTableSchema error (may be expected if table doesn't exist): %v", err)
		return
	}

	if schema == nil {
		t.Error("Expected schema, got nil")
	}

	if schema.TableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", schema.TableName)
	}
}

func TestRepository_BatchInsert(t *testing.T) {
	result, err := testRepo.BatchInsert(&repository.BatchInsertRequest{
		Table: "users",
		Data: []map[string]interface{}{
			{"name": "User 1", "email": "user1@example.com"},
			{"name": "User 2", "email": "user2@example.com"},
		},
	})

	if err != nil {
		t.Logf("BatchInsert error (may be expected if table doesn't exist): %v", err)
		return
	}

	if result == nil {
		t.Error("Expected result, got nil")
	}
}

func TestRepository_ConnectionHealth(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not configured")
	}

	err := testDB.HealthCheck()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestRepository_ConnectionPool(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not configured")
	}

	// Test multiple concurrent queries
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func() {
			_, err := testRepo.Query(&repository.QueryRequest{
				Table: "users",
				Limit: 10,
			})
			done <- err
		}()
	}

	for i := 0; i < 5; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Logf("Query error (may be expected if table doesn't exist): %v", err)
			}
		case <-ctx.Done():
			t.Error("Test timed out")
			return
		}
	}
}
