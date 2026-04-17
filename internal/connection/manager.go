package connection

import (
	"context"
	"fmt"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/driver"
	"db-mcp/pkg/logger"
	"gorm.io/gorm"
)

type Instance struct {
	Name   string
	Driver driver.DatabaseDriver
	Config config.InstanceConfig
}

type ConnectionManager struct {
	instances map[string]*Instance
	current  string
	config   *config.Config
	logger   *logger.Logger
}

func NewConnectionManager(cfg *config.Config, log *logger.Logger) (*ConnectionManager, error) {
	manager := &ConnectionManager{
		instances: make(map[string]*Instance),
		current:   "default",
		config:    cfg,
		logger:    log,
	}

	if len(cfg.Databases) == 0 {
		instance, err := createDefaultInstance(cfg, log)
		if err != nil {
			return nil, err
		}
		manager.instances["default"] = instance
	} else {
		defaultSet := false
		for _, dbCfg := range cfg.Databases {
			instance, err := createInstance(&dbCfg, &cfg.Pool, log)
			if err != nil {
				return nil, fmt.Errorf("failed to create instance %s: %w", dbCfg.Name, err)
			}
			manager.instances[dbCfg.Name] = instance

			if !defaultSet {
				manager.current = dbCfg.Name
				defaultSet = true
			}
		}

		if cfg.Default != "" {
			if _, ok := manager.instances[cfg.Default]; !ok {
				return nil, fmt.Errorf("default instance %s not found", cfg.Default)
			}
			manager.current = cfg.Default
		}
	}

	return manager, nil
}

func createDefaultInstance(cfg *config.Config, log *logger.Logger) (*Instance, error) {
	dbCfg := config.InstanceConfig{
		Type:     "mysql",
		Name:     "default",
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
		Charset:  cfg.Database.Charset,
	}

	dbDriver, err := driver.NewMySQLDriverWithPool(&config.DatabaseConfig{
		Host:     dbCfg.Host,
		Port:     dbCfg.Port,
		User:     dbCfg.User,
		Password: dbCfg.Password,
		Database: dbCfg.Database,
		Charset:  dbCfg.Charset,
	}, &cfg.Pool, log)
	if err != nil {
		return nil, err
	}

	return &Instance{
		Name:   dbCfg.Name,
		Driver: dbDriver,
		Config: dbCfg,
	}, nil
}

func createInstance(dbCfg *config.InstanceConfig, poolCfg *config.PoolConfig, log *logger.Logger) (*Instance, error) {
	switch dbCfg.Type {
	case "mysql":
		mysqlDriver, err := driver.NewMySQLDriverWithPool(&config.DatabaseConfig{
			Host:     dbCfg.Host,
			Port:     dbCfg.Port,
			User:     dbCfg.User,
			Password: dbCfg.Password,
			Database: dbCfg.Database,
			Charset:  dbCfg.Charset,
		}, poolCfg, log)
		if err != nil {
			return nil, err
		}
		return &Instance{
			Name:   dbCfg.Name,
			Driver: mysqlDriver,
			Config: *dbCfg,
		}, nil
	case "mongodb":
		mongoDriver, err := driver.NewMongoDriver(&config.MongoConfig{
			URI:         dbCfg.URI,
			Database:    dbCfg.Database,
			MaxPoolSize: dbCfg.MaxPoolSize,
			MinPoolSize: dbCfg.MinPoolSize,
		}, log)
		if err != nil {
			return nil, err
		}
		return &Instance{
			Name:   dbCfg.Name,
			Driver: mongoDriver,
			Config: *dbCfg,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}
}

func (m *ConnectionManager) DB() *gorm.DB {
	if instance, ok := m.instances[m.current]; ok {
		if mysqlDriver, ok := instance.Driver.(*driver.MySQLDriver); ok {
			return mysqlDriver.GetDB()
		}
	}
	return nil
}

func (m *ConnectionManager) GetDriver(name string) (driver.DatabaseDriver, error) {
	if name == "" {
		name = m.current
	}
	instance, ok := m.instances[name]
	if !ok {
		return nil, fmt.Errorf("instance %s not found", name)
	}
	return instance.Driver, nil
}

func (m *ConnectionManager) SwitchInstance(name string) error {
	if _, ok := m.instances[name]; !ok {
		return fmt.Errorf("instance %s not found", name)
	}
	m.current = name
	return nil
}

func (m *ConnectionManager) ListInstances() []string {
	names := make([]string, 0, len(m.instances))
	for name := range m.instances {
		names = append(names, name)
	}
	return names
}

func (m *ConnectionManager) CurrentInstance() string {
	return m.current
}

func (m *ConnectionManager) Ping() error {
	instance, ok := m.instances[m.current]
	if !ok {
		return fmt.Errorf("current instance %s not found", m.current)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return instance.Driver.Ping(ctx)
}

func (m *ConnectionManager) HealthCheck() error {
	return m.Ping()
}

func (m *ConnectionManager) Close() error {
	var lastErr error
	for _, instance := range m.instances {
		if err := instance.Driver.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
