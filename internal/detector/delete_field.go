package detector

import (
	"regexp"
	"strings"
	"time"
)

type DeleteFieldDetector struct {
	db    interface{}
	cache map[string]*DeleteFieldInfo
}

type DeleteFieldInfo struct {
	TableName   string
	Fields      []Field
	DeleteValue string
}

type Field struct {
	Name      string
	Type      string
	TrueValue string
}

var namePatterns = []string{
	"deleted_at", "delete_time", "delete_date", "deleted_time",
	"is_del", "is_deleted", "deleted", "delete_flag",
	"del_flag", "del", "status",
}

var commentKeywords = []string{
	"删除", "del", "is_del", "逻辑删除", "软删除",
}

var typeBasedValues = map[string]string{
	"tinyint":   "1",
	"smallint":  "1",
	"int":       "1",
	"bigint":    "1",
	"boolean":   "1",
	"timestamp": "__CURRENT_TIMESTAMP__",
	"datetime":  "__CURRENT_TIMESTAMP__",
}

// CurrentTimestampMarker is the sentinel value used to indicate that a datetime/timestamp
// field should be set to the current time rather than a static value.
const CurrentTimestampMarker = "__CURRENT_TIMESTAMP__"

// IsCurrentTimestampMarker returns true if the value indicates "use current time".
func IsCurrentTimestampMarker(value string) bool {
	return value == CurrentTimestampMarker
}

// GetCurrentTimestamp returns the current timestamp in a format suitable for database insertion.
func GetCurrentTimestamp() interface{} {
	return time.Now().Format("2006-01-02 15:04:05")
}

type ColumnInfo struct {
	Name       string
	DataType   string
	ColumnType string
	Nullable   string
	Key        string
	Comment    string
}

func NewDetector() *DeleteFieldDetector {
	return &DeleteFieldDetector{
		cache: make(map[string]*DeleteFieldInfo),
	}
}

func (d *DeleteFieldDetector) Detect(table string, columns []ColumnInfo) *DeleteFieldInfo {
	if d.cache == nil {
		d.cache = make(map[string]*DeleteFieldInfo)
	}

	if info, ok := d.cache[table]; ok {
		return info
	}

	detectedFields := d.detectDeleteFields(columns)

	info := &DeleteFieldInfo{
		TableName:   table,
		Fields:      detectedFields,
		DeleteValue: d.determineDeleteValue(detectedFields),
	}

	d.cache[table] = info
	return info
}

func (d *DeleteFieldDetector) detectDeleteFields(columns []ColumnInfo) []Field {
	var result []Field

	for _, col := range columns {
		field := d.analyzeColumn(col)
		if field != nil {
			result = append(result, *field)
			if len(result) >= 2 {
				break
			}
		}
	}

	return result
}

func (d *DeleteFieldDetector) analyzeColumn(col ColumnInfo) *Field {
	// 1. 字段名匹配检测
	for _, pattern := range namePatterns {
		if strings.EqualFold(col.Name, pattern) {
			return &Field{
				Name:      col.Name,
				Type:      col.DataType,
				TrueValue: typeBasedValues[col.DataType],
			}
		}
	}

	// 2. COMMENT 语义检测
	if ContainsDeleteKeyword(col.Comment) {
		trueValue := typeBasedValues[col.DataType]
		return &Field{
			Name:      col.Name,
			Type:      col.DataType,
			TrueValue: trueValue,
		}
	}

	// 3. COMMENT 值映射检测
	if mappedValue := ExtractDeleteValue(col.Comment, col.DataType); mappedValue != "" {
		return &Field{
			Name:      col.Name,
			Type:      col.DataType,
			TrueValue: mappedValue,
		}
	}

	return nil
}

// ContainsDeleteKeyword checks if comment contains delete-related keywords
func ContainsDeleteKeyword(comment string) bool {
	lowerComment := strings.ToLower(comment)
	for _, keyword := range commentKeywords {
		if strings.Contains(lowerComment, keyword) {
			return true
		}
	}
	return false
}

// ExtractDeleteValue extracts delete value from comment pattern
func ExtractDeleteValue(comment, dataType string) string {
	patterns := []string{
		`(\d+)[\.:：=]*(?:是|删除|yes|true)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(strings.ToLower(comment))
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func (d *DeleteFieldDetector) determineDeleteValue(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}
	return fields[0].TrueValue
}
