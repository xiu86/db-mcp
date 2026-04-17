package driver

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/detector"
	"db-mcp/internal/errors"
	"db-mcp/internal/sanitizer"
	"db-mcp/pkg/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// MySQLDriver MySQL驱动实现
type MySQLDriver struct {
	db     *gorm.DB
	config *config.DatabaseConfig
	logger *logger.Logger
}

// NewMySQLDriver 创建MySQL驱动
func NewMySQLDriver(cfg *config.DatabaseConfig, log *logger.Logger) (*MySQLDriver, error) {
	dsn := buildDSN(cfg)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return &MySQLDriver{db: db, config: cfg, logger: log}, nil
}

// NewMySQLDriverWithPool 创建MySQL驱动(带连接池配置)
func NewMySQLDriverWithPool(cfg *config.DatabaseConfig, poolCfg *config.PoolConfig, log *logger.Logger) (*MySQLDriver, error) {
	dsn := buildDSN(cfg)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	if poolCfg != nil {
		sqlDB.SetMaxIdleConns(poolCfg.MaxIdleConns)
		sqlDB.SetMaxOpenConns(poolCfg.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(poolCfg.ConnMaxIdleTime)
	}

	return &MySQLDriver{db: db, config: cfg, logger: log}, nil
}

// buildDSN 构建MySQL DSN
func buildDSN(cfg *config.DatabaseConfig) string {
	escapedUser := url.QueryEscape(cfg.User)
	escapedPassword := url.QueryEscape(cfg.Password)
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		escapedUser,
		escapedPassword,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)
}

// Ping 检查连接
func (d *MySQLDriver) Ping(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close 关闭连接
func (d *MySQLDriver) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DriverType 获取驱动类型
func (d *MySQLDriver) DriverType() DriverType {
	return DriverMySQL
}

// GetDB 获取GORM DB实例（向后兼容）
func (d *MySQLDriver) GetDB() *gorm.DB {
	return d.db
}

// CurrentDatabase 获取当前数据库
func (d *MySQLDriver) CurrentDatabase() string {
	return d.config.Database
}

// UseDatabase 切换数据库
func (d *MySQLDriver) UseDatabase(database string) error {
	if database == "" {
		return errors.NewError(errors.ErrInvalidInput, "database name cannot be empty", nil)
	}

	// 使用 USE DATABASE 切换
	result := d.db.Exec(fmt.Sprintf("USE `%s`", database))
	if result.Error != nil {
		return errors.WrapGormError(result.Error)
	}

	// 更新配置中的数据库名
	d.config.Database = database
	return nil
}

// Query 查询数据
func (d *MySQLDriver) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	query := d.db.Table(req.Table)

	if len(req.Where) > 0 {
		query = query.Where(req.Where)
	}

	for _, order := range req.Order {
		safeOrder, err := sanitizer.SanitizeOrderField(order.Field, order.Direction)
		if err != nil {
			return nil, err
		}
		query = query.Order(safeOrder)
	}

	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	fields := "*"
	if len(req.Fields) > 0 {
		if err := sanitizer.ValidateFieldList(req.Fields); err != nil {
			return nil, err
		}
		fields = sanitizer.QuoteFieldList(req.Fields)
	}

	err := query.Select(fields).Find(&rows).Error
	if err != nil {
		return nil, errors.WrapGormError(err)
	}

	var total int64
	d.db.Table(req.Table).Where(req.Where).Count(&total)

	return &QueryResult{Rows: rows, Total: total}, nil
}

// Insert 插入数据
func (d *MySQLDriver) Insert(ctx context.Context, req *InsertRequest) (*MutationResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}

	err := d.db.Table(req.Table).Create(req.Data).Error
	if err != nil {
		return nil, errors.WrapGormError(err)
	}

	return &MutationResult{
		AffectedRows: 1,
		LastInsertID: 1,
		Message:      "Insert successful",
	}, nil
}

// Update 更新数据
func (d *MySQLDriver) Update(ctx context.Context, req *UpdateRequest) (*MutationResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}

	result := d.db.Table(req.Table).Where(req.Where).Updates(req.Data)
	if result.Error != nil {
		return nil, errors.WrapGormError(result.Error)
	}

	return &MutationResult{
		AffectedRows: result.RowsAffected,
		Message:      "Update successful",
	}, nil
}

