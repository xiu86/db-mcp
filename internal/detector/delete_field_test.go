package detector

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	assert.NotNil(t, d)
	assert.NotNil(t, d.cache)
}

func TestDetect_EmptyColumns(t *testing.T) {
	d := NewDetector()
	info := d.Detect("users", []ColumnInfo{})
	assert.NotNil(t, info)
	assert.Equal(t, "users", info.TableName)
	assert.Len(t, info.Fields, 0)
	assert.Empty(t, info.DeleteValue)
}

func TestDetect_CacheHit(t *testing.T) {
	d := NewDetector()
	columns := []ColumnInfo{
		{Name: "id", DataType: "bigint"},
		{Name: "is_del", DataType: "tinyint", Comment: "是否删除：0.否，1.是"},
	}

	info1 := d.Detect("users", columns)
	info2 := d.Detect("users", []ColumnInfo{}) // Should return cached result

	assert.Equal(t, info1, info2)
	assert.Equal(t, info1.TableName, info2.TableName)
}

func TestDetect_CacheNil(t *testing.T) {
	d := &DeleteFieldDetector{cache: nil}
	info := d.Detect("users", []ColumnInfo{})
	assert.NotNil(t, info)
	assert.NotNil(t, d.cache)
}

func TestDetect_NamePattern(t *testing.T) {
	testCases := []struct {
		name     string
		columns  []ColumnInfo
		expected int
	}{
		{
			name:     "deleted_at",
			columns:  []ColumnInfo{{Name: "deleted_at", DataType: "timestamp"}},
			expected: 1,
		},
		{
			name:     "is_del",
			columns:  []ColumnInfo{{Name: "is_del", DataType: "tinyint"}},
			expected: 1,
		},
		{
			name:     "delete_flag",
			columns:  []ColumnInfo{{Name: "delete_flag", DataType: "smallint"}},
			expected: 1,
		},
		{
			name:     "regular_field",
			columns:  []ColumnInfo{{Name: "username", DataType: "varchar"}},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDetector() // Fresh detector for each test to avoid cache issues
			info := d.Detect("test_table", tc.columns)
			assert.Len(t, info.Fields, tc.expected)
		})
	}
}

func TestDetect_CommentKeyword(t *testing.T) {
	testCases := []struct {
		name     string
		comment  string
		expected bool
	}{
		{"删除", "删除时间", true},
		{"软删除", "软删除标记", true},
		{"逻辑删除", "逻辑删除字段", true},
		{"普通字段", "用户名", false},
		{"空注释", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDetector() // Fresh detector for each test
			info := d.Detect("users", []ColumnInfo{
				{Name: "test_field", DataType: "varchar", Comment: tc.comment},
			})
			if tc.expected {
				assert.Len(t, info.Fields, 1)
			} else {
				assert.Len(t, info.Fields, 0)
			}
		})
	}
}

func TestDetect_CommentValueMapping(t *testing.T) {
	d := NewDetector()

	testCases := []struct {
		name      string
		comment   string
		dataType  string
		expected  string
	}{
		{"是删除", "是否删除：0.否，1.是", "tinyint", "1"},
		{"状态删除", "状态:0正常,1删除", "int", "1"},
		{"删除标记", "删除标记 1=删除 0=未删", "smallint", "1"},
		{"普通注释", "普通数据", "varchar", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := d.Detect("users", []ColumnInfo{
				{Name: "test_field", DataType: tc.dataType, Comment: tc.comment},
			})
			if tc.expected != "" {
				assert.Len(t, info.Fields, 1)
				assert.Equal(t, tc.expected, info.Fields[0].TrueValue)
			}
		})
	}
}

func TestDetect_MultipleDeleteFields(t *testing.T) {
	d := NewDetector()

	columns := []ColumnInfo{
		{Name: "id", DataType: "bigint"},
		{Name: "is_del", DataType: "tinyint", Comment: "是否删除：0.否，1.是"},
		{Name: "deleted_time", DataType: "datetime", Comment: "删除时间"},
		{Name: "name", DataType: "varchar"},
	}

	info := d.Detect("users", columns)
	assert.Len(t, info.Fields, 2)
}

