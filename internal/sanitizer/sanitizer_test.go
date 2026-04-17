package sanitizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid identifiers
		{"simple table", "users", false},
		{"underscore", "user_orders", false},
		{"mixed case", "UserOrders", false},
		{"with numbers", "users2024", false},
		{"single char", "u", false},
		// Invalid - SQL injection payloads
		{"injection semicolon", "users; DROP TABLE users", true},
		{"injection comment", "users--", true},
		{"injection union", "users UNION SELECT", true},
		{"injection or", "id OR 1=1", true},
		{"injection quote", "users' OR '1'='1", true},
		{"injection backtick", "users`", true},
		{"injection parens", "users()", true},
		{"injection space", "user name", true},
		{"injection dot dot", "users..admin", true},
		{"injection asterisk", "users*", true},
		{"starts with digit", "2024users", true},
		{"starts with underscore ok", "_users", false},
		{"empty string", "", true},
		{"very long name", string(make([]byte, 65)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTableName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid simple column names
		{"simple column", "id", false},
		{"underscore", "user_id", false},
		{"mixed case", "FirstName", false},
		{"with numbers", "col2024", false},
		// Valid qualified names (alias.column)
		{"qualified u.id", "u.id", false},
		{"qualified o.order_id", "o.order_id", false},
		// Invalid - SQL injection payloads
		{"injection semicolon", "id; DROP", true},
		{"injection or", "id OR 1=1", true},
		{"injection space", "user name", true},
		{"injection quote", "'admin'", true},
		{"injection backtick", "col`", true},
		{"injection parens", "func()", true},
		{"empty string", "", true},
		{"only dot", ".", true},
		{"double dot", "a..b", true},
		{"trailing dot", "a.", true},
		{"starts with digit", "1col", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColumnName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple alias", "u", false},
		{"underscore alias", "usr", false},
		{"empty allowed", "", false},
		{"valid short", "o", false},
		{"too long", string(make([]byte, 33)), true},
		{"space", "u ", true},
		{"semicolon", "u;", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlias(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFieldList(t *testing.T) {
	t.Run("valid list", func(t *testing.T) {
		err := ValidateFieldList([]string{"id", "name", "email"})
		assert.NoError(t, err)
	})

	t.Run("empty list", func(t *testing.T) {
		err := ValidateFieldList([]string{})
		assert.NoError(t, err)
	})

	t.Run("invalid field at index", func(t *testing.T) {
		err := ValidateFieldList([]string{"id", "name; DROP", "email"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index 1")
	})
}

func TestValidateOrderDirection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"asc lowercase", "asc", false},
		{"desc lowercase", "desc", false},
		{"ASC uppercase", "ASC", false},
		{"DESC uppercase", "DESC", false},
		{"empty allowed", "", false},
		{"random string", "RANDOM", true},
		{"injection", "asc; DROP", true},
		{"multiple words", "asc desc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrderDirection(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJoinType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"inner", "inner", false},
		{"left", "left", false},
		{"right", "right", false},
		{"INNER uppercase", "INNER", false},
		{"LEFT uppercase", "LEFT", false},
		{"RIGHT uppercase", "RIGHT", false},
		{"empty", "", true},
		{"cross", "cross", true},
		{"full", "full", true},
		{"injection", "left; DROP", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJoinType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "id", "`id`"},
		{"table name", "users", "`users`"},
		{"qualified name", "u.id", "`u`.`id`"},
		{"qualified two dots", "u.user_id", "`u`.`user_id`"},
		{"alias order", "o.id", "`o`.`id`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQuoteFieldList(t *testing.T) {
	t.Run("multiple fields", func(t *testing.T) {
		result := QuoteFieldList([]string{"id", "name", "email"})
		assert.Equal(t, "`id`, `name`, `email`", result)
	})

	t.Run("qualified fields", func(t *testing.T) {
		result := QuoteFieldList([]string{"u.id", "o.name"})
		assert.Equal(t, "`u`.`id`, `o`.`name`", result)
	})

	t.Run("empty returns asterisk", func(t *testing.T) {
		result := QuoteFieldList([]string{})
		assert.Equal(t, "*", result)
	})

	t.Run("single field", func(t *testing.T) {
		result := QuoteFieldList([]string{"id"})
		assert.Equal(t, "`id`", result)
	})
}

func TestSanitizeOrderField(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		direction  string
		wantResult string
		wantErr    bool
	}{
		{"asc simple", "id", "asc", "`id` ASC", false},
		{"desc simple", "created_at", "desc", "`created_at` DESC", false},
		{"qualified field asc", "u.id", "asc", "`u`.`id` ASC", false},
		{"empty direction defaults asc", "name", "", "`name` ASC", false},
		{"uppercase direction", "id", "DESC", "`id` DESC", false},
		{"invalid field", "id; DROP", "asc", "", true},
		{"invalid direction", "id", "RANDOM", "", true},
		{"empty field", "", "asc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeOrderField(tt.field, tt.direction)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func TestValidateTableNames(t *testing.T) {
	t.Run("valid list", func(t *testing.T) {
		err := ValidateTableNames([]string{"users", "orders"})
		assert.NoError(t, err)
	})

	t.Run("invalid at index", func(t *testing.T) {
		err := ValidateTableNames([]string{"users", "id; DROP", "orders"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index 1")
	})
}
