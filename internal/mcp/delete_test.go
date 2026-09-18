package mcp

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"db-mcp/internal/config"
	"db-mcp/internal/detector"
	"db-mcp/internal/driver"
	"db-mcp/internal/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Exercise configuration loading, MCP parsing and service policy together;
// only the external database boundary is mocked.
func TestDeletePhysicalPolicy(t *testing.T) {
	for _, batch := range []bool{false, true} {
		name := "single"
		if batch {
			name = "batch"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name      string
				config    string
				flag      interface{}
				physical  bool
				wantError string
			}{
				{"defaults stay logical", "", nil, false, ""},
				{"enabled still defaults logical", "allowPhysicalDelete: true\n", nil, false, ""},
				{"explicit logical", "allowPhysicalDelete: true\n", false, false, ""},
				{"missing config denies physical", "", true, false, "physical deletion is disabled"},
				{"disabled denies physical", "allowPhysicalDelete: false\n", true, false, "physical deletion is disabled"},
				{"enabled permits physical", "allowPhysicalDelete: true\n", true, true, ""},
				{"string flag is rejected", "allowPhysicalDelete: true\n", "true", false, "physical_delete must be a boolean"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					dir := t.TempDir()
					path := filepath.Join(dir, "config.yaml")
					require.NoError(t, os.WriteFile(path, []byte(tc.config), 0600))
					cfg, err := config.Load(path)
					require.NoError(t, err)
					cfg.Log.AuditFile = filepath.Join(dir, "audit.log")
					cfg.RateLimit.Enabled = false
					s, err := NewMCPServer(nil, cfg, nil)
					require.NoError(t, err)
					repo := driver.NewMockDatabaseDriver(gomock.NewController(t))
					s.crud = service.NewCRUDService(repo, service.NewAuditService(cfg.Log.AuditFile), cfg, nil)
					t.Cleanup(func() { _ = s.Close() })
					if tc.wantError == "" {
						if !tc.physical {
							repo.EXPECT().GetTableSchema("users").Return(&driver.TableSchema{Columns: []driver.ColumnInfo{{Name: "is_deleted", DataType: "tinyint"}}}, nil)
						}
						checkRequest := func(physical bool, field *detector.DeleteFieldInfo) {
							require.Equal(t, tc.physical, physical)
							if tc.physical {
								require.Nil(t, field)
							} else {
								require.NotNil(t, field)
							}
						}
						if batch {
							repo.EXPECT().BatchDelete(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *driver.BatchDeleteRequest) (*driver.BatchResult, error) {
								checkRequest(req.PhysicalDelete, req.DeleteField)
								require.Equal(t, []string{"1", "2"}, req.IDs)
								return &driver.BatchResult{SuccessCount: 2}, nil
							})
						} else {
							repo.EXPECT().Query(gomock.Any(), gomock.Any()).Return(&driver.QueryResult{}, nil)
							repo.EXPECT().Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *driver.DeleteRequest) (*driver.MutationResult, error) {
								checkRequest(req.PhysicalDelete, req.DeleteField)
								require.Equal(t, map[string]interface{}{"id": 1}, req.Where)
								return &driver.MutationResult{AffectedRows: 1}, nil
							})
						}
					}
					args := map[string]interface{}{"table": "users", "where": map[string]interface{}{"id": 1}, "ids": []interface{}{"1", "2"}}
					if tc.flag != nil {
						args["physical_delete"] = tc.flag
					}
					req := mcp.CallToolRequest{}
					req.Params.Arguments = args
					handler := s.handleDelete
					if batch {
						handler = s.handleBatchDelete
					}
					result, err := handler(context.Background(), req)
					require.NoError(t, err)
					if tc.wantError != "" {
						require.True(t, result.IsError)
						require.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.wantError)
					} else {
						require.False(t, result.IsError)
					}
				})
			}
		})
	}
}

