package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"db-mcp/internal/config"
	"db-mcp/internal/connection"
	"db-mcp/internal/driver"
	"db-mcp/internal/errors"
	"db-mcp/internal/middleware"
	"db-mcp/internal/repository"
	"db-mcp/internal/service"
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Type aliases for driver types to maintain backward compatibility
type (
	OrderBy    = driver.OrderBy
	TableRef   = driver.TableRef
	JoinClause = driver.JoinClause
)

type MCPServer struct {
	server      *server.MCPServer
	connManager *connection.ConnectionManager
	crud        *service.CRUDService
	txService   *service.TransactionService
	config      *config.Config
	logger      *logger.Logger
	rateLimiter *middleware.RateLimiter
	timeout     *middleware.TimeoutConfig
}

func NewMCPServer(cm *connection.ConnectionManager, cfg *config.Config, log *logger.Logger) *MCPServer {
	s := server.NewMCPServer(
		"db-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	var crudSvc *service.CRUDService
	var txSvc *service.TransactionService
	if cm != nil {
		driver, err := cm.GetDriver(cm.CurrentInstance())
		if err != nil {
			panic(fmt.Sprintf("failed to get driver: %v", err))
		}
		repo := repository.New(driver)
		auditSvc := service.NewAuditServiceWithDB(cfg.Log.AuditFile, cm.DB())
		crudSvc = service.NewCRUDService(repo, auditSvc, cfg, log)
		txSvc = service.NewTransactionService(cm.DB(), auditSvc, cfg, log)
	}
	rateLimiter := middleware.NewRateLimiter(&cfg.RateLimit)
	timeout := middleware.DefaultTimeoutConfig()
	if cfg.Timeout.Connect > 0 {
		timeout.Connect = time.Duration(cfg.Timeout.Connect) * time.Second
	}
	if cfg.Timeout.Query > 0 {
		timeout.Query = time.Duration(cfg.Timeout.Query) * time.Second
	}
	if cfg.Timeout.Mutation > 0 {
		timeout.Mutation = time.Duration(cfg.Timeout.Mutation) * time.Second
	}
	if cfg.Timeout.Transaction > 0 {
		timeout.Transaction = time.Duration(cfg.Timeout.Transaction) * time.Second
	}

	mcpServer := &MCPServer{
		server:      s,
		connManager: cm,
		crud:        crudSvc,
		txService:   txSvc,
		config:      cfg,
		logger:      log,
		rateLimiter: rateLimiter,
		timeout:     timeout,
	}

	mcpServer.registerTools()
	return mcpServer
}

func (s *MCPServer) checkRateLimit() error {
	if s.rateLimiter == nil || !s.rateLimiter.Enabled() {
		return nil
	}
	if !s.rateLimiter.Allow() {
		return errors.NewError(errors.ErrRateLimit, "rate limit exceeded", nil)
	}
	return nil
}

func (s *MCPServer) withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout == nil {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.timeout.QueryTimeout())
}

func (s *MCPServer) withMutationTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout == nil {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.timeout.MutationTimeout())
}

func (s *MCPServer) withTransactionTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout == nil {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.timeout.TransactionTimeout())
}

func (s *MCPServer) GetServer() *server.MCPServer {
	return s.server
}

// Close closes all resources held by the MCP server (audit log file, etc.)
func (s *MCPServer) Close() error {
	if s.crud != nil {
		s.crud.Close()
	}
	return nil
}

