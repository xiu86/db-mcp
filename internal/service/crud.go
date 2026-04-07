package service

import (
    "context"
    "db-mcp/internal/config"
    "db-mcp/internal/detector"
    "db-mcp/internal/repository"
    "db-mcp/pkg/logger"
)

type CRUDService struct {
    repo     *repository.Repository
    audit    *AuditService
    detector *detector.DeleteFieldDetector
    config   *config.Config
    logger   *logger.Logger
}

func NewCRUDService(repo *repository.Repository, audit *AuditService, cfg *config.Config, log *logger.Logger) *CRUDService {
    return &CRUDService{
        repo:     repo,
        audit:    audit,
        detector: detector.NewDetector(),
        config:   cfg,
        logger:   log,
    }
}

func (s *CRUDService) Query(ctx context.Context, table string, fields []string, where map[string]interface{}, order []repository.OrderBy, limit, offset int) (*repository.QueryResult, error) {
    auditCtx := s.audit.Start("query", table, "")
    result, err := s.repo.Query(&repository.QueryRequest{
        Table:  table,
        Fields: fields,
        Where:  where,
        Order:  order,
        Limit:  limit,
        Offset: offset,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, nil, result.Rows, int64(len(result.Rows)))
    return result, nil
}

func (s *CRUDService) Insert(ctx context.Context, table string, data map[string]interface{}) (*repository.MutationResult, error) {
    auditCtx := s.audit.Start("insert", table, "")
    result, err := s.repo.Insert(&repository.InsertRequest{
        Table: table,
        Data:  data,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, nil, data, result.AffectedRows)
    return result, nil
}

func (s *CRUDService) Update(ctx context.Context, table string, data, where map[string]interface{}) (*repository.MutationResult, error) {
    auditCtx := s.audit.Start("update", table, "")

    // Get before data for audit
    beforeResult, _ := s.repo.Query(&repository.QueryRequest{
        Table: table,
        Where: where,
        Limit: 100,
    })

    result, err := s.repo.Update(&repository.UpdateRequest{
        Table: table,
        Data:  data,
        Where: where,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, beforeResult.Rows, data, result.AffectedRows)
    return result, nil
}

func (s *CRUDService) Delete(ctx context.Context, table string, where map[string]interface{}) (*repository.MutationResult, error) {
    auditCtx := s.audit.Start("delete", table, "")

    // Detect delete fields
    columns := s.getTableColumns(table)
    deleteField := s.detector.Detect(table, columns)

    // Get before data for audit
    beforeResult, _ := s.repo.Query(&repository.QueryRequest{
        Table: table,
        Where: where,
        Limit: 100,
    })

    result, err := s.repo.LogicalDelete(&repository.DeleteRequest{
        Table:       table,
        Where:       where,
        DeleteField: deleteField,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, beforeResult.Rows, nil, result.AffectedRows)
    return result, nil
}

func (s *CRUDService) BatchInsert(ctx context.Context, table string, data []map[string]interface{}) (*repository.BatchResult, error) {
    auditCtx := s.audit.Start("batch_insert", table, "")
    result, err := s.repo.BatchInsert(&repository.BatchInsertRequest{
        Table: table,
        Data:  data,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, nil, data, result.SuccessCount)
    return result, nil
}

func (s *CRUDService) BatchUpdate(ctx context.Context, table string, data []map[string]interface{}, keyField string) (*repository.BatchResult, error) {
    auditCtx := s.audit.Start("batch_update", table, "")
    result, err := s.repo.BatchUpdate(&repository.BatchUpdateRequest{
        Table:    table,
        Data:     data,
        KeyField: keyField,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, nil, data, result.SuccessCount)
    return result, nil
}

func (s *CRUDService) BatchDelete(ctx context.Context, table string, ids []string, idField string) (*repository.BatchResult, error) {
    auditCtx := s.audit.Start("batch_delete", table, "")

    columns := s.getTableColumns(table)
    deleteField := s.detector.Detect(table, columns)

    result, err := s.repo.BatchLogicalDelete(&repository.BatchDeleteRequest{
        Table:       table,
        IDs:         ids,
        IDField:     idField,
        DeleteField: deleteField,
    })
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, nil, nil, result.SuccessCount)
    return result, nil
}

func (s *CRUDService) Join(ctx context.Context, req *repository.JoinRequest) (*repository.QueryResult, error) {
    auditCtx := s.audit.Start("join", req.Tables[0].Name, "")
    result, err := s.repo.JoinQuery(req)
    if err != nil {
        s.audit.Fail(auditCtx, err.Error())
        return nil, err
    }
    s.audit.Success(auditCtx, nil, result.Rows, int64(len(result.Rows)))
    return result, nil
}

// GetSchema 获取表结构及检测到的删除字段
func (s *CRUDService) GetSchema(ctx context.Context, table string) (map[string]interface{}, error) {
    schema, err := s.repo.GetTableSchema(table)
    if err != nil {
        return nil, err
    }

    // 检测删除字段
    columns := s.toDetectorColumns(schema.Columns)
    deleteField := s.detector.Detect(table, columns)

    return map[string]interface{}{
        "table_name":    schema.TableName,
        "columns":       schema.Columns,
        "delete_fields": deleteField,
    }, nil
}

func (s *CRUDService) getTableColumns(table string) []detector.ColumnInfo {
    schema, err := s.repo.GetTableSchema(table)
    if err != nil {
        return []detector.ColumnInfo{}
    }
    return s.toDetectorColumns(schema.Columns)
}

func (s *CRUDService) toDetectorColumns(columns []repository.ColumnInfo) []detector.ColumnInfo {
    var result []detector.ColumnInfo
    for _, col := range columns {
        result = append(result, detector.ColumnInfo{
            Name:     col.Name,
            DataType: col.DataType,
            Comment:  col.Comment,
        })
    }
    return result
}
