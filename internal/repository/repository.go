package repository

import (
    "db-mcp/internal/detector"
    "db-mcp/internal/errors"

    "gorm.io/gorm"
)

type Repository struct {
    db *gorm.DB
}

type QueryRequest struct {
    Table  string
    Fields []string
    Where  map[string]interface{}
    Order  []OrderBy
    Limit  int
    Offset int
}

type OrderBy struct {
    Field     string
    Direction string
}

type InsertRequest struct {
    Table string
    Data  map[string]interface{}
}

type UpdateRequest struct {
    Table string
    Data  map[string]interface{}
    Where map[string]interface{}
}

type DeleteRequest struct {
    Table       string
    Where       map[string]interface{}
    DeleteField *detector.DeleteFieldInfo
}

type BatchInsertRequest struct {
    Table string
    Data  []map[string]interface{}
}

type BatchUpdateRequest struct {
    Table    string
    Data     []map[string]interface{}
    KeyField string
}

type BatchDeleteRequest struct {
    Table       string
    IDs         []string
    IDField     string
    DeleteField *detector.DeleteFieldInfo
}

type JoinRequest struct {
    Tables []TableRef
    Joins  []JoinClause
    Fields []string
    Where  map[string]interface{}
    Order  []OrderBy
    Limit  int
}

type TableRef struct {
    Name  string
    Alias string
}

type JoinClause struct {
    Type      string
    FromTable string
    FromField string
    ToTable   string
    ToField   string
}

type QueryResult struct {
    Rows    []map[string]interface{}
    Total   int64
    Message string
}

type MutationResult struct {
    AffectedRows int64
    LastInsertID int64
    Message      string
}

type BatchResult struct {
    SuccessCount int64
    FailedCount  int64
    Errors       []BatchError
}

type BatchError struct {
    Index   int
    Message string
}

// TableSchema 用于描述表结构
type TableSchema struct {
    TableName string
    Columns   []ColumnInfo
}

// ColumnInfo 用于描述列信息
type ColumnInfo struct {
    Name         string
    DataType     string
    IsNullable   string
    ColumnKey    string
    Extra        string
    ColumnDefault *string
    Comment      string
}

func New(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) Query(req *QueryRequest) (*QueryResult, error) {
    var rows []map[string]interface{}

    query := r.db.Table(req.Table)

    if len(req.Where) > 0 {
        query = query.Where(req.Where)
    }

    for _, order := range req.Order {
        dir := "ASC"
        if order.Direction == "desc" {
            dir = "DESC"
        }
        query = query.Order(order.Field + " " + dir)
    }

    if req.Limit > 0 {
        query = query.Limit(req.Limit)
    }
    if req.Offset > 0 {
        query = query.Offset(req.Offset)
    }

    fields := "*"
    if len(req.Fields) > 0 {
        fields = joinFields(req.Fields)
    }

    err := query.Select(fields).Find(&rows).Error
    if err != nil {
        return nil, errors.WrapGormError(err)
    }

    var total int64
    r.db.Table(req.Table).Where(req.Where).Count(&total)

    return &QueryResult{Rows: rows, Total: total}, nil
}

func (r *Repository) Insert(req *InsertRequest) (*MutationResult, error) {
    err := r.db.Table(req.Table).Create(req.Data).Error
    if err != nil {
        return nil, errors.WrapGormError(err)
    }

    return &MutationResult{
        AffectedRows: 1,
        LastInsertID: 1,
        Message:      "Insert successful",
    }, nil
}

func (r *Repository) Update(req *UpdateRequest) (*MutationResult, error) {
    result := r.db.Table(req.Table).Where(req.Where).Updates(req.Data)
    if result.Error != nil {
        return nil, errors.WrapGormError(result.Error)
    }

    return &MutationResult{
        AffectedRows: result.RowsAffected,
        Message:      "Update successful",
    }, nil
}

func (r *Repository) LogicalDelete(req *DeleteRequest) (*MutationResult, error) {
    if req.DeleteField == nil || len(req.DeleteField.Fields) == 0 {
        return nil, errors.NewError(errors.ErrInvalidInput, "no delete field detected", nil)
    }

    updates := make(map[string]interface{})
    for _, field := range req.DeleteField.Fields {
        updates[field.Name] = field.TrueValue
    }

    result := r.db.Table(req.Table).Where(req.Where).Updates(updates)
    if result.Error != nil {
        return nil, errors.WrapGormError(result.Error)
    }

    return &MutationResult{
        AffectedRows: result.RowsAffected,
        Message:      "Logical delete successful",
    }, nil
}

func (r *Repository) BatchInsert(req *BatchInsertRequest) (*BatchResult, error) {
    var successCount, failedCount int64
    var batchErrors []BatchError

    for i, data := range req.Data {
        err := r.db.Table(req.Table).Create(data).Error
        if err != nil {
            failedCount++
            batchErrors = append(batchErrors, BatchError{Index: i, Message: err.Error()})
        } else {
            successCount++
        }
    }

    return &BatchResult{
        SuccessCount: successCount,
        FailedCount:  failedCount,
        Errors:       batchErrors,
    }, nil
}

