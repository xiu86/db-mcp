package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/service"
)

func TestAuditContext(t *testing.T) {
	ctx := &service.AuditContext{
		RequestID: "req-123",
		Operation: "SELECT",
		Table:     "users",
		RecordID:  "1",
		StartTime: time.Now(),
	}

	assert.Equal(t, "req-123", ctx.RequestID)
	assert.Equal(t, "SELECT", ctx.Operation)
	assert.Equal(t, "users", ctx.Table)
}

func TestNewAuditService(t *testing.T) {
	svc := service.NewAuditService(nil, "_audit_logs")
	assert.NotNil(t, svc)
	assert.Equal(t, "_audit_logs", svc.Table)
}

func TestAuditService_Start(t *testing.T) {
	svc := service.NewAuditService(nil, "_audit_logs")
	ctx := svc.Start("SELECT", "users", "1")

	assert.NotEmpty(t, ctx.RequestID)
	assert.Equal(t, "SELECT", ctx.Operation)
	assert.Equal(t, "users", ctx.Table)
	assert.Equal(t, "1", ctx.RecordID)
}
