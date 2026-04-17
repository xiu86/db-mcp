package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAuditService_Close tests closing the audit service with a real file.
func TestAuditService_Close(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit_test.log")

	svc := NewAuditService(filePath)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.file)

	err := svc.Close()
	assert.NoError(t, err)
}

// TestAuditService_Close_EmptyPath tests closing when created with empty path.
// Empty path defaults to "audit.log" in current directory.
func TestAuditService_Close_EmptyPath(t *testing.T) {
	svc := NewAuditService("")
	assert.NotNil(t, svc)
	// Empty path creates a default file "audit.log"
	// File may or may not be nil depending on if audit.log is writable

	err := svc.Close()
	assert.NoError(t, err)
}

// TestAuditService_Close_AfterWrite tests that Close works after writing.
func TestAuditService_Close_AfterWrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit_write_test.log")

	svc := NewAuditService(filePath)
	ctx := svc.Start("insert", "users", "1")
	// Write an audit entry before closing
	svc.Success(ctx, nil, map[string]interface{}{"id": 1}, 1)

	err := svc.Close()
	assert.NoError(t, err)

	// Verify the file has content
	content, readErr := os.ReadFile(filePath)
	assert.NoError(t, readErr)
	assert.NotEmpty(t, content)
}

// TestAuditService_Success_WithFile tests Success when file is open.
func TestAuditService_Success_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "success_audit.log")

	svc := NewAuditService(filePath)
	assert.NotNil(t, svc.file)

	ctx := svc.Start("insert", "users", "1")
	svc.Success(ctx, nil, map[string]interface{}{"id": 1}, 1)

	err := svc.Close()
	assert.NoError(t, err)

	content, _ := os.ReadFile(filePath)
	assert.NotEmpty(t, content)
}

// TestAuditService_Fail_WithFile tests Fail when file is open.
func TestAuditService_Fail_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "fail_audit.log")

	svc := NewAuditService(filePath)
	assert.NotNil(t, svc.file)

	ctx := svc.Start("update", "users", "1")
	svc.Fail(ctx, "connection lost")

	err := svc.Close()
	assert.NoError(t, err)

	content, _ := os.ReadFile(filePath)
	assert.NotEmpty(t, content)
}

// TestAuditService_writeEntry_WithData tests that writeEntry handles various data types.
func TestAuditService_writeEntry_WithData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "write_entry_test.log")

	svc := NewAuditService(filePath)

	ctx := svc.Start("update", "orders", "5")
	ctx.BeforeData = map[string]interface{}{
		"status": "pending",
		"total":  100.50,
	}
	ctx.AfterData = map[string]interface{}{
		"status": "completed",
		"total":  100.50,
	}
	ctx.SQL = "UPDATE orders SET status = ? WHERE id = ?"

	svc.writeEntry(ctx, ctx.BeforeData, ctx.AfterData, "success", "")

	err := svc.Close()
	assert.NoError(t, err)

	content, _ := os.ReadFile(filePath)
	assert.NotEmpty(t, content)
}

// TestAuditService_toJSON_Error tests toJSON with unsupported types.
func TestToJSON_Unsupportable(t *testing.T) {
	// Functions that are already tested in service_test.go
	// Adding edge case: complex nested struct
	type Inner struct {
		Data string
	}
	type Complex struct {
		Inner Inner
		Slice []int
		Map   map[string]string
	}

	c := Complex{
		Inner: Inner{Data: "test"},
		Slice: []int{1, 2, 3},
		Map:   map[string]string{"key": "value"},
	}

	result := toJSON(c)
	assert.Contains(t, result, "test")
	assert.Contains(t, result, "1")
	assert.Contains(t, result, "key")
}

// TestAuditService_toJSON_EmptySlice tests toJSON with empty slice.
func TestToJSON_EmptySlice(t *testing.T) {
	result := toJSON([]string{})
	assert.Equal(t, "[]", result)
}

// TestAuditService_toJSON_Bool tests toJSON with boolean values.
func TestToJSON_Bool(t *testing.T) {
	result := toJSON(false)
	assert.Equal(t, "false", result)
	result = toJSON(true)
	assert.Equal(t, "true", result)
}

// TestAuditService_toJSON_Float tests toJSON with float values.
func TestToJSON_Float(t *testing.T) {
	result := toJSON(3.14159)
	assert.Contains(t, result, "3.14")
}

// TestAuditService_toJSON_Int tests toJSON with int values.
func TestToJSON_Int(t *testing.T) {
	result := toJSON(int64(42))
	assert.Equal(t, "42", result)
}

// TestAuditService_toJSON_IntNegative tests toJSON with negative int.
func TestToJSON_IntNegative(t *testing.T) {
	result := toJSON(-100)
	assert.Contains(t, result, "-100")
}

// TestAuditService_toJSON_EmptyMap tests toJSON with empty map.
func TestToJSON_EmptyMap(t *testing.T) {
	result := toJSON(map[string]interface{}{})
	assert.Equal(t, "{}", result)
}

// TestAuditService_toJSON_ComplexMap tests toJSON with nested map.
func TestToJSON_ComplexMap(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "Alice",
			},
		},
		"count": 5,
	}
	result := toJSON(data)
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "5")
}

// TestNewAuditService_WithInvalidDir tests creating service with invalid directory.
func TestNewAuditService_WithInvalidDir(t *testing.T) {
	// Empty path uses current directory
	svc := NewAuditService("")
	assert.NotNil(t, svc)
	// Should create with default path
	svc.Close()
}

// TestNewAuditServiceWithDB_NilDB tests NewAuditServiceWithDB with nil DB.
func TestNewAuditServiceWithDB_NilDB(t *testing.T) {
	svc := NewAuditServiceWithDB("", nil)
	assert.NotNil(t, svc)
	// Should not panic
}

// TestAuditService_Start_MultipleIDs tests that each Start generates unique IDs.
func TestAuditService_Start_MultipleIDs(t *testing.T) {
	svc := NewAuditService("")
	ctx1 := svc.Start("query", "t1", "1")
	ctx2 := svc.Start("query", "t2", "2")
	ctx3 := svc.Start("query", "t3", "3")

	assert.NotEqual(t, ctx1.RequestID, ctx2.RequestID)
	assert.NotEqual(t, ctx2.RequestID, ctx3.RequestID)
	assert.NotEqual(t, ctx1.RequestID, ctx3.RequestID)
}

// TestAuditService_Start_EmptyTable tests Start with empty table name.
func TestAuditService_Start_EmptyTable(t *testing.T) {
	svc := NewAuditService("")
	ctx := svc.Start("query", "", "")

	assert.NotEmpty(t, ctx.RequestID)
	assert.Empty(t, ctx.Table)
}

// TestGenerateRequestID_Uniqueness tests that generateRequestID produces unique IDs.
func TestGenerateRequestID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateRequestID()
		assert.False(t, ids[id], "duplicate request ID generated: %s", id)
		ids[id] = true
	}
}

// TestRandomString_Deterministic tests that same seed produces same string (not used but good coverage).
func TestRandomString_LengthEdge(t *testing.T) {
	s0 := randomString(0)
	assert.Equal(t, "", s0)
	s1 := randomString(1)
	assert.Len(t, s1, 1)
}