package mocks

import (
	"database/sql"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// MockDB provides a mock implementation of gorm.DB behavior for testing
type MockDB struct {
	Tables       map[string][]map[string]interface{}
	ShouldError  bool
	ErrorMsg     string
	LastInsertID int64
	RowsAffected int64
	Closed       bool
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		Tables:      make(map[string][]map[string]interface{}),
		ShouldError: false,
	}
}

// AddTable adds a table with initial data
func (m *MockDB) AddTable(name string, data []map[string]interface{}) {
	m.Tables[name] = data
}

// AddRow adds a row to a table
func (m *MockDB) AddRow(table string, row map[string]interface{}) {
	if m.Tables[table] == nil {
		m.Tables[table] = []map[string]interface{}{}
	}
	m.Tables[table] = append(m.Tables[table], row)
}

// Table returns a query builder for the table
func (m *MockDB) Table(name string) *MockQuery {
	return &MockQuery{
		mock:     m,
		table:    name,
		where:    make(map[string]interface{}),
		orders:   []orderClause{},
		limit:    0,
		offset:   0,
		joins:    []joinClause{},
		selected: "*",
	}
}

// Begin starts a transaction
func (m *MockDB) Begin() *MockDB {
	return &MockDB{
		Tables:       m.Tables,
		ShouldError:  m.ShouldError,
		ErrorMsg:     m.ErrorMsg,
		LastInsertID: m.LastInsertID,
		RowsAffected: m.RowsAffected,
	}
}

type orderClause struct {
	field     string
	direction string
}

type joinClause struct {
	joinType string
	table    string
	onClause string
}

// MockQuery represents a mock query builder
type MockQuery struct {
	mock       *MockDB
	table      string
	where      map[string]interface{}
	orders     []orderClause
	limit      int
	offset     int
	joins      []joinClause
	selected   string
	updateData map[string]interface{}
	createData map[string]interface{}
}

// Where adds where conditions
func (q *MockQuery) Where(query interface{}, args ...interface{}) *MockQuery {
	if whereMap, ok := query.(map[string]interface{}); ok {
		for k, v := range whereMap {
			q.where[k] = v
		}
	}
	return q
}

// Order adds order by clause
func (q *MockQuery) Order(value string) *MockQuery {
	// Parse "field direction"
	field := value
	direction := "ASC"
	if len(value) > 4 {
		last4 := value[len(value)-4:]
		if last4 == " DESC" {
			direction = "DESC"
			field = value[:len(value)-5]
		} else if last4 == " ASC" {
			field = value[:len(value)-4]
		}
	}
	q.orders = append(q.orders, orderClause{field: field, direction: direction})
	return q
}

// Limit sets the limit
func (q *MockQuery) Limit(limit int) *MockQuery {
	q.limit = limit
	return q
}

// Offset sets the offset
func (q *MockQuery) Offset(offset int) *MockQuery {
	q.offset = offset
	return q
}

// Select sets the select fields
func (q *MockQuery) Select(fields string) *MockQuery {
	q.selected = fields
	return q
}

// Joins adds a join clause
func (q *MockQuery) Joins(query string) *MockQuery {
	// Simple parsing - in real tests, this would be more sophisticated
	q.joins = append(q.joins, joinClause{joinType: "INNER", onClause: query})
	return q
}

// Find executes the query and scans results
func (q *MockQuery) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	result := &gorm.DB{}

	if q.mock.ShouldError {
		result.Error = errors.New(q.mock.ErrorMsg)
		return result
	}

	// Type assert dest to expected type
	rowsSlice, ok := dest.(*[]map[string]interface{})
	if !ok {
		result.Error = fmt.Errorf("unexpected dest type: %T", dest)
		return result
	}

	tableData := q.mock.Tables[q.table]
	if tableData == nil {
		tableData = []map[string]interface{}{}
	}

	// Apply filters
	var filtered []map[string]interface{}
	for _, row := range tableData {
		if q.matchesWhere(row) {
			filtered = append(filtered, q.copyRow(row))
		}
	}

	// Apply ordering
	filtered = q.applyOrder(filtered)

	// Apply limit/offset
	if q.offset > 0 {
		if q.offset >= len(filtered) {
			filtered = []map[string]interface{}{}
		} else {
			filtered = filtered[q.offset:]
		}
	}
	if q.limit > 0 && q.limit < len(filtered) {
		filtered = filtered[:q.limit]
	}

	*rowsSlice = filtered
	result.RowsAffected = int64(len(filtered))
	return result
}

// Count counts the records
func (q *MockQuery) Count(count *int64) *gorm.DB {
	result := &gorm.DB{}

	if q.mock.ShouldError {
		result.Error = errors.New(q.mock.ErrorMsg)
		return result
	}

	tableData := q.mock.Tables[q.table]
	if tableData == nil {
		*count = 0
		return result
	}

	var filtered int64
	for _, row := range tableData {
		if q.matchesWhere(row) {
			filtered++
		}
	}

	*count = filtered
	return result
}