func (r *Repository) BatchUpdate(req *BatchUpdateRequest) (*BatchResult, error) {
    var successCount, failedCount int64
    var batchErrors []BatchError

    keyField := req.KeyField
    if keyField == "" {
        keyField = "id"
    }

    for i, data := range req.Data {
        keyValue := data[keyField]
        if keyValue == nil {
            failedCount++
            batchErrors = append(batchErrors, BatchError{Index: i, Message: "key field value is nil"})
            continue
        }

        delete(data, keyField)
        result := r.db.Table(req.Table).Where(keyField+" = ?", keyValue).Updates(data)
        if result.Error != nil {
            failedCount++
            batchErrors = append(batchErrors, BatchError{Index: i, Message: result.Error.Error()})
        } else {
            successCount++
        }
    }

    return &BatchResult{
        SuccessCount: successCount,
        FailedCount:  failedCount,
        Errors:       batchErrors,
    }, nil
}

func (r *Repository) BatchLogicalDelete(req *BatchDeleteRequest) (*BatchResult, error) {
    if req.DeleteField == nil || len(req.DeleteField.Fields) == 0 {
        return nil, errors.NewError(errors.ErrInvalidInput, "no delete field detected", nil)
    }

    var successCount, failedCount int64
    var batchErrors []BatchError

    idField := req.IDField
    if idField == "" {
        idField = "id"
    }

    updates := make(map[string]interface{})
    for _, field := range req.DeleteField.Fields {
        updates[field.Name] = field.TrueValue
    }

    for i, id := range req.IDs {
        result := r.db.Table(req.Table).Where(idField+" = ?", id).Updates(updates)
        if result.Error != nil {
            failedCount++
            batchErrors = append(batchErrors, BatchError{Index: i, Message: result.Error.Error()})
        } else {
            successCount++
        }
    }

    return &BatchResult{
        SuccessCount: successCount,
        FailedCount:  failedCount,
        Errors:       batchErrors,
    }, nil
}

func (r *Repository) JoinQuery(req *JoinRequest) (*QueryResult, error) {
    if len(req.Tables) < 2 {
        return nil, errors.NewError(errors.ErrInvalidInput, "at least 2 tables required for join", nil)
    }

    table0 := req.Tables[0]
    query := r.db.Table(table0.Name + " AS " + table0.Alias)

    for _, join := range req.Joins {
        joinType := "INNER JOIN"
        switch join.Type {
        case "left":
            joinType = "LEFT JOIN"
        case "right":
            joinType = "RIGHT JOIN"
        }
        query = query.Joins(joinType + " " + join.ToTable + " ON " +
            join.FromTable+"."+join.FromField+" = "+
            join.ToTable+"."+join.ToField)
    }

    if len(req.Where) > 0 {
        query = query.Where(req.Where)
    }

    for _, order := range req.Order {
        dir := "ASC"
        if order.Direction == "desc" {
            dir = "DESC"
        }
        query = query.Order(order.Field + " " + dir)
    }

    if req.Limit > 0 {
        query = query.Limit(req.Limit)
    }

    fields := "*"
    if len(req.Fields) > 0 {
        fields = joinFields(req.Fields)
    }

    var rows []map[string]interface{}
    err := query.Select(fields).Find(&rows).Error
    if err != nil {
        return nil, errors.WrapGormError(err)
    }

    return &QueryResult{Rows: rows, Total: int64(len(rows))}, nil
}

func joinFields(fields []string) string {
    result := ""
    for i, f := range fields {
        if i > 0 {
            result += ", "
        }
        result += f
    }
    return result
}

// GetTableSchema 获取表结构信息
func (r *Repository) GetTableSchema(tableName string) (*TableSchema, error) {
	var results []struct {
		Field          string `gorm:"column:COLUMN_NAME"`
		Type           string `gorm:"column:DATA_TYPE"`
		Null           string `gorm:"column:IS_NULLABLE"`
		Key            string `gorm:"column:COLUMN_KEY"`
		Default        *string `gorm:"column:COLUMN_DEFAULT"`
		Extra          string `gorm:"column:EXTRA"`
		Comment        string `gorm:"column:COLUMN_COMMENT"`
	}

	err := r.db.Table("information_schema.COLUMNS").
		Select("COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA, COLUMN_DEFAULT, COLUMN_COMMENT").
		Where("TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", tableName).
		Scan(&results).Error

	if err != nil {
		return nil, errors.WrapGormError(err)
	}

	var columns []ColumnInfo
	for _, col := range results {
		columns = append(columns, ColumnInfo{
			Name:          col.Field,
			DataType:      col.Type,
			IsNullable:    col.Null,
			ColumnKey:     col.Key,
			Extra:         col.Extra,
			ColumnDefault: col.Default,
			Comment:       col.Comment,
		})
	}

	return &TableSchema{
		TableName: tableName,
		Columns:   columns,
	}, nil
}
