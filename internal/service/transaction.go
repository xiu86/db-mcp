package service

import (
	"context"
	"db-mcp/internal/config"
	"db-mcp/internal/detector"
	"db-mcp/internal/errors"
	"db-mcp/internal/sanitizer"
	"db-mcp/pkg/logger"

	"gorm.io/gorm"
)

type TransactionService struct {
	db       *gorm.DB
	audit    *AuditService
	detector *detector.DeleteFieldDetector
	config   *config.Config
	logger   *logger.Logger
}

type TransactionContext struct {
	db       *gorm.DB
	tx       *gorm.DB
	audit    *AuditService
	detector *detector.DeleteFieldDetector
	logger   *logger.Logger
}

func NewTransactionService(db *gorm.DB, audit *AuditService, cfg *config.Config, log *logger.Logger) *TransactionService {
	return &TransactionService{
		db:       db,
		audit:    audit,
		detector: detector.NewDetector(),
		config:   cfg,
		logger:   log,
	}
}

func (s *TransactionService) Begin(ctx context.Context) (*TransactionContext, error) {
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.WrapGormError(tx.Error)
	}

	return &TransactionContext{
		db:       s.db,
		tx:       tx,
		audit:    s.audit,
		detector: s.detector,
		logger:   s.logger,
	}, nil
}

func (tc *TransactionContext) Commit() error {
	if tc.tx == nil {
		return errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
	}

	if err := tc.tx.Commit().Error; err != nil {
		return errors.WrapGormError(err)
	}

	tc.tx = nil
	return nil
}

func (tc *TransactionContext) Rollback() error {
	if tc.tx == nil {
		return errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
	}

	if err := tc.tx.Rollback().Error; err != nil {
		return errors.WrapGormError(err)
	}

	tc.tx = nil
	return nil
}

func (tc *TransactionContext) Query(table string, fields []string, where map[string]interface{}) (interface{}, error) {
	if tc.tx == nil {
		return nil, errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
	}
	if err := sanitizer.ValidateTableName(table); err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	query := tc.tx.Table(table)

	if len(where) > 0 {
		query = query.Where(where)
	}

	fieldsStr := "*"
	if len(fields) > 0 {
		if err := sanitizer.ValidateFieldList(fields); err != nil {
			return nil, err
		}
		fieldsStr = sanitizer.QuoteFieldList(fields)
	}

	if err := query.Select(fieldsStr).Find(&rows).Error; err != nil {
		return nil, errors.WrapGormError(err)
	}

	return rows, nil
}

func (tc *TransactionContext) Insert(table string, data map[string]interface{}) error {
	if tc.tx == nil {
		return errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
	}
	if err := sanitizer.ValidateTableName(table); err != nil {
		return err
	}

	if err := tc.tx.Table(table).Create(data).Error; err != nil {
		return errors.WrapGormError(err)
	}

	return nil
}

func (tc *TransactionContext) Update(table string, data, where map[string]interface{}) error {
	if tc.tx == nil {
		return errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
	}
	if err := sanitizer.ValidateTableName(table); err != nil {
		return err
	}

	if err := tc.tx.Table(table).Where(where).Updates(data).Error; err != nil {
		return errors.WrapGormError(err)
	}

	return nil
}

func (tc *TransactionContext) Delete(table string, where map[string]interface{}) error {
	if tc.tx == nil {
		return errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
	}
	if err := sanitizer.ValidateTableName(table); err != nil {
		return err
	}

	// Detect and use logical delete fields
	deleteField := tc.detector.Detect(table, tc.getTableColumns(table))
	if deleteField == nil || len(deleteField.Fields) == 0 {
		return errors.NewError(errors.ErrInvalidInput, "no delete field detected", nil)
	}

	updates := make(map[string]interface{})
	for _, field := range deleteField.Fields {
		if field.TrueValue == detector.CurrentTimestampMarker {
			updates[field.Name] = detector.GetCurrentTimestamp()
		} else {
			updates[field.Name] = field.TrueValue
		}
	}

	if err := tc.tx.Table(table).Where(where).Updates(updates).Error; err != nil {
		return errors.WrapGormError(err)
	}

	return nil
}

func (tc *TransactionContext) getTableColumns(table string) []detector.ColumnInfo {
	var results []struct {
		Field   string `gorm:"column:COLUMN_NAME"`
		Type    string `gorm:"column:DATA_TYPE"`
		Comment string `gorm:"column:COLUMN_COMMENT"`
	}

	tc.db.Table("information_schema.COLUMNS").
		Select("COLUMN_NAME, DATA_TYPE, COLUMN_COMMENT").
		Where("TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", table).
		Scan(&results)

	var cols []detector.ColumnInfo
	for _, r := range results {
		cols = append(cols, detector.ColumnInfo{
			Name:     r.Field,
			DataType: r.Type,
			Comment:  r.Comment,
		})
	}
	return cols
}
