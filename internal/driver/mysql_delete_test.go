package driver

import (
	"context"
	"testing"

	"db-mcp/internal/detector"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMySQLDeleteModeSQL(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, physical := range []bool{false, true} {
			name := "single_logical"
			if batch {
				name = "batch_logical"
			}
			if physical {
				name += "_physical"
			}
			t.Run(name, func(t *testing.T) {
				db, err := gorm.Open(mysql.New(mysql.Config{DSN: "unused:unused@tcp(localhost:1)/unused", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
				require.NoError(t, err)
				sqlDB, err := db.DB()
				require.NoError(t, err)
				t.Cleanup(func() { _ = sqlDB.Close() })
				var statements []string
				capture := func(tx *gorm.DB) { statements = append(statements, tx.Statement.SQL.String()) }
				require.NoError(t, db.Callback().Delete().After("gorm:delete").Register("test:capture_delete", capture))
				require.NoError(t, db.Callback().Update().After("gorm:update").Register("test:capture_update", capture))
				d := &MySQLDriver{db: db}
				deleteField := &detector.DeleteFieldInfo{Fields: []detector.Field{{Name: "is_deleted", TrueValue: "1"}}}
				if physical {
					deleteField = nil
				}

				if batch {
					req := &BatchDeleteRequest{PhysicalDelete: physical, Table: "users", IDs: []string{"1", "2"}, IDField: "id", DeleteField: deleteField}
					result, err := d.BatchDelete(context.Background(), req)
					require.NoError(t, err)
					require.Zero(t, result.FailedCount)
					require.Len(t, statements, 2)
				} else {
					req := &DeleteRequest{PhysicalDelete: physical, Table: "users", Where: map[string]interface{}{"id": 1}, DeleteField: deleteField}
					_, err := d.Delete(context.Background(), req)
					require.NoError(t, err)
					require.Len(t, statements, 1)
				}
				for _, sql := range statements {
					where := " WHERE `users`.`id` = ?"
					if batch {
						where = " WHERE `id` = ?"
					}
					if physical {
						require.Equal(t, "DELETE FROM `users`"+where, sql)
					} else {
						require.Equal(t, "UPDATE `users` SET `is_deleted`=?"+where, sql)
					}
				}
			})
		}
	}
}
