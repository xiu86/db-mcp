package mcp

import (
	"testing"

	"db-mcp/internal/config"
	"db-mcp/internal/driver"
	"db-mcp/pkg/logger"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMCPServer(t *testing.T) {
	cfg := &config.Config{
		Databases: []config.InstanceConfig{
			{
				Type:     "mysql",
				Name:     "default",
				Host:     "localhost",
				Port:     3306,
				Database: "test_db",
			},
		},
		Default: "default",
	}
	log := logger.NewLogger(&config.LogConfig{Level: "info", Format: "text", Output: "stdout"})

	server, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.logger)
}

func TestMCPServer_GetServer(t *testing.T) {
	cfg := config.DefaultConfig()
	log := logger.NewLogger(&config.LogConfig{Level: "info", Format: "text", Output: "stdout"})

	server, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

	mcpServer := server.GetServer()
	assert.NotNil(t, mcpServer)
}

func TestGetArgs(t *testing.T) {
	t.Run("nil arguments", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		args := getArgs(req)
		assert.NotNil(t, args)
		assert.Empty(t, args)
	})

	t.Run("with arguments", func(t *testing.T) {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
			"where": map[string]interface{}{"id": 1},
		}
		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
		assert.Equal(t, 1, args["where"].(map[string]interface{})["id"])
	})
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"number", 123, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToStringSlice(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toStringSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("valid slice", func(t *testing.T) {
		input := []interface{}{"a", "b", "c"}
		result := toStringSlice(input)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("mixed types", func(t *testing.T) {
		input := []interface{}{"a", 123, "c"}
		result := toStringSlice(input)
		assert.Equal(t, []string{"a", "c"}, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []interface{}{}
		result := toStringSlice(input)
		// Empty slice returns nil
		assert.Nil(t, result)
	})
}

func TestToMap(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toMap(nil)
		assert.Nil(t, result)
	})

	t.Run("valid map", func(t *testing.T) {
		input := map[string]interface{}{"key": "value"}
		result := toMap(input)
		assert.Equal(t, input, result)
	})

	t.Run("nested map", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{"name": "Alice"},
			"id":   1,
		}
		result := toMap(input)
		assert.NotNil(t, result)
		assert.Equal(t, "Alice", result["user"].(map[string]interface{})["name"])
	})
}

func TestToMapSlice(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toMapSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("valid slice", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
		}
		result := toMapSlice(input)
		assert.Len(t, result, 2)
		assert.Equal(t, 1, result[0]["id"])
	})

	t.Run("mixed types", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"id": 1},
			"string",
		}
		result := toMapSlice(input)
		assert.Len(t, result, 1)
	})
}

func TestToOrderBySlice(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toOrderBySlice(nil)
		assert.Nil(t, result)
	})

	t.Run("valid slice", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"field": "name", "direction": "asc"},
			map[string]interface{}{"field": "id", "direction": "desc"},
		}
		result := toOrderBySlice(input)
		assert.Len(t, result, 2)
		assert.Equal(t, "name", result[0].Field)
		assert.Equal(t, "asc", result[0].Direction)
		assert.Equal(t, "desc", result[1].Direction)
	})

	t.Run("missing fields", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"field": "name"},
		}
		result := toOrderBySlice(input)
		assert.Len(t, result, 1)
		assert.Equal(t, "name", result[0].Field)
		assert.Equal(t, "", result[0].Direction)
	})
}

func TestToTableRefs(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toTableRefs(nil)
		assert.Nil(t, result)
	})

	t.Run("valid slice", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		}
		result := toTableRefs(input)
		assert.Len(t, result, 2)
		assert.Equal(t, "users", result[0].Name)
		assert.Equal(t, "u", result[0].Alias)
	})

	t.Run("missing alias", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{"name": "users"},
		}
		result := toTableRefs(input)
		assert.Len(t, result, 1)
		assert.Equal(t, "users", result[0].Name)
		assert.Equal(t, "", result[0].Alias)
	})
}

func TestToJoinClauses(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toJoinClauses(nil)
		assert.Nil(t, result)
	})

	t.Run("valid slice", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"type":        "left",
				"from_table":  "u",
				"from_field":  "id",
				"to_table":    "o",
				"to_field":    "user_id",
			},
		}
		result := toJoinClauses(input)
		assert.Len(t, result, 1)
		assert.Equal(t, "left", result[0].Type)
		assert.Equal(t, "u", result[0].FromTable)
		assert.Equal(t, "id", result[0].FromField)
		assert.Equal(t, "o", result[0].ToTable)
		assert.Equal(t, "user_id", result[0].ToField)
	})
}