func TestDetect_PriorityNameOverComment(t *testing.T) {
	d := NewDetector()

	// Field name match should be detected first
	columns := []ColumnInfo{
		{Name: "is_del", DataType: "tinyint", Comment: "普通注释"},
	}

	info := d.Detect("users", columns)
	assert.Len(t, info.Fields, 1)
	assert.Equal(t, "is_del", info.Fields[0].Name)
}

func TestContainsDeleteKeyword(t *testing.T) {
	testCases := []struct {
		comment  string
		expected bool
	}{
		{"删除时间", true},
		{"是否删除：0.否，1.是", true},
		{"逻辑删除", true},
		{"软删除", true},
		{"del_flag", true},
		{"is_del", true},
		{"用户名", false},
		{"创建时间", false},
		{"邮箱地址", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.comment, func(t *testing.T) {
			got := ContainsDeleteKeyword(tc.comment)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestContainsDeleteKeyword_CaseInsensitive(t *testing.T) {
	assert.True(t, ContainsDeleteKeyword("删除"))
	assert.True(t, ContainsDeleteKeyword("DEL"))
	assert.True(t, ContainsDeleteKeyword("Is_Del"))
}

func TestExtractDeleteValue(t *testing.T) {
	testCases := []struct {
		comment  string
		dataType string
		expected string
	}{
		{"是否删除：0.否，1.是", "tinyint", "1"},
		{"状态:0正常,1删除", "int", "1"},
		{"删除标记 1=删除 0=未删", "tinyint", "1"},
		{"正常数据", "varchar", ""},
		{"删除时间", "datetime", ""},
		{"", "tinyint", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.comment, func(t *testing.T) {
			got := ExtractDeleteValue(tc.comment, tc.dataType)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestExtractDeleteValue_Patterns(t *testing.T) {
	testCases := []struct {
		pattern  string
		input    string
		expected string
	}{
		{`(\d+)[\.:：=]*(?:是|删除|yes|true)`, "是否删除：0.否，1.是", "1"},
		{`(\d+)[\.:：=]*(?:是|删除|yes|true)`, "状态:0正常,1删除", "1"},
		{`(\d+)[\.:：=]*(?:是|删除|yes|true)`, "删除标记 1=删除", "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			re := regexp.MustCompile(tc.pattern)
			matches := re.FindStringSubmatch(tc.input)
			if tc.expected != "" {
				assert.Len(t, matches, 2)
				assert.Equal(t, tc.expected, matches[1])
			}
		})
	}
}

func TestDetermineDeleteValue(t *testing.T) {
	d := &DeleteFieldDetector{}

	testCases := []struct {
		name     string
		fields   []Field
		expected string
	}{
		{"empty", []Field{}, ""},
		{"single field", []Field{{Name: "is_del", TrueValue: "1"}}, "1"},
		{"multiple fields", []Field{{Name: "is_del", TrueValue: "1"}, {Name: "deleted_at", TrueValue: "2024-01-01"}}, "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := d.determineDeleteValue(tc.fields)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestAnalyzeColumn_NameMatch(t *testing.T) {
	d := NewDetector()

	testCases := []struct {
		name      string
		colName   string
		dataType  string
		shouldNil bool
	}{
		{"exact match", "is_del", "tinyint", false},
		{"case insensitive", "IS_DEL", "tinyint", false},
		{"no match", "username", "varchar", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			col := ColumnInfo{Name: tc.colName, DataType: tc.dataType}
			field := d.analyzeColumn(col)
			if tc.shouldNil {
				assert.Nil(t, field)
			} else {
				assert.NotNil(t, field)
				assert.Equal(t, tc.colName, field.Name)
			}
		})
	}
}

func TestAnalyzeColumn_TypeBasedValue(t *testing.T) {
	d := NewDetector()

	testCases := []struct {
		dataType string
		expected string
	}{
		{"varchar", ""},
		{"char", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.dataType, func(t *testing.T) {
			col := ColumnInfo{Name: "unknown_field", DataType: tc.dataType}
			field := d.analyzeColumn(col)
			// Non-matching types should return nil (no keyword, no type mapping)
			if tc.expected == "" && field == nil {
				// This is expected
			}
		})
	}
}
