package detector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/detector"
)

func TestNamePatterns(t *testing.T) {
	patterns := []string{
		"deleted_at", "delete_time", "delete_date", "deleted_time",
		"is_del", "is_deleted", "deleted", "delete_flag",
		"del_flag", "del", "status",
	}
	assert.Len(t, patterns, 11)
}

func TestTypeBasedValues(t *testing.T) {
	values := map[string]string{
		"tinyint":   "1",
		"smallint":  "1",
		"int":       "1",
		"bigint":    "1",
		"boolean":   "1",
		"timestamp": "0000-00-00 00:00:00",
		"datetime":  "0000-00-00 00:00:00",
	}

	assert.Equal(t, "1", values["tinyint"])
	assert.Equal(t, "0000-00-00 00:00:00", values["timestamp"])
}

func TestContainsDeleteKeyword(t *testing.T) {
	tests := []struct {
		comment  string
		expected bool
	}{
		{"删除时间", true},
		{"是否删除：0.否，1.是", true},
		{"逻辑删除", true},
		{"软删除", true},
		{"用户名", false},
		{"创建时间", false},
	}

	for _, tt := range tests {
		got := detector.ContainsDeleteKeyword(tt.comment)
		assert.Equal(t, tt.expected, got, "comment: %s", tt.comment)
	}
}

func TestExtractDeleteValue(t *testing.T) {
	tests := []struct {
		comment  string
		dataType string
		expected string
	}{
		{"是否删除：0.否，1.是", "tinyint", "1"},
		{"状态:0正常,1删除", "int", "1"},
		{"删除标记 1=删除 0=未删", "tinyint", "1"},
		{"正常数据", "varchar", ""},
	}

	for _, tt := range tests {
		got := detector.ExtractDeleteValue(tt.comment, tt.dataType)
		assert.Equal(t, tt.expected, got, "comment: %s", tt.comment)
	}
}

func TestNewDetector(t *testing.T) {
	d := detector.NewDetector()
	assert.NotNil(t, d)
}

func TestDetect(t *testing.T) {
	d := detector.NewDetector()

	columns := []detector.ColumnInfo{
		{Name: "id", DataType: "int"},
		{Name: "deleted_time", DataType: "datetime", Comment: "删除时间"},
		{Name: "is_del", DataType: "tinyint", Comment: "是否删除：0.否，1.是"},
	}

	info := d.Detect("users", columns)
	assert.NotNil(t, info)
	assert.Equal(t, "users", info.TableName)
	assert.Len(t, info.Fields, 2)
}