// Create inserts a record
func (q *MockQuery) Create(value interface{}) *gorm.DB {
	result := &gorm.DB{}

	if q.mock.ShouldError {
		result.Error = errors.New(q.mock.ErrorMsg)
		return result
	}

	data, ok := value.(map[string]interface{})
	if !ok {
		result.Error = fmt.Errorf("unexpected value type: %T", value)
		return result
	}

	if q.mock.Tables[q.table] == nil {
		q.mock.Tables[q.table] = []map[string]interface{}{}
	}

	// Add ID if not present
	if _, exists := data["id"]; !exists {
		data["id"] = len(q.mock.Tables[q.table]) + 1
	}

	q.mock.Tables[q.table] = append(q.mock.Tables[q.table], data)
	result.RowsAffected = 1
	return result
}

// CreateInBatches inserts records in batches
func (q *MockQuery) CreateInBatches(value interface{}, batchSize int) *gorm.DB {
	return q.Create(value)
}

// Updates updates records
func (q *MockQuery) Updates(values interface{}) *gorm.DB {
	result := &gorm.DB{}

	if q.mock.ShouldError {
		result.Error = errors.New(q.mock.ErrorMsg)
		return result
	}

	data, ok := values.(map[string]interface{})
	if !ok {
		result.Error = fmt.Errorf("unexpected values type: %T", values)
		return result
	}

	tableData := q.mock.Tables[q.table]
	var updated int64

	for i, row := range tableData {
		if q.matchesWhere(row) {
			for k, v := range data {
				tableData[i][k] = v
			}
			updated++
		}
	}

	result.RowsAffected = updated
	return result
}

// Delete deletes records
func (q *MockQuery) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	result := &gorm.DB{}

	if q.mock.ShouldError {
		result.Error = errors.New(q.mock.ErrorMsg)
		return result
	}

	return result
}

// Scan scans results
func (q *MockQuery) Scan(dest interface{}) *gorm.DB {
	result := &gorm.DB{}

	if q.mock.ShouldError {
		result.Error = errors.New(q.mock.ErrorMsg)
		return result
	}

	// Simplified scan - just return empty results
	return result
}

// Error returns the error
func (q *MockQuery) Error() error {
	if q.mock.ShouldError {
		return errors.New(q.mock.ErrorMsg)
	}
	return nil
}

// matchesWhere checks if a row matches where conditions
func (q *MockQuery) matchesWhere(row map[string]interface{}) bool {
	if len(q.where) == 0 {
		return true
	}

	for k, v := range q.where {
		if rowValue, exists := row[k]; !exists || rowValue != v {
			return false
		}
	}
	return true
}

// copyRow creates a copy of a row
func (q *MockQuery) copyRow(row map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range row {
		copy[k] = v
	}
	return copy
}

// applyOrder applies ordering to the results
func (q *MockQuery) applyOrder(rows []map[string]interface{}) []map[string]interface{} {
	// Simplified - just return as-is for testing
	// In a full implementation, this would sort based on q.orders
	return rows
}

// ToGormDB converts MockDB to a gorm.DB-like interface
// This is used in tests that expect *gorm.DB
func (m *MockDB) ToGormDB() *gorm.DB {
	// This is a simplified adapter - in production, you'd use a proper mock
	// For now, we'll return nil and tests will use MockDB directly
	return &gorm.DB{
		Statement: &gorm.Statement{},
		Config:    &gorm.Config{},
		Error:     m.getError(),
	}
}

func (m *MockDB) getError() error {
	if m.ShouldError {
		return errors.New(m.ErrorMsg)
	}
	return nil
}

// MockTx implements a mock transaction
type MockTx struct {
	*MockDB
	committed bool
	rolledBack bool
}

// Commit commits the transaction
func (m *MockTx) Commit() error {
	if m.ShouldError {
		return errors.New(m.ErrorMsg)
	}
	m.committed = true
	return nil
}

// Rollback rolls back the transaction
func (m *MockTx) Rollback() error {
	if m.ShouldError {
		return errors.New(m.ErrorMsg)
	}
	m.rolledBack = true
	return nil
}

// MockRows implements sql.Rows interface for mocking query results
type MockRows struct {
	columns []string
	values  [][]interface{}
	pos     int
	closed  bool
}

// NewMockRows creates a new MockRows
func NewMockRows(columns []string, values ...[]interface{}) *MockRows {
	return &MockRows{
		columns: columns,
		values:  values,
		pos:     -1,
		closed:  false,
	}
}

// Columns returns column names
func (m *MockRows) Columns() ([]string, error) {
	return m.columns, nil
}

// Close closes the rows
func (m *MockRows) Close() error {
	m.closed = true
	return nil
}

// Next advances to next row
func (m *MockRows) Next() bool {
	m.pos++
	return m.pos < len(m.values)
}

// Scan scans the current row
func (m *MockRows) Scan(dest ...interface{}) error {
	if m.pos < 0 || m.pos >= len(m.values) {
		return sql.ErrNoRows
	}

	row := m.values[m.pos]
	for i, val := range row {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *interface{}:
				*d = val
			case *string:
				if s, ok := val.(string); ok {
					*d = s
				}
			case *int:
				if i, ok := val.(int); ok {
					*d = i
				}
			case *int64:
				switch v := val.(type) {
				case int:
					*d = int64(v)
				case int64:
					*d = v
				}
			case *bool:
				if b, ok := val.(bool); ok {
					*d = b
				}
			}
		}
	}
	return nil
}

// Err returns any error
func (m *MockRows) Err() error {
	return nil
}
