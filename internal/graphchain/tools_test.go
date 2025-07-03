package graphchain

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"rocksdb-cli/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "rocksdb-test-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	database, err := db.Open(dir)
	require.NoError(t, err)
	return database
}

func TestParseToolInput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected *ToolInput
	}{
		{
			name:  "valid JSON input",
			input: `{"args": {"key": "test-key", "value": "test-value"}}`,
			expected: &ToolInput{
				Args: map[string]interface{}{
					"key":   "test-key",
					"value": "test-value",
				},
			},
		},
		{
			name:  "simple string input",
			input: "simple text",
			expected: &ToolInput{
				Args: map[string]interface{}{
					"input": "simple text",
				},
			},
		},
		{
			name:  "empty input",
			input: "",
			expected: &ToolInput{
				Args: map[string]interface{}{
					"input": "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseToolInput(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetStringHelper(t *testing.T) {
	args := map[string]interface{}{
		"string_key": "test-value",
		"int_key":    123,
		"bool_key":   true,
	}

	assert.Equal(t, "test-value", getString(args, "string_key"))
	assert.Equal(t, "", getString(args, "int_key"))
	assert.Equal(t, "", getString(args, "bool_key"))
	assert.Equal(t, "", getString(args, "nonexistent"))
}

func TestGetIntHelper(t *testing.T) {
	args := map[string]interface{}{
		"int_key":     123,
		"float_key":   456.0,
		"string_key":  "789",
		"invalid_key": "abc",
		"bool_key":    true,
	}

	assert.Equal(t, 123, getInt(args, "int_key"))
	assert.Equal(t, 456, getInt(args, "float_key"))
	assert.Equal(t, 789, getInt(args, "string_key"))
	assert.Equal(t, 0, getInt(args, "invalid_key"))
	assert.Equal(t, 0, getInt(args, "bool_key"))
	assert.Equal(t, 0, getInt(args, "nonexistent"))
}

func TestGetBoolHelper(t *testing.T) {
	args := map[string]interface{}{
		"true_key":   true,
		"false_key":  false,
		"string_key": "test",
		"int_key":    123,
	}

	assert.True(t, getBool(args, "true_key"))
	assert.False(t, getBool(args, "false_key"))
	assert.False(t, getBool(args, "string_key"))
	assert.False(t, getBool(args, "int_key"))
	assert.False(t, getBool(args, "nonexistent"))
}

func TestGetValueTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewGetValueTool(database)

	// Prepare test data
	err := database.PutCF("default", "test-key", "test-value")
	require.NoError(t, err)

	t.Run("successful get", func(t *testing.T) {
		input := `{"args": {"key": "test-key"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-key", response["key"])
		assert.Equal(t, "test-value", response["value"])
		assert.Equal(t, "default", response["column_family"])
	})

	t.Run("get with custom column family", func(t *testing.T) {
		input := `{"args": {"key": "test-key", "column_family": "default"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "default", response["column_family"])
	})

	t.Run("missing key parameter", func(t *testing.T) {
		input := `{"args": {}}`
		_, err := tool.Call(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key parameter is required")
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "get_value", tool.Name())
		assert.Contains(t, tool.Description(), "Get a value by key")
	})
}

func TestPutValueTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewPutValueTool(database)

	t.Run("successful put", func(t *testing.T) {
		input := `{"args": {"key": "new-key", "value": "new-value"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Equal(t, "new-key", response["key"])
		assert.Equal(t, "new-value", response["value"])
		assert.Equal(t, "default", response["column_family"])

		// Verify data was actually stored
		value, err := database.GetCF("default", "new-key")
		require.NoError(t, err)
		assert.Equal(t, "new-value", value)
	})

	t.Run("put with custom column family", func(t *testing.T) {
		input := `{"args": {"key": "cf-key", "value": "cf-value", "column_family": "default"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "default", response["column_family"])
	})

	t.Run("missing key parameter", func(t *testing.T) {
		input := `{"args": {"value": "test-value"}}`
		_, err := tool.Call(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both key and value parameters are required")
	})

	t.Run("missing value parameter", func(t *testing.T) {
		input := `{"args": {"key": "test-key"}}`
		_, err := tool.Call(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both key and value parameters are required")
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "put_value", tool.Name())
		assert.Contains(t, tool.Description(), "Put a key-value pair")
	})
}

func TestScanRangeTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewScanRangeTool(database)

	// Prepare test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
	}

	for k, v := range testData {
		err := database.PutCF("default", k, v)
		require.NoError(t, err)
	}

	t.Run("successful range scan", func(t *testing.T) {
		input := `{"args": {"start_key": "key1", "end_key": "key3", "limit": 5}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "default", response["column_family"])
		assert.Equal(t, "key1", response["start_key"])
		assert.Equal(t, "key3", response["end_key"])
		assert.Contains(t, response, "results")
	})

	t.Run("scan with default limit", func(t *testing.T) {
		input := `{"args": {"start_key": "key1"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "results")
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "scan_range", tool.Name())
		assert.Contains(t, tool.Description(), "Scan a range of keys")
	})
}

func TestPrefixScanTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewPrefixScanTool(database)

	// Prepare test data
	testData := map[string]string{
		"user:1":  "alice",
		"user:2":  "bob",
		"admin:1": "root",
		"user:3":  "charlie",
	}

	for k, v := range testData {
		err := database.PutCF("default", k, v)
		require.NoError(t, err)
	}

	t.Run("successful prefix scan", func(t *testing.T) {
		input := `{"args": {"prefix": "user:", "limit": 10}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "user:", response["prefix"])
		assert.Equal(t, "default", response["column_family"])
		assert.Contains(t, response, "results")

		// Check that results contain user keys
		results, ok := response["results"].(map[string]interface{})
		if !ok {
			// Try as array if map conversion fails
			resultsArray, arrayOk := response["results"].([]interface{})
			assert.True(t, arrayOk || ok, "Results should be either map or array")
			if arrayOk {
				assert.True(t, len(resultsArray) >= 1, "Should find at least 1 result")
			}
		} else {
			assert.True(t, len(results) >= 1, "Should find at least 1 result") // Should find at least some user keys
		}
	})

	t.Run("missing prefix parameter", func(t *testing.T) {
		input := `{"args": {}}`
		_, err := tool.Call(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prefix parameter is required")
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "prefix_scan", tool.Name())
		assert.Contains(t, tool.Description(), "Scan keys with a specific prefix")
	})
}

func TestListColumnFamiliesTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewListColumnFamiliesTool(database)

	t.Run("successful list", func(t *testing.T) {
		input := `{"args": {}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "column_families")
		cfs := response["column_families"].([]interface{})
		assert.True(t, len(cfs) >= 1) // Should at least have "default"
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "list_column_families", tool.Name())
		assert.Contains(t, tool.Description(), "List all column families")
	})
}

func TestGetLastTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewGetLastTool(database)

	// Prepare test data
	err := database.PutCF("default", "last-key", "last-value")
	require.NoError(t, err)

	t.Run("successful get last", func(t *testing.T) {
		input := `{"args": {}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "default", response["column_family"])
		assert.Contains(t, response, "key")
		assert.Contains(t, response, "value")
	})

	t.Run("get last with custom column family", func(t *testing.T) {
		input := `{"args": {"column_family": "default"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "default", response["column_family"])
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "get_last", tool.Name())
		assert.Contains(t, tool.Description(), "Get the last key-value pair")
	})
}

func TestJSONQueryTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewJSONQueryTool(database)

	// Prepare JSON test data
	jsonData := map[string]string{
		"user:1": `{"name": "alice", "age": 30}`,
		"user:2": `{"name": "bob", "age": 25}`,
		"user:3": `{"name": "charlie", "age": 35}`,
	}

	for k, v := range jsonData {
		err := database.PutCF("default", k, v)
		require.NoError(t, err)
	}

	t.Run("successful JSON query", func(t *testing.T) {
		input := `{"args": {"field": "name", "value": "alice"}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Equal(t, "name", response["field"])
		assert.Equal(t, "alice", response["value"])
		assert.Equal(t, "default", response["column_family"])
		assert.Contains(t, response, "results")
	})

	t.Run("missing field parameter", func(t *testing.T) {
		input := `{"args": {"value": "alice"}}`
		_, err := tool.Call(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both field and value parameters are required")
	})

	t.Run("missing value parameter", func(t *testing.T) {
		input := `{"args": {"field": "name"}}`
		_, err := tool.Call(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "both field and value parameters are required")
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "json_query", tool.Name())
		assert.Contains(t, tool.Description(), "Query JSON values by field")
	})
}

func TestGetStatsTool(t *testing.T) {
	database := setupTestDB(t)
	tool := NewGetStatsTool(database)

	t.Run("successful get stats", func(t *testing.T) {
		input := `{"args": {}}`
		result, err := tool.Call(context.Background(), input)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "stats")
	})

	t.Run("tool metadata", func(t *testing.T) {
		assert.Equal(t, "get_stats", tool.Name())
		assert.Contains(t, tool.Description(), "Get RocksDB database statistics")
	})
}

func TestCreateRocksDBTools(t *testing.T) {
	database := setupTestDB(t)
	tools := CreateRocksDBTools(database)

	assert.Equal(t, 8, len(tools), "Should create 8 tools")

	// Verify all tools are created
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		// All tools should implement the Name() method
		if namedTool, ok := tool.(interface{ Name() string }); ok {
			toolNames[namedTool.Name()] = true
		}
	}

	expectedTools := []string{
		"get_value", "put_value", "scan_range", "prefix_scan",
		"list_column_families", "get_last", "json_query", "get_stats",
	}

	for _, expected := range expectedTools {
		assert.True(t, toolNames[expected], "Tool %s should be created", expected)
	}
}
