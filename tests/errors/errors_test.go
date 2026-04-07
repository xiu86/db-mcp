package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	dberrors "db-mcp/internal/errors"
)

func TestNewError(t *testing.T) {
	cause := errors.New("original error")
	err := dberrors.NewError(dberrors.ErrInvalidInput, "validation failed", cause)

	assert.Equal(t, dberrors.ErrInvalidInput, err.Code)
	assert.Equal(t, "validation failed", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestError_Error(t *testing.T) {
	err := dberrors.NewError(dberrors.ErrRecordNotFound, "not found", nil)
	assert.Equal(t, "[RECORD_NOT_FOUND] not found", err.Error())

	cause := errors.New("db error")
	err = dberrors.NewError(dberrors.ErrInternal, "internal", cause)
	assert.Contains(t, err.Error(), "db error")
}

func TestWrapGormError_Nil(t *testing.T) {
	err := dberrors.WrapGormError(nil)
	assert.Nil(t, err)
}

func TestWrapGormError_DuplicateEntry(t *testing.T) {
	err := dberrors.WrapGormError(errors.New("Duplicate entry '1' for key 'PRIMARY'"))
	assert.NotNil(t, err)
	assert.Equal(t, dberrors.ErrDuplicateEntry, err.Code)
}

func TestWrapGormError_ForeignKey(t *testing.T) {
	err := dberrors.WrapGormError(errors.New("foreign key constraint fails"))
	assert.NotNil(t, err)
	assert.Equal(t, dberrors.ErrForeignKey, err.Code)
}
