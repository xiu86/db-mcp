package service

import (
	"context"
	"db-mcp/internal/config"
	"db-mcp/internal/errors"
	"db-mcp/pkg/logger"

	"gorm.io/gorm"
)

type TransactionService struct {
	db     *gorm.DB
	audit  *AuditService
	config *config.Config
	logger *logger.Logger
}

type TransactionContext struct {
	db     *gorm.DB
	tx     *gorm.DB
	audit  *AuditService
	logger *logger.Logger
}

func NewTransactionService(db *gorm.DB, audit *AuditService, cfg *config.Config, log *logger.Logger) *TransactionService {
	return &TransactionService{
		db:     db,
		audit:  audit,
		config: cfg,
		logger: log,
	}
}

func (s *TransactionService) Begin(ctx context.Context) (*TransactionContext, error) {
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.WrapGormError(tx.Error)
	}

	return &TransactionContext{
		db:     s.db,
		tx:     tx,
		audit:  s.audit,
		logger: s.logger,
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

	var rows []map[string]interface{}
	query := tc.tx.Table(table)

	if len(where) > 0 {
		query = query.Where(where)
	}

	fieldsStr := "*"
	if len(fields) > 0 {
		fieldsStr = joinFields(fields)
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

	if err := tc.tx.Table(table).Create(data).Error; err != nil {
		return errors.WrapGormError(err)
	}

	return nil
}

func (tc *TransactionContext) Update(table string, data, where map[string]interface{}) error {
	if tc.tx == nil {
		return errors.NewError(errors.ErrInvalidInput, "transaction not started", nil)
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

	if err := tc.tx.Table(table).Where(where).Delete(nil).Error; err != nil {
		return errors.WrapGormError(err)
	}

	return nil
}

func joinFields(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}