// Delete 逻辑删除
func (d *MySQLDriver) Delete(ctx context.Context, req *DeleteRequest) (*MutationResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}
	if req.DeleteField == nil || len(req.DeleteField.Fields) == 0 {
		return nil, errors.NewError(errors.ErrInvalidInput, "no delete field detected", nil)
	}

	updates := make(map[string]interface{})
	for _, field := range req.DeleteField.Fields {
		if field.TrueValue == detector.CurrentTimestampMarker {
			updates[field.Name] = detector.GetCurrentTimestamp()
		} else {
			updates[field.Name] = field.TrueValue
		}
	}

	result := d.db.Table(req.Table).Where(req.Where).Updates(updates)
	if result.Error != nil {
		return nil, errors.WrapGormError(result.Error)
	}

	return &MutationResult{
		AffectedRows: result.RowsAffected,
		Message:      "Logical delete successful",
	}, nil
}

// BatchInsert 批量插入
func (d *MySQLDriver) BatchInsert(ctx context.Context, req *BatchInsertRequest) (*BatchResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}

	var successCount, failedCount int64
	var batchErrors []BatchError

	for i, data := range req.Data {
		err := d.db.Table(req.Table).Create(data).Error
		if err != nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: err.Error()})
		} else {
			successCount++
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       batchErrors,
	}, nil
}

// BatchUpdate 批量更新
func (d *MySQLDriver) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}

	var successCount, failedCount int64
	var batchErrors []BatchError

	keyField := req.KeyField
	if keyField == "" {
		keyField = "id"
	}

	if err := sanitizer.ValidateFieldName(keyField); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("invalid key field: %s", keyField), err)
	}
	safeKeyField := sanitizer.QuoteIdentifier(keyField)

	for i, data := range req.Data {
		keyValue := data[keyField]
		if keyValue == nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: "key field value is nil"})
			continue
		}

		delete(data, keyField)
		result := d.db.Table(req.Table).Where(safeKeyField+" = ?", keyValue).Updates(data)
		if result.Error != nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: result.Error.Error()})
		} else {
			successCount++
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       batchErrors,
	}, nil
}

// BatchDelete 批量逻辑删除
func (d *MySQLDriver) BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchResult, error) {
	if err := sanitizer.ValidateTableName(req.Table); err != nil {
		return nil, err
	}
	if req.DeleteField == nil || len(req.DeleteField.Fields) == 0 {
		return nil, errors.NewError(errors.ErrInvalidInput, "no delete field detected", nil)
	}

	var successCount, failedCount int64
	var batchErrors []BatchError

	idField := req.IDField
	if idField == "" {
		idField = "id"
	}

	if err := sanitizer.ValidateFieldName(idField); err != nil {
		return nil, errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("invalid id field: %s", idField), err)
	}
	safeIDField := sanitizer.QuoteIdentifier(idField)

	updates := make(map[string]interface{})
	for _, field := range req.DeleteField.Fields {
		if field.TrueValue == detector.CurrentTimestampMarker {
			updates[field.Name] = detector.GetCurrentTimestamp()
		} else {
			updates[field.Name] = field.TrueValue
		}
	}

	for i, id := range req.IDs {
		result := d.db.Table(req.Table).Where(safeIDField+" = ?", id).Updates(updates)
		if result.Error != nil {
			failedCount++
			batchErrors = append(batchErrors, BatchError{Index: i, Message: result.Error.Error()})
		} else {
			successCount++
		}
	}

	return &BatchResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       batchErrors,
	}, nil
}

