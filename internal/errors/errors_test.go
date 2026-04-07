package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrInvalidInput, "test message", nil)

	assert.NotNil(t, err)
	assert.Equal(t, ErrInvalidInput, err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Contains(t, err.Error(), "test message")
}

func TestNewError_WithCause(t *testing.T) {
	cause := errors.New("original error")
	err := NewError(ErrInvalidInput, "test message", cause)

	assert.Equal(t, cause, err.Cause)
	assert.Contains(t, err.Error(), "original error")
}

func TestWrapGormError_RecordNotFound(t *testing.T) {
	gormErr := errors.New("record not found")
	wrapped := WrapGormError(gormErr)

	assert.NotNil(t, wrapped)
	assert.Equal(t, ErrRecordNotFound, wrapped.Code)
	assert.Contains(t, wrapped.Message, "record not found")
}

func TestWrapGormError_DuplicateEntry(t *testing.T) {
	gormErr := errors.New("Duplicate entry 'test' for key 'PRIMARY'")
	wrapped := WrapGormError(gormErr)

	assert.NotNil(t, wrapped)
	assert.Equal(t, ErrDuplicateEntry, wrapped.Code)
	assert.Contains(t, wrapped.Message, "duplicate entry")
}

func TestWrapGormError_ForeignKey(t *testing.T) {
	gormErr := errors.New("foreign key constraint fails")
	wrapped := WrapGormError(gormErr)

	assert.NotNil(t, wrapped)
	assert.Equal(t, ErrForeignKey, wrapped.Code)
}

func TestWrapGormError_OtherError(t *testing.T) {
	gormErr := errors.New("some database error")
	wrapped := WrapGormError(gormErr)

	assert.NotNil(t, wrapped)
	assert.Equal(t, ErrInternal, wrapped.Code)
}

func TestWrapGormError_NilError(t *testing.T) {
	wrapped := WrapGormError(nil)
	assert.Nil(t, wrapped)
}

func TestDBError_Error(t *testing.T) {
	testCases := []struct {
		code    ErrorCode
		message string
		expect  string
	}{
		{ErrInvalidInput, "invalid input", "INVALID_INPUT"},
		{ErrTableNotFound, "table not found", "TABLE_NOT_FOUND"},
		{ErrRecordNotFound, "record not found", "RECORD_NOT_FOUND"},
		{ErrTimeout, "timeout", "TIMEOUT"},
		{ErrRateLimit, "rate limited", "RATE_LIMIT_EXCEEDED"},
		{ErrInternal, "internal error", "INTERNAL_ERROR"},
	}

	for _, tc := range testCases {
		err := NewError(tc.code, tc.message, nil)
		assert.Contains(t, err.Error(), tc.expect)
	}
}

func TestDBError_Unwrap(t *testing.T) {
	cause := errors.New("original")
	err := NewError(ErrInvalidInput, "test", cause)

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestErrorCodes(t *testing.T) {
	// Verify all error codes are unique strings
	codes := []ErrorCode{
		ErrInvalidInput,
		ErrTableNotFound,
		ErrRecordNotFound,
		ErrDuplicateEntry,
		ErrForeignKey,
		ErrTimeout,
		ErrRateLimit,
		ErrInternal,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		codeStr := string(code)
		assert.False(t, seen[codeStr], "Duplicate error code: %s", codeStr)
		seen[codeStr] = true
	}
}
