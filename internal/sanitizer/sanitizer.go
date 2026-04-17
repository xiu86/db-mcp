package sanitizer

import (
	"db-mcp/internal/errors"
	"fmt"
	"regexp"
	"strings"
)

// identifierRegex allows: letters, digits, underscores.
// Must start with a letter or underscore. No spaces, semicolons, quotes, parentheses.
var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// qualifiedIdentifierRegex allows: alias.column pattern like u.id
var qualifiedIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*$`)

const (
	maxTableNameLength = 64
	maxIdentifierLength = 128
	maxAliasLength = 32
)

// ValidateTableName checks that a table name is a valid SQL identifier.
func ValidateTableName(name string) error {
	if name == "" {
		return errors.NewError(errors.ErrInvalidInput, "table name cannot be empty", nil)
	}
	if len(name) > maxTableNameLength {
		return errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("table name exceeds maximum length of %d: %s", maxTableNameLength, name), nil)
	}
	if !identifierRegex.MatchString(name) {
		return errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("invalid table name (contains illegal characters): %s", name), nil)
	}
	return nil
}

// ValidateColumnName checks that a column name (possibly with alias prefix like "u.id") is valid.
func ValidateColumnName(name string) error {
	if name == "" {
		return errors.NewError(errors.ErrInvalidInput, "column name cannot be empty", nil)
	}
	if len(name) > maxIdentifierLength {
		return errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("column name exceeds maximum length of %d: %s", maxIdentifierLength, name), nil)
	}
	if !identifierRegex.MatchString(name) && !qualifiedIdentifierRegex.MatchString(name) {
		return errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("invalid column name (contains illegal characters): %s", name), nil)
	}
	return nil
}

// ValidateAlias checks that an alias is a valid short identifier.
// Empty alias is allowed.
func ValidateAlias(alias string) error {
	if alias == "" {
		return nil
	}
	if len(alias) > maxAliasLength {
		return errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("alias exceeds maximum length of %d: %s", maxAliasLength, alias), nil)
	}
	if !identifierRegex.MatchString(alias) {
		return errors.NewError(errors.ErrInvalidInput,
			fmt.Sprintf("invalid alias (contains illegal characters): %s", alias), nil)
	}
	return nil
}

// ValidateFieldName validates a field name used in ORDER BY, WHERE key, etc.
// Alias for ValidateColumnName for semantic clarity.
func ValidateFieldName(name string) error {
	return ValidateColumnName(name)
}

// ValidateFieldList validates a slice of field names for SELECT clauses.
func ValidateFieldList(fields []string) error {
	if len(fields) == 0 {
		return nil
	}
	for i, f := range fields {
		if err := ValidateColumnName(f); err != nil {
			return errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid field name at index %d: %s", i, f), err)
		}
	}
	return nil
}

// ValidateOrderDirection ensures direction is exactly ASC or DESC.
func ValidateOrderDirection(dir string) error {
	lower := strings.ToLower(dir)
	if lower == "" || lower == "asc" || lower == "desc" {
		return nil
	}
	return errors.NewError(errors.ErrInvalidInput,
		fmt.Sprintf("invalid order direction (must be 'asc' or 'desc'): %s", dir), nil)
}

// ValidateJoinType ensures join type is one of: "inner", "left", "right".
func ValidateJoinType(joinType string) error {
	lower := strings.ToLower(joinType)
	if lower == "inner" || lower == "left" || lower == "right" {
		return nil
	}
	return errors.NewError(errors.ErrInvalidInput,
		fmt.Sprintf("invalid join type (must be 'inner', 'left', or 'right'): %s", joinType), nil)
}

// QuoteIdentifier wraps a SQL identifier in MySQL backticks for safe interpolation.
// It splits on "." to handle "alias.column" patterns, quoting each segment.
func QuoteIdentifier(name string) string {
	parts := strings.Split(name, ".")
	var quoted []string
	for _, part := range parts {
		quoted = append(quoted, "`"+part+"`")
	}
	return strings.Join(quoted, ".")
}

// QuoteFieldList validates and backtick-quotes a list of field names, then joins with commas.
func QuoteFieldList(fields []string) string {
	if len(fields) == 0 {
		return "*"
	}
	var quoted []string
	for _, f := range fields {
		quoted = append(quoted, QuoteIdentifier(f))
	}
	return strings.Join(quoted, ", ")
}

// SanitizeOrderField takes a validated field name and direction, returns a safe ORDER BY string.
func SanitizeOrderField(field string, direction string) (string, error) {
	if err := ValidateFieldName(field); err != nil {
		return "", err
	}
	if err := ValidateOrderDirection(direction); err != nil {
		return "", err
	}

	dir := "ASC"
	if strings.ToLower(direction) == "desc" {
		dir = "DESC"
	}
	return QuoteIdentifier(field) + " " + dir, nil
}

// ValidateTableNames validates a slice of table names.
func ValidateTableNames(names []string) error {
	for i, name := range names {
		if err := ValidateTableName(name); err != nil {
			return errors.NewError(errors.ErrInvalidInput,
				fmt.Sprintf("invalid table name at index %d: %s", i, name), err)
		}
	}
	return nil
}
