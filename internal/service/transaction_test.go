//go:build integration
// +build integration

package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"db-mcp/internal/config"
	"db-mcp/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func getTestDBHost() string {
	if v := os.Getenv("TEST_DB_HOST"); v != "" {
		return v
	}
	return "127.0.0.1"
}

func getTestDBPort() int {
	if v := os.Getenv("TEST_DB_PORT"); v != "" {
		var port int
		fmt.Sscanf(v, "%d", &port)
		return port
	}
	return 3306
}

func getTestDBUser() string {
	if v := os.Getenv("TEST_DB_USER"); v != "" {
		return v
	}
	return "root"
}

func getTestDBPassword() string {
	if v := os.Getenv("TEST_DB_PASSWORD"); v != "" {
		return v
	}
	return ""
}

func getTestDBName() string {
	if v := os.Getenv("TEST_DB_NAME"); v != "" {
		return v
	}
	return "test_db"
}

func getTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host:     getTestDBHost(),
			Port:     getTestDBPort(),
			User:     getTestDBUser(),
			Password: getTestDBPassword(),
			Database: getTestDBName(),
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	cfg := getTestConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to database: %v", err)
		return nil
	}

	return db
}

func setupTransactionTestTable(t *testing.T, db *gorm.DB) {
	db.Exec("DROP TABLE IF EXISTS tx_test_users")
	db.Exec(`
		CREATE TABLE tx_test_users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100),
			status TINYINT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
}

func cleanupTransactionTestTable(db *gorm.DB) {
	db.Exec("DROP TABLE IF EXISTS tx_test_users")
}

func TestTransactionService_Begin(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("begin transaction successfully", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)

		require.NoError(t, err)
		assert.NotNil(t, txCtx)
		assert.NotNil(t, txCtx.tx)
		assert.NotNil(t, txCtx.db)
	})

	t.Run("begin transaction with db error", func(t *testing.T) {
		// Create a service with a closed database to simulate error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)

		// Database is closed, should fail
		assert.Error(t, err)
		assert.Nil(t, txCtx)
	})
}

func TestTransactionContext_Commit(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("commit successfully", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)

		// Insert data within transaction
		err = txCtx.Insert("tx_test_users", map[string]interface{}{
			"name":  "CommitTest",
			"email": "commit@test.com",
		})
		require.NoError(t, err)

		// Commit
		err = txCtx.Commit()
		require.NoError(t, err)

		// Verify data was committed
		var count int64
		db.Table("tx_test_users").Where("name = ?", "CommitTest").Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("commit without transaction", func(t *testing.T) {
		txCtx := &TransactionContext{
			tx: nil, // No transaction started
		}

		err := txCtx.Commit()
		assert.Error(t, err)
	})
}

func TestTransactionContext_Rollback(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("rollback successfully", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)

		// Insert data within transaction
		err = txCtx.Insert("tx_test_users", map[string]interface{}{
			"name":  "RollbackTest",
			"email": "rollback@test.com",
		})
		require.NoError(t, err)

		// Rollback
		err = txCtx.Rollback()
		require.NoError(t, err)

		// Verify data was rolled back
		var count int64
		db.Table("tx_test_users").Where("name = ?", "RollbackTest").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("rollback without transaction", func(t *testing.T) {
		txCtx := &TransactionContext{
			tx: nil, // No transaction started
		}

		err := txCtx.Rollback()
		assert.Error(t, err)
	})
}

func TestTransactionContext_Query(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	// Insert test data
	db.Exec("INSERT INTO tx_test_users (name, email, status) VALUES ('Alice', 'alice@test.com', 0)")
	db.Exec("INSERT INTO tx_test_users (name, email, status) VALUES ('Bob', 'bob@test.com', 1)")

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("query all fields", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)
		defer txCtx.Rollback()

		result, err := txCtx.Query("tx_test_users", nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)

		rows, ok := result.([]map[string]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(rows), 2)
	})

	t.Run("query with specific fields", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)
		defer txCtx.Rollback()

		result, err := txCtx.Query("tx_test_users", []string{"id", "name"}, nil)
		require.NoError(t, err)

		rows := result.([]map[string]interface{})
		if len(rows) > 0 {
			assert.Contains(t, rows[0], "id")
			assert.Contains(t, rows[0], "name")
		}
	})

	t.Run("query with where clause", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)
		defer txCtx.Rollback()

		result, err := txCtx.Query("tx_test_users", nil, map[string]interface{}{"status": 0})
		require.NoError(t, err)

		rows := result.([]map[string]interface{})
		for _, row := range rows {
			if status, ok := row["status"].(int64); ok {
				assert.Equal(t, int64(0), status)
			}
		}
	})

	t.Run("query without transaction", func(t *testing.T) {
		txCtx := &TransactionContext{
			tx: nil,
		}

		result, err := txCtx.Query("tx_test_users", nil, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestTransactionContext_Insert(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("insert successfully", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)
		defer txCtx.Rollback()

		err = txCtx.Insert("tx_test_users", map[string]interface{}{
			"name":  "InsertTest",
			"email": "insert@test.com",
		})
		require.NoError(t, err)

		// Commit to verify
		err = txCtx.Commit()
		require.NoError(t, err)

		var count int64
		db.Table("tx_test_users").Where("name = ?", "InsertTest").Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("insert without transaction", func(t *testing.T) {
		txCtx := &TransactionContext{
			tx: nil,
		}

		err := txCtx.Insert("tx_test_users", map[string]interface{}{
			"name": "Test",
		})
		assert.Error(t, err)
	})
}

func TestTransactionContext_Update(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	// Insert test data
	db.Exec("INSERT INTO tx_test_users (name, email, status) VALUES ('UpdateTest', 'update@test.com', 0)")

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("update successfully", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)
		defer txCtx.Rollback()

		err = txCtx.Update("tx_test_users",
			map[string]interface{}{"status": 1},
			map[string]interface{}{"name": "UpdateTest"},
		)
		require.NoError(t, err)

		// Commit to verify
		err = txCtx.Commit()
		require.NoError(t, err)

		var status int
		db.Table("tx_test_users").Where("name = ?", "UpdateTest").Select("status").Scan(&status)
		assert.Equal(t, 1, status)
	})

	t.Run("update without transaction", func(t *testing.T) {
		txCtx := &TransactionContext{
			tx: nil,
		}

		err := txCtx.Update("tx_test_users",
			map[string]interface{}{"status": 1},
			map[string]interface{}{"id": 1},
		)
		assert.Error(t, err)
	})
}

func TestTransactionContext_Delete(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTransactionTestTable(db)
	setupTransactionTestTable(t, db)

	// Insert test data
	db.Exec("INSERT INTO tx_test_users (name, email, status) VALUES ('DeleteTest', 'delete@test.com', 0)")

	cfg := getTestConfig()
	log := logger.NewLogger(&cfg.Log)
	audit := NewAuditService("")
	txService := NewTransactionService(db, audit, cfg, log)

	t.Run("delete successfully", func(t *testing.T) {
		ctx := context.Background()
		txCtx, err := txService.Begin(ctx)
		require.NoError(t, err)
		defer txCtx.Rollback()

		err = txCtx.Delete("tx_test_users", map[string]interface{}{"name": "DeleteTest"})
		require.NoError(t, err)

		// Commit to verify
		err = txCtx.Commit()
		require.NoError(t, err)

		var count int64
		db.Table("tx_test_users").Where("name = ?", "DeleteTest").Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("delete without transaction", func(t *testing.T) {
		txCtx := &TransactionContext{
			tx: nil,
		}

		err := txCtx.Delete("tx_test_users", map[string]interface{}{"id": 1})
		assert.Error(t, err)
	})
}
