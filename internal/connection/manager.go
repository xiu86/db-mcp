package connection

import (
	"context"
	"fmt"
	"time"

	"db-mcp/internal/config"
	"db-mcp/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ConnectionManager struct {
	db     *gorm.DB
	config *config.Config
	logger *logger.Logger
}

func NewConnectionManager(cfg *config.Config, log *logger.Logger) (*ConnectionManager, error) {
	dsn := BuildDSN(&cfg.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.Pool.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Pool.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Pool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Pool.ConnMaxIdleTime)

	return &ConnectionManager{db: db, config: cfg, logger: log}, nil
}

func (m *ConnectionManager) DB() *gorm.DB {
	return m.db
}

func (m *ConnectionManager) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (m *ConnectionManager) HealthCheck() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

func BuildDSN(cfg *config.DatabaseConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)
}