func TestTransactionPhysicalDeleteRejectedBeforeBegin(t *testing.T) {
	for _, tc := range []struct {
		name      string
		allow     bool
		flag      interface{}
		where     map[string]interface{}
		wantError string
	}{
		{"disabled", false, true, map[string]interface{}{"id": 1}, "physical deletion is disabled"},
		{"invalid boolean", true, "true", map[string]interface{}{"id": 1}, "physical_delete must be a boolean"},
		{"empty where", true, true, nil, "physical deletion requires non-empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.AllowPhysicalDelete = tc.allow
			s, err := NewMCPServer(nil, cfg, nil)
			require.NoError(t, err)
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]interface{}{"operations": []interface{}{
				map[string]interface{}{"type": "insert", "table": "users", "data": map[string]interface{}{"id": 1}},
				map[string]interface{}{"type": "delete", "table": "users", "where": tc.where, "physical_delete": tc.flag},
			}}
			// No transaction connection is provided: forbidden operations must be
			// rejected before beginning a transaction or executing earlier writes.
			result, err := s.handleTransaction(context.Background(), req)
			require.NoError(t, err)
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].(mcp.TextContent).Text, tc.wantError)
		})
	}
}

// SQL generation stays real; this pool supplies transaction boundaries only,
// while GORM DryRun prevents all database reads/writes.
type deleteTransactionPool struct {
	gorm.ConnPool
	begins, commits, rollbacks int
}

func (p *deleteTransactionPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	p.begins++
	return p, nil
}
func (p *deleteTransactionPool) Commit() error   { p.commits++; return nil }
func (p *deleteTransactionPool) Rollback() error { p.rollbacks++; return nil }

func TestTransactionPhysicalDeleteCommitAndRollback(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			pool := &deleteTransactionPool{}
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
			require.NoError(t, err)
			var statements []string
			require.NoError(t, db.Callback().Delete().After("gorm:delete").Register("test:delete", func(tx *gorm.DB) {
				statements = append(statements, tx.Statement.SQL.String())
			}))
			cfg := config.DefaultConfig()
			cfg.AllowPhysicalDelete = true
			cfg.RateLimit.Enabled = false
			s, err := NewMCPServer(nil, cfg, nil)
			require.NoError(t, err)
			s.txService = service.NewTransactionService(db, nil, cfg, nil)
			secondType := "delete"
			if rollback {
				secondType = "unsupported"
			}
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]interface{}{"operations": []interface{}{
				map[string]interface{}{"type": "delete", "table": "users", "where": map[string]interface{}{"id": 1}, "physical_delete": true},
				map[string]interface{}{"type": secondType, "table": "users", "where": map[string]interface{}{"id": 2}, "physical_delete": true},
			}}
			result, err := s.handleTransaction(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, rollback, result.IsError)
			require.Equal(t, 1, pool.begins)
			if rollback {
				require.Equal(t, 1, pool.rollbacks)
				require.Zero(t, pool.commits)
				require.Len(t, statements, 1)
			} else {
				require.Equal(t, 1, pool.commits)
				require.Zero(t, pool.rollbacks)
				require.Len(t, statements, 2)
			}
			for _, statement := range statements {
				require.Equal(t, "DELETE FROM `users` WHERE `users`.`id` = ?", statement)
			}
		})
	}
}

func TestDeleteRejectsNonBooleanMode(t *testing.T) {
	for _, value := range []interface{}{nil, 0, 1, "false", []interface{}{true}} {
		cfg := config.DefaultConfig()
		cfg.AllowPhysicalDelete = true
		s, err := NewMCPServer(nil, cfg, nil)
		require.NoError(t, err)
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users", "where": map[string]interface{}{"id": 1},
			"ids": []interface{}{"1"}, "physical_delete": value,
		}
		for _, handler := range []func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){s.handleDelete, s.handleBatchDelete} {
			result, err := handler(context.Background(), req)
			require.NoError(t, err)
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].(mcp.TextContent).Text, "physical_delete must be a boolean")
		}
	}
}
