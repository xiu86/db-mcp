package repository_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "db-mcp/internal/detector"
    "db-mcp/internal/repository"
)

func TestNewRepository(t *testing.T) {
    repo := repository.New(nil)
    assert.NotNil(t, repo)
}

func TestQueryRequest_Validation(t *testing.T) {
    req := &repository.QueryRequest{Table: "users"}
    assert.Equal(t, "users", req.Table)
    assert.Equal(t, 0, req.Limit)
    assert.Equal(t, 0, req.Offset)
}

func TestDeleteRequest_WithDeleteField(t *testing.T) {
    req := &repository.DeleteRequest{
        Table:        "users",
        Where:        map[string]interface{}{"id": 1},
        DeleteField: &detector.DeleteFieldInfo{
            TableName: "users",
            Fields: []detector.Field{
                {Name: "deleted_time", Type: "datetime", TrueValue: "0000-00-00 00:00:00"},
            },
        },
    }
    assert.NotNil(t, req.DeleteField)
    assert.Equal(t, "deleted_time", req.DeleteField.Fields[0].Name)
}