func TestToOperations(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := toOperations(nil)
		assert.Nil(t, result)
	})

	t.Run("valid operations", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"type":   "insert",
				"table":  "users",
				"data":   map[string]interface{}{"name": "Alice"},
				"where":  map[string]interface{}{},
				"fields": []string{"id"},
			},
			map[string]interface{}{
				"type":  "update",
				"table": "users",
				"data":  map[string]interface{}{"status": "active"},
				"where": map[string]interface{}{"id": 1},
			},
		}
		result := toOperations(input)
		assert.Len(t, result, 2)
		assert.Equal(t, "insert", result[0].Type)
		assert.Equal(t, "users", result[0].Table)
		assert.Equal(t, "update", result[1].Type)
		assert.Equal(t, "active", result[1].Data["status"])
	})

	t.Run("missing fields", func(t *testing.T) {
		input := []interface{}{
			map[string]interface{}{
				"type":  "delete",
				"table": "users",
			},
		}
		result := toOperations(input)
		assert.Len(t, result, 1)
		assert.Equal(t, "delete", result[0].Type)
		assert.Equal(t, "users", result[0].Table)
		assert.Nil(t, result[0].Data)
	})
}

func TestToJSON(t *testing.T) {
	t.Run("valid object", func(t *testing.T) {
		input := map[string]interface{}{"key": "value", "count": 42}
		result := toJSON(input)
		assert.Contains(t, result, "key")
		assert.Contains(t, result, "value")
		assert.Contains(t, result, "42")
	})

	t.Run("array", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := toJSON(input)
		assert.Contains(t, result, "a")
		assert.Contains(t, result, "b")
	})

	t.Run("error handling", func(t *testing.T) {
		// Pass a channel which cannot be marshaled
		input := make(chan int)
		result := toJSON(input)
		assert.Equal(t, "{}", result)
	})
}

func TestOperation_Struct(t *testing.T) {
	op := Operation{
		Type:   "insert",
		Table:  "users",
		Data:   map[string]interface{}{"name": "Alice"},
		Where:  map[string]interface{}{},
		Fields: []string{"id"},
	}

	assert.Equal(t, "insert", op.Type)
	assert.Equal(t, "users", op.Table)
	assert.NotNil(t, op.Data)
	assert.Len(t, op.Fields, 1)
}

func TestBatchResult_JSON(t *testing.T) {
	result := map[string]interface{}{
		"success_count": int64(5),
		"failed_count":  int64(2),
		"errors": []map[string]interface{}{
			{"index": 3, "message": "duplicate key"},
		},
	}

	jsonStr := toJSON(result)
	assert.Contains(t, jsonStr, "5")
	assert.Contains(t, jsonStr, "2")
	assert.Contains(t, jsonStr, "duplicate key")
}

