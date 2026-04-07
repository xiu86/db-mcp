package errors

import (
	"fmt"
	"strings"
)

type ErrorCode string

const (
	ErrInvalidInput   ErrorCode = "INVALID_INPUT"
	ErrTableNotFound ErrorCode = "TABLE_NOT_FOUND"
	ErrRecordNotFound ErrorCode = "RECORD_NOT_FOUND"
	ErrDuplicateEntry ErrorCode = "DUPLICATE_ENTRY"
	ErrForeignKey    ErrorCode = "FOREIGN_KEY_ERROR"
	ErrTimeout        ErrorCode = "TIMEOUT"
	ErrRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrInternal       ErrorCode = "INTERNAL_ERROR"
)

type DBError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *DBError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DBError) Unwrap() error {
	return e.Cause
}

func NewError(code ErrorCode, message string, cause error) *DBError {
	return &DBError{Code: code, Message: message, Cause: cause}
}

func WrapGormError(err error) *DBError {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "record not found"):
		return NewError(ErrRecordNotFound, "record not found", err)
	case strings.Contains(errStr, "Duplicate entry"):
		return NewError(ErrDuplicateEntry, "duplicate entry", err)
	case strings.Contains(errStr, "foreign key constraint"):
		return NewError(ErrForeignKey, "foreign key constraint failed", err)
	default:
		return NewError(ErrInternal, "database error", err)
	}
}
