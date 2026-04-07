package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"db-mcp/internal/config"
	"db-mcp/internal/repository"
	"db-mcp/internal/service"
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"
)

type MCPServer struct {
	server    *server.MCPServer
	crud      *service.CRUDService
	txService *service.TransactionService
	config    *config.Config
	logger    *logger.Logger
}

func NewMCPServer(db *gorm.DB, cfg *config.Config, log *logger.Logger) *MCPServer {
	s := server.NewMCPServer(
		"db-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	repo := repository.New(db)
	auditSvc := service.NewAuditService(nil, cfg.Database.Database+"_audit")
	crudSvc := service.NewCRUDService(repo, auditSvc, cfg, log)
	txSvc := service.NewTransactionService(db, auditSvc, cfg, log)

	mcpServer := &MCPServer{
		server:    s,
		crud:      crudSvc,
		txService: txSvc,
		config:    cfg,
		logger:    log,
	}

	mcpServer.registerTools()
	return mcpServer
}

func (s *MCPServer) GetServer() *server.MCPServer {
	return s.server
}

func (s *MCPServer) registerTools() {
	// db_query tool
	s.server.AddTool(mcp.NewTool("db_query",
		mcp.WithDescription("Query data from database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name to query")),
		mcp.WithArray("fields", mcp.Description("Fields to select (default: all)")),
		mcp.WithObject("where", mcp.Description("Where conditions")),
		mcp.WithArray("order", mcp.Description("Order by clauses")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of rows")),
		mcp.WithNumber("offset", mcp.Description("Offset for pagination")),
	), s.handleQuery)

	// db_insert tool
	s.server.AddTool(mcp.NewTool("db_insert",
		mcp.WithDescription("Insert data into database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithObject("data", mcp.Required(), mcp.Description("Data to insert")),
	), s.handleInsert)

	// db_update tool
	s.server.AddTool(mcp.NewTool("db_update",
		mcp.WithDescription("Update data in database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithObject("data", mcp.Required(), mcp.Description("Data to update")),
		mcp.WithObject("where", mcp.Required(), mcp.Description("Where conditions")),
	), s.handleUpdate)

	// db_delete tool (logical delete only)
	s.server.AddTool(mcp.NewTool("db_delete",
		mcp.WithDescription("Logical delete from database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithObject("where", mcp.Required(), mcp.Description("Where conditions")),
	), s.handleDelete)

	// db_batch_insert tool
	s.server.AddTool(mcp.NewTool("db_batch_insert",
		mcp.WithDescription("Batch insert data into database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("Array of data to insert")),
	), s.handleBatchInsert)

	// db_batch_update tool
	s.server.AddTool(mcp.NewTool("db_batch_update",
		mcp.WithDescription("Batch update data in database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("Array of data to update")),
		mcp.WithString("key_field", mcp.Description("Key field for matching (default: id)")),
	), s.handleBatchUpdate)

	// db_batch_delete tool
	s.server.AddTool(mcp.NewTool("db_batch_delete",
		mcp.WithDescription("Batch logical delete from database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithArray("ids", mcp.Required(), mcp.Description("Array of IDs to delete")),
		mcp.WithString("id_field", mcp.Description("ID field name (default: id)")),
	), s.handleBatchDelete)

	// db_join tool
	s.server.AddTool(mcp.NewTool("db_join",
		mcp.WithDescription("Perform JOIN query across multiple tables"),
		mcp.WithArray("tables", mcp.Required(), mcp.Description("Tables with aliases")),
		mcp.WithArray("joins", mcp.Required(), mcp.Description("Join clauses")),
		mcp.WithArray("fields", mcp.Description("Fields to select")),
		mcp.WithObject("where", mcp.Description("Where conditions")),
		mcp.WithNumber("limit", mcp.Description("Maximum rows")),
	), s.handleJoin)

	// db_transaction tool
	s.server.AddTool(mcp.NewTool("db_transaction",
		mcp.WithDescription("Execute operations in a transaction"),
		mcp.WithArray("operations", mcp.Required(), mcp.Description("Array of operations to execute")),
	), s.handleTransaction)
}

func getArgs(request mcp.CallToolRequest) map[string]interface{} {
	args := request.GetArguments()
	if args == nil {
		return make(map[string]interface{})
	}
	// Convert map[string]any to map[string]interface{}
	result := make(map[string]interface{})
	for k, v := range args {
		result[k] = v
	}
	return result
}

func (s *MCPServer) handleQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	fields := toStringSlice(args["fields"])
	where, _ := args["where"].(map[string]interface{})
	order := toOrderBySlice(args["order"])
	limit, _ := args["limit"].(float64)
	offset, _ := args["offset"].(float64)

	result, err := s.crud.Query(ctx, table, fields, where, order, int(limit), int(offset))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleInsert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	data, _ := args["data"].(map[string]interface{})

	result, err := s.crud.Insert(ctx, table, data)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	data, _ := args["data"].(map[string]interface{})
	where, _ := args["where"].(map[string]interface{})

	result, err := s.crud.Update(ctx, table, data, where)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	where, _ := args["where"].(map[string]interface{})

	result, err := s.crud.Delete(ctx, table, where)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleBatchInsert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	dataArray := toMapSlice(args["data"])

	result, err := s.crud.BatchInsert(ctx, table, dataArray)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleBatchUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	dataArray := toMapSlice(args["data"])
	keyField, _ := args["key_field"].(string)
	if keyField == "" {
		keyField = "id"
	}

	result, err := s.crud.BatchUpdate(ctx, table, dataArray, keyField)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleBatchDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	table, _ := args["table"].(string)
	ids := toStringSlice(args["ids"])
	idField, _ := args["id_field"].(string)
	if idField == "" {
		idField = "id"
	}

	result, err := s.crud.BatchDelete(ctx, table, ids, idField)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleJoin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	joinReq := &repository.JoinRequest{
		Tables: toTableRefs(args["tables"]),
		Joins:  toJoinClauses(args["joins"]),
		Fields: toStringSlice(args["fields"]),
		Where:  toMap(args["where"]),
	}
	if limit, ok := args["limit"].(float64); ok {
		joinReq.Limit = int(limit)
	}

	result, err := s.crud.Join(ctx, joinReq)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	operations := toOperations(args["operations"])

	txCtx, err := s.txService.Begin(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var results []interface{}
	for _, op := range operations {
		var res interface{}
		var opErr error

		switch op.Type {
		case "insert":
			opErr = txCtx.Insert(op.Table, op.Data)
			res = map[string]string{"status": "inserted"}
		case "update":
			opErr = txCtx.Update(op.Table, op.Data, op.Where)
			res = map[string]string{"status": "updated"}
		case "delete":
			opErr = txCtx.Delete(op.Table, op.Where)
			res = map[string]string{"status": "deleted"}
		case "query":
			res, opErr = txCtx.Query(op.Table, op.Fields, op.Where)
		}

		if opErr != nil {
			txCtx.Rollback()
			return mcp.NewToolResultError(fmt.Sprintf("Transaction failed at operation %s: %v", op.Type, opErr)), nil
		}
		results = append(results, res)
	}

	if err := txCtx.Commit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(map[string]interface{}{
		"status":  "committed",
		"results": results,
	})), nil
}

// Helper types and functions

type Operation struct {
	Type   string
	Table  string
	Data   map[string]interface{}
	Where  map[string]interface{}
	Fields []string
}

func toOperations(v interface{}) []Operation {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var ops []Operation
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			op := Operation{
				Type:   toString(m["type"]),
				Table:  toString(m["table"]),
				Data:   toMap(m["data"]),
				Where:  toMap(m["where"]),
				Fields: toStringSlice(m["fields"]),
			}
			ops = append(ops, op)
		}
	}
	return ops
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func toMapSlice(v interface{}) []map[string]interface{} {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []map[string]interface{}
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}

func toOrderBySlice(v interface{}) []repository.OrderBy {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []repository.OrderBy
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, repository.OrderBy{
				Field:     toString(m["field"]),
				Direction: toString(m["direction"]),
			})
		}
	}
	return result
}

func toTableRefs(v interface{}) []repository.TableRef {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []repository.TableRef
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, repository.TableRef{
				Name:  toString(m["name"]),
				Alias: toString(m["alias"]),
			})
		}
	}
	return result
}

func toJoinClauses(v interface{}) []repository.JoinClause {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []repository.JoinClause
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, repository.JoinClause{
				Type:      toString(m["type"]),
				FromTable: toString(m["from_table"]),
				FromField: toString(m["from_field"]),
				ToTable:   toString(m["to_table"]),
				ToField:   toString(m["to_field"]),
			})
		}
	}
	return result
}

func toJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}