func TestMCPServer_Response_Format(t *testing.T) {
	t.Run("QueryResult format", func(t *testing.T) {
		result := &driver.QueryResult{
			Rows: []map[string]interface{}{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
			Total:   2,
			Message: "success",
		}

		jsonStr := toJSON(result)
		assert.Contains(t, jsonStr, "Alice")
		assert.Contains(t, jsonStr, "Bob")
		assert.Contains(t, jsonStr, "2")
	})

	t.Run("MutationResult format", func(t *testing.T) {
		result := &driver.MutationResult{
			AffectedRows: 1,
			LastInsertID: 100,
			Message:      "Insert successful",
		}

		jsonStr := toJSON(result)
		assert.Contains(t, jsonStr, "1")
		assert.Contains(t, jsonStr, "100")
		assert.Contains(t, jsonStr, "Insert successful")
	})
}

func TestMCPServer_Request_Parsing(t *testing.T) {
	cfg := &config.Config{}
	log := logger.NewLogger(&config.LogConfig{Level: "info", Format: "text", Output: "stdout"})

	t.Run("Insert request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
			"data": map[string]interface{}{
				"name":  "Alice",
				"email": "alice@example.com",
			},
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
		assert.Equal(t, "Alice", args["data"].(map[string]interface{})["name"])
	})

	t.Run("Update request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
			"data": map[string]interface{}{
				"status": "inactive",
			},
			"where": map[string]interface{}{
				"id": 1,
			},
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
		assert.Equal(t, "inactive", args["data"].(map[string]interface{})["status"])
		assert.Equal(t, 1, args["where"].(map[string]interface{})["id"])
	})

	t.Run("Delete request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
			"where": map[string]interface{}{
				"id": 1,
			},
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
		assert.Equal(t, 1, args["where"].(map[string]interface{})["id"])
	})

	t.Run("BatchInsert request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
			"data": []interface{}{
				map[string]interface{}{"name": "User1"},
				map[string]interface{}{"name": "User2"},
				map[string]interface{}{"name": "User3"},
			},
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])

		data := toMapSlice(args["data"])
		assert.Len(t, data, 3)
		assert.Equal(t, "User1", data[0]["name"])
	})

	t.Run("BatchUpdate request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
			"data": []interface{}{
				map[string]interface{}{"id": 1, "status": "active"},
				map[string]interface{}{"id": 2, "status": "inactive"},
			},
			"key_field": "id",
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
		assert.Equal(t, "id", args["key_field"])
	})

	t.Run("BatchDelete request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table":    "users",
			"ids":      []interface{}{"1", "2", "3"},
			"id_field": "id",
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
		assert.Equal(t, "id", args["id_field"])

		ids := toStringSlice(args["ids"])
		assert.Equal(t, []string{"1", "2", "3"}, ids)
	})

	t.Run("Join request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"tables": []interface{}{
				map[string]interface{}{"name": "users", "alias": "u"},
				map[string]interface{}{"name": "orders", "alias": "o"},
			},
			"joins": []interface{}{
				map[string]interface{}{
					"type":        "left",
					"from_table":  "u",
					"from_field":  "id",
					"to_table":    "o",
					"to_field":    "user_id",
				},
			},
			"fields": []interface{}{"u.id", "u.name", "o.total"},
			"where": map[string]interface{}{
				"u.status": "active",
			},
			"limit": float64(100),
		}

		args := getArgs(req)
		tables := toTableRefs(args["tables"])
		joins := toJoinClauses(args["joins"])
		fields := toStringSlice(args["fields"])

		assert.Len(t, tables, 2)
		assert.Len(t, joins, 1)
		assert.Equal(t, "left", joins[0].Type)
		assert.Len(t, fields, 3)
		assert.Equal(t, float64(100), args["limit"])
	})

	t.Run("Transaction request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"operations": []interface{}{
				map[string]interface{}{
					"type":  "insert",
					"table": "users",
					"data":  map[string]interface{}{"name": "Alice"},
				},
				map[string]interface{}{
					"type":  "update",
					"table": "users",
					"data":  map[string]interface{}{"status": "active"},
					"where": map[string]interface{}{"id": 1},
				},
			},
		}

		args := getArgs(req)
		operations := toOperations(args["operations"])

		assert.Len(t, operations, 2)
		assert.Equal(t, "insert", operations[0].Type)
		assert.Equal(t, "update", operations[1].Type)
		assert.Equal(t, "Alice", operations[0].Data["name"])
	})

	t.Run("Schema request", func(t *testing.T) {
		_, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]interface{}{
			"table": "users",
		}

		args := getArgs(req)
		assert.Equal(t, "users", args["table"])
	})
}

func TestJoinRequest_Parsing(t *testing.T) {
	req := map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{"name": "users", "alias": "u"},
			map[string]interface{}{"name": "orders", "alias": "o"},
		},
		"joins": []interface{}{
			map[string]interface{}{
				"type":        "inner",
				"from_table":  "u",
				"from_field":  "id",
				"to_table":    "o",
				"to_field":    "user_id",
			},
		},
		"fields": []interface{}{"u.id", "u.name", "o.id", "o.total"},
		"where": map[string]interface{}{
			"u.status": "active",
			"o.status": "completed",
		},
		"order": []interface{}{
			map[string]interface{}{"field": "u.created_at", "direction": "desc"},
		},
		"limit": float64(50),
	}

	tables := toTableRefs(req["tables"])
	joins := toJoinClauses(req["joins"])
	fields := toStringSlice(req["fields"])
	where := toMap(req["where"])
	order := toOrderBySlice(req["order"])

	assert.Len(t, tables, 2)
	assert.Len(t, joins, 1)
	assert.Len(t, fields, 4)
	assert.Len(t, where, 2)
	assert.Len(t, order, 1)
	assert.Equal(t, float64(50), req["limit"])
}

func TestMCPServer_Config_Passing(t *testing.T) {
	cfg := &config.Config{
		Databases: []config.InstanceConfig{
			{
				Type:     "mysql",
				Name:     "default",
				Host:     "localhost",
				Port:     3306,
				User:     "test_user",
				Password: "test_pass",
				Database: "test_db",
			},
		},
		Default: "default",
		Log: config.LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	log := logger.NewLogger(&cfg.Log)
	server, err := NewMCPServer(nil, cfg, log)
	require.NoError(t, err)

	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
	assert.Equal(t, "test_db", server.config.Databases[0].Database)
}