func (s *MCPServer) registerTools() {
	// db_query tool
	s.server.AddTool(mcp.NewTool("db_query",
		mcp.WithDescription("Query data from database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name to query")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
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
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithObject("data", mcp.Required(), mcp.Description("Data to insert")),
	), s.handleInsert)

	// db_update tool
	s.server.AddTool(mcp.NewTool("db_update",
		mcp.WithDescription("Update data in database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithObject("data", mcp.Required(), mcp.Description("Data to update")),
		mcp.WithObject("where", mcp.Required(), mcp.Description("Where conditions")),
	), s.handleUpdate)

	// db_delete tool (logical delete only)
	s.server.AddTool(mcp.NewTool("db_delete",
		mcp.WithDescription("Logical delete from database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithObject("where", mcp.Required(), mcp.Description("Where conditions")),
	), s.handleDelete)

	// db_batch_insert tool
	s.server.AddTool(mcp.NewTool("db_batch_insert",
		mcp.WithDescription("Batch insert data into database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("Array of data to insert")),
	), s.handleBatchInsert)

	// db_batch_update tool
	s.server.AddTool(mcp.NewTool("db_batch_update",
		mcp.WithDescription("Batch update data in database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithArray("data", mcp.Required(), mcp.Description("Array of data to update")),
		mcp.WithString("key_field", mcp.Description("Key field for matching (default: id)")),
	), s.handleBatchUpdate)

	// db_batch_delete tool
	s.server.AddTool(mcp.NewTool("db_batch_delete",
		mcp.WithDescription("Batch logical delete from database table"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithArray("ids", mcp.Required(), mcp.Description("Array of IDs to delete")),
		mcp.WithString("id_field", mcp.Description("ID field name (default: id)")),
	), s.handleBatchDelete)

	// db_join tool
	s.server.AddTool(mcp.NewTool("db_join",
		mcp.WithDescription("Perform JOIN query across multiple tables"),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithArray("tables", mcp.Required(), mcp.Description("Tables with aliases")),
		mcp.WithArray("joins", mcp.Required(), mcp.Description("Join clauses")),
		mcp.WithArray("fields", mcp.Description("Fields to select")),
		mcp.WithObject("where", mcp.Description("Where conditions")),
		mcp.WithNumber("limit", mcp.Description("Maximum rows")),
	), s.handleJoin)

	// db_transaction tool
	s.server.AddTool(mcp.NewTool("db_transaction",
		mcp.WithDescription("Execute operations in a transaction"),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
		mcp.WithArray("operations", mcp.Required(), mcp.Description("Array of operations to execute")),
	), s.handleTransaction)

	// db_describe tool
	s.server.AddTool(mcp.NewTool("db_describe",
		mcp.WithDescription("Get table schema and detected delete fields"),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name to get schema")),
		mcp.WithString("instance", mcp.Description("Database instance name (optional, uses current if not specified)")),
	), s.handleSchema)

	// db_switch tool - switch between database instances
	s.server.AddTool(mcp.NewTool("db_switch",
		mcp.WithDescription("Switch to a different database instance"),
		mcp.WithString("instance", mcp.Required(), mcp.Description("Instance name to switch to")),
	), s.handleSwitch)

	// db_list_instances tool - list all available database instances
	s.server.AddTool(mcp.NewTool("db_list_instances",
		mcp.WithDescription("List all available database instances"),
	), s.handleListInstances)
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
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	timeoutCtx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	args := getArgs(request)
	table, _ := args["table"].(string)
	fields := toStringSlice(args["fields"])
	where, _ := args["where"].(map[string]interface{})
	order := toOrderBySlice(args["order"])
	limit, _ := args["limit"].(float64)
	offset, _ := args["offset"].(float64)

	result, err := s.crud.Query(timeoutCtx, table, fields, where, order, int(limit), int(offset))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleInsert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	timeoutCtx, cancel := s.withMutationTimeout(ctx)
	defer cancel()

	args := getArgs(request)
	table, _ := args["table"].(string)
	data, _ := args["data"].(map[string]interface{})

	result, err := s.crud.Insert(timeoutCtx, table, data)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	timeoutCtx, cancel := s.withMutationTimeout(ctx)
	defer cancel()

	args := getArgs(request)
	table, _ := args["table"].(string)
	data, _ := args["data"].(map[string]interface{})
	where, _ := args["where"].(map[string]interface{})

	result, err := s.crud.Update(timeoutCtx, table, data, where)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	timeoutCtx, cancel := s.withMutationTimeout(ctx)
	defer cancel()

	args := getArgs(request)
	table, _ := args["table"].(string)
	where, _ := args["where"].(map[string]interface{})

	result, err := s.crud.Delete(timeoutCtx, table, where)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleBatchInsert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := getArgs(request)
	table, _ := args["table"].(string)
	dataArray := toMapSlice(args["data"])

	if len(dataArray) > 1000 {
		return mcp.NewToolResultError(errors.NewError(errors.ErrInvalidInput, "batch insert exceeds maximum of 1000 items", nil).Error()), nil
	}

	timeoutCtx, cancel := s.withMutationTimeout(ctx)
	defer cancel()

	result, err := s.crud.BatchInsert(timeoutCtx, table, dataArray)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleBatchUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := getArgs(request)
	table, _ := args["table"].(string)
	dataArray := toMapSlice(args["data"])
	keyField, _ := args["key_field"].(string)
	if keyField == "" {
		keyField = "id"
	}

	if len(dataArray) > 1000 {
		return mcp.NewToolResultError(errors.NewError(errors.ErrInvalidInput, "batch update exceeds maximum of 1000 items", nil).Error()), nil
	}

	timeoutCtx, cancel := s.withMutationTimeout(ctx)
	defer cancel()

	result, err := s.crud.BatchUpdate(timeoutCtx, table, dataArray, keyField)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleBatchDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := getArgs(request)
	table, _ := args["table"].(string)
	ids := toStringSlice(args["ids"])
	idField, _ := args["id_field"].(string)
	if idField == "" {
		idField = "id"
	}

	if len(ids) > 1000 {
		return mcp.NewToolResultError(errors.NewError(errors.ErrInvalidInput, "batch delete exceeds maximum of 1000 IDs", nil).Error()), nil
	}

	timeoutCtx, cancel := s.withMutationTimeout(ctx)
	defer cancel()

	result, err := s.crud.BatchDelete(timeoutCtx, table, ids, idField)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleJoin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := getArgs(request)
	tables := toTableRefs(args["tables"])
	if len(tables) > 5 {
		return mcp.NewToolResultError(errors.NewError(errors.ErrInvalidInput, "join exceeds maximum of 5 tables", nil).Error()), nil
	}

	joinReq := &repository.JoinRequest{
		Tables: tables,
		Joins:  toJoinClauses(args["joins"]),
		Fields: toStringSlice(args["fields"]),
		Where:  toMap(args["where"]),
	}
	if limit, ok := args["limit"].(float64); ok {
		joinReq.Limit = int(limit)
	}

	timeoutCtx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	result, err := s.crud.Join(timeoutCtx, joinReq)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := getArgs(request)
	operations := toOperations(args["operations"])
	if len(operations) < 2 {
		return mcp.NewToolResultError(errors.NewError(errors.ErrInvalidInput, "transaction requires at least 2 operations", nil).Error()), nil
	}
	if len(operations) > 50 {
		return mcp.NewToolResultError(errors.NewError(errors.ErrInvalidInput, "transaction exceeds maximum of 50 operations", nil).Error()), nil
	}

	timeoutCtx, cancel := s.withTransactionTimeout(ctx)
	defer cancel()

	txCtx, err := s.txService.Begin(timeoutCtx)
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
		default:
			opErr = fmt.Errorf("unknown operation type: %s", op.Type)
		}

		if opErr != nil {
			_ = txCtx.Rollback() // Best effort rollback
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

func (s *MCPServer) handleSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := s.checkRateLimit(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	timeoutCtx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	args := getArgs(request)
	table, _ := args["table"].(string)

	result, err := s.crud.GetSchema(timeoutCtx, table)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(toJSON(result)), nil
}

func (s *MCPServer) handleSwitch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	instance, _ := args["instance"].(string)

	if err := s.connManager.SwitchInstance(instance); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(`{"status":"switched","instance":"%s"}`, instance)), nil
}

func (s *MCPServer) handleListInstances(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	instances := s.connManager.ListInstances()
	current := s.connManager.CurrentInstance()

	return mcp.NewToolResultText(toJSON(map[string]interface{}{
		"instances": instances,
		"current":    current,
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

func toOrderBySlice(v interface{}) []OrderBy {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []OrderBy
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, OrderBy{
				Field:     toString(m["field"]),
				Direction: toString(m["direction"]),
			})
		}
	}
	return result
}

func toTableRefs(v interface{}) []TableRef {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []TableRef
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, TableRef{
				Name:  toString(m["name"]),
				Alias: toString(m["alias"]),
			})
		}
	}
	return result
}

func toJoinClauses(v interface{}) []JoinClause {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var result []JoinClause
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, JoinClause{
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
