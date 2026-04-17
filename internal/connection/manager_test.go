package connection

import (
	"context"
	"fmt"
	"testing"
	"time"

	"db-mcp/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func getTestDBHost() string {
	return "127.0.0.1"
}

func getTestDBPort() int {
	return 3306
}

func getTestDBUser() string {
	return "root"
}

func getTestDBPassword() string {
	return "123456"
}

func getTestDBName() string {
	return "video-core_gzminjieadmin_test"
}

func buildTestDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		getTestDBUser(), getTestDBPassword(), getTestDBHost(), getTestDBPort(), getTestDBName())
}

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := buildTestDSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to database: %v", err)
		return nil
	}
	return db
}

func TestBuildDSN(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "testdb",
		Charset:  "utf8mb4",
	}

	dsn := BuildDSN(cfg)

	assert.Contains(t, dsn, "root:secret")
	assert.Contains(t, dsn, "tcp(localhost:3306)")
	assert.Contains(t, dsn, "testdb")
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "parseTime=True")
	assert.Contains(t, dsn, "loc=Local")
}

func TestBuildDSN_SpecialCharacters(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "db.example.com",
		Port:     3307,
		User:     "admin",
		Password: "p@ss!word",
		Database: "production_db",
		Charset:  "utf8",
	}

	dsn := BuildDSN(cfg)

	assert.Contains(t, dsn, "admin:p%40ss%21word")
	assert.Contains(t, dsn, "tcp(db.example.com:3307)")
	assert.Contains(t, dsn, "production_db")
}

func TestBuildDSN_DifferentPorts(t *testing.T) {
	testCases := []struct {
		port    int
		pattern string
	}{
		{3306, "tcp(localhost:3306)"},
		{3307, "tcp(localhost:3307)"},
		{5432, "tcp(localhost:5432)"},
	}

	for _, tc := range testCases {
		cfg := &config.DatabaseConfig{
			Host:     "localhost",
			Port:     tc.port,
			User:     "root",
			Password: "secret",
			Database: "testdb",
			Charset:  "utf8mb4",
		}

		dsn := BuildDSN(cfg)
		assert.Contains(t, dsn, tc.pattern, "Port %d should be in DSN", tc.port)
	}
}

func TestNewConnectionManager_InvalidDSN(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "invalid-host-that-does-not-exist",
			Port:     99999,
			User:     "root",
			Password: "secret",
			Database: "testdb",
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	// This should fail because the host is invalid
	_, err := NewConnectionManager(cfg, nil)
	assert.Error(t, err)
}

func TestConnectionManager_DB(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getTestDBHost(),
			Port:     getTestDBPort(),
			User:     getTestDBUser(),
			Password: getTestDBPassword(),
			Database: getTestDBName(),
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	mgr, err := NewConnectionManager(cfg, nil)
	require.NoError(t, err)

	t.Run("get database connection", func(t *testing.T) {
		connDB := mgr.DB()
		assert.NotNil(t, connDB)

		// Test that we can use the connection
		sqlDB, err := connDB.DB()
		require.NoError(t, err)
		assert.NotNil(t, sqlDB)
	})
}

func TestConnectionManager_Close(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getTestDBHost(),
			Port:     getTestDBPort(),
			User:     getTestDBUser(),
			Password: getTestDBPassword(),
			Database: getTestDBName(),
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	mgr, err := NewConnectionManager(cfg, nil)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
		return
	}

	t.Run("close connection", func(t *testing.T) {
		// Get the connection first
		connDB := mgr.DB()
		require.NotNil(t, connDB)

		// Close
		err := mgr.Close()
		assert.NoError(t, err)
	})

	t.Run("health check after close", func(t *testing.T) {
		// HealthCheck should fail after close
		err := mgr.HealthCheck()
		assert.Error(t, err)
	})
}

func TestConnectionManager_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getTestDBHost(),
			Port:     getTestDBPort(),
			User:     getTestDBUser(),
			Password: getTestDBPassword(),
			Database: getTestDBName(),
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	mgr, err := NewConnectionManager(cfg, nil)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
		return
	}

	t.Run("health check success", func(t *testing.T) {
		err := mgr.HealthCheck()
		assert.NoError(t, err)
	})

	t.Run("health check with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test HealthCheck via DB method
		db := mgr.DB()
		require.NotNil(t, db)

		sqlDB, err := db.DB()
		require.NoError(t, err)

		err = sqlDB.PingContext(ctx)
		assert.NoError(t, err)
	})
}

func TestConnectionManager_Operations(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	// Create test table
	db.Exec("DROP TABLE IF EXISTS connection_test")
	db.Exec(`
		CREATE TABLE connection_test (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	defer db.Exec("DROP TABLE IF EXISTS connection_test")

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getTestDBHost(),
			Port:     getTestDBPort(),
			User:     getTestDBUser(),
			Password: getTestDBPassword(),
			Database: getTestDBName(),
			Charset:  "utf8mb4",
		},
		Pool: config.PoolConfig{
			MaxIdleConns:    5,
			MaxOpenConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	mgr, err := NewConnectionManager(cfg, nil)
	require.NoError(t, err)
	defer mgr.Close()

	t.Run("perform query operation", func(t *testing.T) {
		// Insert test data
		db.Exec("INSERT INTO connection_test (name) VALUES ('Test1')")

		// Use the connection
		connDB := mgr.DB()
		require.NotNil(t, connDB)

		var count int64
		err := connDB.Table("connection_test").Count(&count).Error
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})

	t.Run("connection remains healthy after operations", func(t *testing.T) {
		err := mgr.HealthCheck()
		assert.NoError(t, err)
	})
}