// JoinQuery Join查询
func (d *MySQLDriver) JoinQuery(ctx context.Context, req *JoinRequest) (*QueryResult, error) {
	if len(req.Tables) < 2 {
		return nil, errors.NewError(errors.ErrInvalidInput, "at least 2 tables required for join", nil)
	}
	if len(req.Tables) > 5 {
		return nil, errors.NewError(errors.ErrInvalidInput, "join exceeds maximum of 5 tables", nil)
	}

	// Validate all table names and aliases
	for i, tbl := range req.Tables {
		if err := sanitizer.ValidateTableName(tbl.Name); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid table name at index %d: %s", i, tbl.Name), err)
		}
		if err := sanitizer.ValidateAlias(tbl.Alias); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid alias at index %d: %s", i, tbl.Alias), err)
		}
	}

	table0 := req.Tables[0]
	query := d.db.Table(
		sanitizer.QuoteIdentifier(table0.Name)+" AS "+sanitizer.QuoteIdentifier(table0.Alias),
	)

	for i, join := range req.Joins {
		if err := sanitizer.ValidateJoinType(join.Type); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid join type at index %d: %s", i, join.Type), err)
		}
		if err := sanitizer.ValidateAlias(join.FromTable); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid from table alias at join index %d: %s", i, join.FromTable), err)
		}
		if err := sanitizer.ValidateColumnName(join.FromField); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid from field at join index %d: %s", i, join.FromField), err)
		}
		if err := sanitizer.ValidateAlias(join.ToTable); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid to table alias at join index %d: %s", i, join.ToTable), err)
		}
		if err := sanitizer.ValidateColumnName(join.ToField); err != nil {
			return nil, errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid to field at join index %d: %s", i, join.ToField), err)
		}

		joinType := "INNER JOIN"
		switch join.Type {
		case "left":
			joinType = "LEFT JOIN"
		case "right":
			joinType = "RIGHT JOIN"
		}

		query = query.Joins(fmt.Sprintf("%s %s ON %s.%s = %s.%s",
			joinType,
			sanitizer.QuoteIdentifier(join.ToTable),
			sanitizer.QuoteIdentifier(join.FromTable),
			sanitizer.QuoteIdentifier(join.FromField),
			sanitizer.QuoteIdentifier(join.ToTable),
			sanitizer.QuoteIdentifier(join.ToField),
		))
	}

	if len(req.Where) > 0 {
		query = query.Where(req.Where)
	}

	for _, order := range req.Order {
		safeOrder, err := sanitizer.SanitizeOrderField(order.Field, order.Direction)
		if err != nil {
			return nil, err
		}
		query = query.Order(safeOrder)
	}

	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}

	fields := "*"
	if len(req.Fields) > 0 {
		if err := sanitizer.ValidateFieldList(req.Fields); err != nil {
			return nil, err
		}
		fields = sanitizer.QuoteFieldList(req.Fields)
	}

	var rows []map[string]interface{}
	err := query.Select(fields).Find(&rows).Error
	if err != nil {
		return nil, errors.WrapGormError(err)
	}

	return &QueryResult{Rows: rows, Total: int64(len(rows))}, nil
}

// GetTableSchema 获取表结构
func (d *MySQLDriver) GetTableSchema(tableName string) (*TableSchema, error) {
	if err := sanitizer.ValidateTableName(tableName); err != nil {
		return nil, err
	}

	var results []struct {
		Field          string `gorm:"column:COLUMN_NAME"`
		Type           string `gorm:"column:DATA_TYPE"`
		Null           string `gorm:"column:IS_NULLABLE"`
		Key            string `gorm:"column:COLUMN_KEY"`
		Default        *string `gorm:"column:COLUMN_DEFAULT"`
		Extra          string `gorm:"column:EXTRA"`
		Comment        string `gorm:"column:COLUMN_COMMENT"`
	}

	err := d.db.Table("information_schema.COLUMNS").
		Select("COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA, COLUMN_DEFAULT, COLUMN_COMMENT").
		Where("TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", tableName).
		Scan(&results).Error

	if err != nil {
		return nil, errors.WrapGormError(err)
	}

	var columns []ColumnInfo
	for _, col := range results {
		columns = append(columns, ColumnInfo{
			Name:          col.Field,
			DataType:      col.Type,
			IsNullable:    col.Null,
			ColumnKey:     col.Key,
			Extra:         col.Extra,
			ColumnDefault: col.Default,
			Comment:       col.Comment,
		})
	}

	return &TableSchema{
		TableName: tableName,
		Columns:   columns,
	}, nil
}
