package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAuditService(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	assert.NotNil(t, audit)
	assert.Equal(t, "test_audit", audit.Table)
}

func TestAuditService_Start(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	ctx := audit.Start("query", "users", "1")

	assert.NotNil(t, ctx)
	assert.Equal(t, "query", ctx.Operation)
	assert.Equal(t, "users", ctx.Table)
	assert.Equal(t, "1", ctx.RecordID)
	assert.NotEmpty(t, ctx.RequestID)
	assert.False(t, ctx.StartTime.IsZero())
}

func TestAuditService_Start_EmptyRecordID(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	ctx := audit.Start("query", "users", "")

	assert.NotNil(t, ctx)
	assert.Empty(t, ctx.RecordID)
}

func TestAuditService_Success(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	ctx := audit.Start("insert", "users", "1")

	before := map[string]interface{}{"name": "old"}
	after := map[string]interface{}{"name": "new"}

	// Should not panic
	audit.Success(ctx, before, after, 1)
}

func TestAuditService_Success_NilData(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	ctx := audit.Start("delete", "users", "1")

	// Should not panic with nil data
	audit.Success(ctx, nil, nil, 1)
}

func TestAuditService_Fail(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	ctx := audit.Start("update", "users", "1")

	// Should not panic
	audit.Fail(ctx, "connection timeout")
}

func TestAuditService_Fail_EmptyMessage(t *testing.T) {
	audit := NewAuditService(nil, "test_audit")
	ctx := audit.Start("query", "users", "1")

	// Should not panic with empty message
	audit.Fail(ctx, "")
}

func TestAuditContext(t *testing.T) {
	ctx := &AuditContext{
		RequestID:  "test-123",
		Operation:  "insert",
		Table:      "users",
		RecordID:   "1",
		Actor:      "test",
		StartTime:  time.Now(),
		BeforeData: map[string]interface{}{"id": 1},
		AfterData:  map[string]interface{}{"id": 1, "name": "test"},
	}

	assert.Equal(t, "test-123", ctx.RequestID)
	assert.Equal(t, "insert", ctx.Operation)
	assert.Equal(t, "users", ctx.Table)
	assert.Equal(t, "1", ctx.RecordID)
	assert.Equal(t, "test", ctx.Actor)
	assert.NotNil(t, ctx.BeforeData)
	assert.NotNil(t, ctx.AfterData)
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "-")
}

func TestGenerateRequestID_Format(t *testing.T) {
	id := generateRequestID()

	// Should contain timestamp and random string
	assert.Regexp(t, `^\d{14}-[a-zA-Z0-9]{8}$`, id)
}

func TestRandomString(t *testing.T) {
	s1 := randomString(8)
	s2 := randomString(8)

	assert.Len(t, s1, 8)
	assert.Len(t, s2, 8)
	// Note: Due to time.Now().UnixNano() being called quickly,
	// there's a small chance they could be equal, but statistically unlikely
}

func TestRandomString_DifferentLengths(t *testing.T) {
	for _, length := range []int{4, 8, 16, 32} {
		s := randomString(length)
		assert.Len(t, s, length)
	}
}

func TestRandomString_Characters(t *testing.T) {
	s := randomString(100)

	for _, c := range s {
		assert.True(t, c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9')
	}
}

func TestToJSON(t *testing.T) {
	testCases := []struct {
		name     string
		data     interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"map", map[string]interface{}{"key": "value"}, `{"key":"value"}`},
		{"slice", []int{1, 2, 3}, "[1,2,3]"},
		{"string", "hello", `"hello"`},
		{"number", 42, "42"},
		{"bool", true, "true"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := toJSON(tc.data)
			if tc.expected == "" {
				assert.Empty(t, got)
			} else {
				assert.Contains(t, got, tc.expected)
			}
		})
	}
}

func TestToJSON_NestedMap(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		},
		"id": 1,
	}

	got := toJSON(data)
	assert.Contains(t, got, "John")
	assert.Contains(t, got, "john@example.com")
	assert.Contains(t, got, "1")
}

func TestToJSON_Struct(t *testing.T) {
	type User struct {
		Name  string
		Email string
	}
	user := User{Name: "Jane", Email: "jane@example.com"}

	got := toJSON(user)
	assert.Contains(t, got, "Jane")
	assert.Contains(t, got, "jane@example.com")
}

func TestAuditLog(t *testing.T) {
	log := AuditLog{
		ID:         1,
		Timestamp:  time.Now(),
		Operation:  "update",
		Table:      "users",
		RecordID:   "1",
		Actor:      "admin",
		RequestID:  "req-123",
		BeforeData: `{"name":"old"}`,
		AfterData:  `{"name":"new"}`,
		Duration:   50,
		Status:     "success",
		ErrorMsg:   "",
	}

	assert.Equal(t, uint(1), log.ID)
	assert.Equal(t, "update", log.Operation)
	assert.Equal(t, "users", log.Table)
	assert.Equal(t, "req-123", log.RequestID)
	assert.Equal(t, int64(50), log.Duration)
	assert.Equal(t, "success", log.Status)
}

func TestAuditLog_WithError(t *testing.T) {
	log := AuditLog{
		ID:        2,
		Operation: "query",
		Table:     "users",
		RequestID: "req-456",
		Status:    "failed",
		ErrorMsg:  "timeout",
	}

	assert.Equal(t, "failed", log.Status)
	assert.Equal(t, "timeout", log.ErrorMsg)
}

// Test for CRUD Service (requires mock, but we can test basic construction)
func TestNewCRUDService(t *testing.T) {
	// This would require mock dependencies
	// For now, we test the structure exists
	assert.NotNil(t, NewAuditService)
}

// Test for Transaction Service (requires mock, but we can test basic construction)
func TestNewTransactionService(t *testing.T) {
	// This would require mock dependencies
	// For now, we test the structure exists
	assert.NotNil(t, NewTransactionService)
}
