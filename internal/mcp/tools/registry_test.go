package tools

import (
	"context"
	"fmt"
	"testing"

	"rocksdb-cli/internal/mcp/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test creating a new registry
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	require.NotNil(t, registry)
	assert.Equal(t, 0, len(registry.ListTools("")))
}

// Test registering local tools
func TestRegistry_RegisterLocal(t *testing.T) {
	registry := NewRegistry()

	// Create a mock tool handler
	handler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
		return &protocol.ToolCallResult{
			Content: []protocol.Content{
				{Type: "text", Text: "success"},
			},
			IsError: false,
		}, nil
	}

	tool := protocol.Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}

	t.Run("registers local tool successfully", func(t *testing.T) {
		err := registry.RegisterLocal(tool, handler)
		require.NoError(t, err)

		tools := registry.ListTools("")
		assert.Equal(t, 1, len(tools))
		assert.Equal(t, "local.test-tool", tools[0].Name)
	})

	t.Run("returns error for duplicate local tool", func(t *testing.T) {
		err := registry.RegisterLocal(tool, handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

// Test registering remote tools
func TestRegistry_RegisterRemote(t *testing.T) {
	registry := NewRegistry()

	tools := []protocol.Tool{
		{
			Name:        "fs-read",
			Description: "Read file",
			InputSchema: map[string]interface{}{"type": "object"},
		},
		{
			Name:        "fs-write",
			Description: "Write file",
			InputSchema: map[string]interface{}{"type": "object"},
		},
	}

	t.Run("registers remote tools successfully", func(t *testing.T) {
		err := registry.RegisterRemote("filesystem", tools)
		require.NoError(t, err)

		allTools := registry.ListTools("")
		assert.Equal(t, 2, len(allTools))
		assert.Equal(t, "filesystem.fs-read", allTools[0].Name)
		assert.Equal(t, "filesystem.fs-write", allTools[1].Name)
	})

	t.Run("updates remote tools on re-registration", func(t *testing.T) {
		newTools := []protocol.Tool{
			{
				Name:        "fs-read",
				Description: "Read file updated",
				InputSchema: map[string]interface{}{"type": "object"},
			},
		}

		err := registry.RegisterRemote("filesystem", newTools)
		require.NoError(t, err)

		allTools := registry.ListTools("")
		assert.Equal(t, 1, len(allTools))
		assert.Contains(t, allTools[0].Description, "updated")
	})
}

// Test listing tools
func TestRegistry_ListTools(t *testing.T) {
	registry := NewRegistry()

	// Register local tool
	localHandler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
		return &protocol.ToolCallResult{}, nil
	}
	registry.RegisterLocal(protocol.Tool{
		Name:        "local-tool",
		Description: "Local tool",
	}, localHandler)

	// Register remote tools
	registry.RegisterRemote("client1", []protocol.Tool{
		{Name: "remote-tool-1", Description: "Remote tool 1"},
		{Name: "remote-tool-2", Description: "Remote tool 2"},
	})
	registry.RegisterRemote("client2", []protocol.Tool{
		{Name: "remote-tool-3", Description: "Remote tool 3"},
	})

	t.Run("lists all tools", func(t *testing.T) {
		tools := registry.ListTools("")
		assert.Equal(t, 4, len(tools))
	})

	t.Run("filters tools by namespace", func(t *testing.T) {
		localTools := registry.ListTools("local")
		assert.Equal(t, 1, len(localTools))
		assert.Equal(t, "local.local-tool", localTools[0].Name)

		client1Tools := registry.ListTools("client1")
		assert.Equal(t, 2, len(client1Tools))

		client2Tools := registry.ListTools("client2")
		assert.Equal(t, 1, len(client2Tools))
	})

	t.Run("returns empty list for unknown namespace", func(t *testing.T) {
		tools := registry.ListTools("unknown")
		assert.Equal(t, 0, len(tools))
	})
}

// Test getting tool by name
func TestRegistry_GetTool(t *testing.T) {
	registry := NewRegistry()

	localHandler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
		return &protocol.ToolCallResult{}, nil
	}
	registry.RegisterLocal(protocol.Tool{
		Name:        "test-tool",
		Description: "Test tool",
	}, localHandler)

	registry.RegisterRemote("client1", []protocol.Tool{
		{Name: "remote-tool", Description: "Remote tool"},
	})

	t.Run("gets local tool by full name", func(t *testing.T) {
		tool := registry.GetTool("local.test-tool")
		require.NotNil(t, tool)
		assert.Equal(t, "local.test-tool", tool.Name)
		assert.Equal(t, "Test tool", tool.Description)
	})

	t.Run("gets remote tool by full name", func(t *testing.T) {
		tool := registry.GetTool("client1.remote-tool")
		require.NotNil(t, tool)
		assert.Equal(t, "client1.remote-tool", tool.Name)
	})

	t.Run("returns nil for non-existent tool", func(t *testing.T) {
		tool := registry.GetTool("non-existent")
		assert.Nil(t, tool)
	})
}

// Test executing local tools
func TestRegistry_ExecuteLocal(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	t.Run("executes local tool successfully", func(t *testing.T) {
		called := false
		handler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
			called = true
			assert.Equal(t, "value", args["key"])
			return &protocol.ToolCallResult{
				Content: []protocol.Content{
					{Type: "text", Text: "success"},
				},
				IsError: false,
			}, nil
		}

		registry.RegisterLocal(protocol.Tool{Name: "test-tool"}, handler)

		result, err := registry.Execute(ctx, "local.test-tool", map[string]interface{}{
			"key": "value",
		})

		require.NoError(t, err)
		assert.True(t, called)
		assert.False(t, result.IsError)
		assert.Equal(t, "success", result.Content[0].Text)
	})

	t.Run("returns error for non-existent tool", func(t *testing.T) {
		_, err := registry.Execute(ctx, "non-existent", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Test tool namespacing
func TestRegistry_Namespacing(t *testing.T) {
	registry := NewRegistry()

	localHandler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
		return &protocol.ToolCallResult{Content: []protocol.Content{{Type: "text", Text: "local"}}}, nil
	}

	// Register same tool name in different namespaces
	registry.RegisterLocal(protocol.Tool{
		Name:        "read",
		Description: "Local read",
	}, localHandler)

	registry.RegisterRemote("filesystem", []protocol.Tool{
		{Name: "read", Description: "FS read"},
	})

	registry.RegisterRemote("database", []protocol.Tool{
		{Name: "read", Description: "DB read"},
	})

	t.Run("tools have unique namespaced names", func(t *testing.T) {
		tools := registry.ListTools("")
		assert.Equal(t, 3, len(tools))

		names := make(map[string]bool)
		for _, tool := range tools {
			names[tool.Name] = true
		}

		assert.True(t, names["local.read"])
		assert.True(t, names["filesystem.read"])
		assert.True(t, names["database.read"])
	})

	t.Run("can get tools by namespaced name", func(t *testing.T) {
		localTool := registry.GetTool("local.read")
		assert.Equal(t, "Local read", localTool.Description)

		fsTool := registry.GetTool("filesystem.read")
		assert.Equal(t, "FS read", fsTool.Description)

		dbTool := registry.GetTool("database.read")
		assert.Equal(t, "DB read", dbTool.Description)
	})
}

// Test unregistering tools
func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	localHandler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
		return &protocol.ToolCallResult{}, nil
	}
	registry.RegisterLocal(protocol.Tool{Name: "test-tool"}, localHandler)
	registry.RegisterRemote("client1", []protocol.Tool{
		{Name: "remote-tool"},
	})

	t.Run("unregisters remote client tools", func(t *testing.T) {
		err := registry.UnregisterRemote("client1")
		require.NoError(t, err)

		tools := registry.ListTools("")
		assert.Equal(t, 1, len(tools))
		assert.Equal(t, "local.test-tool", tools[0].Name)
	})

	t.Run("returns error for non-existent client", func(t *testing.T) {
		err := registry.UnregisterRemote("non-existent")
		assert.Error(t, err)
	})
}

// Test concurrent operations
func TestRegistry_ConcurrentOperations(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	handler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
		return &protocol.ToolCallResult{Content: []protocol.Content{{Type: "text", Text: "ok"}}}, nil
	}

	t.Run("concurrent registrations", func(t *testing.T) {
		errCh := make(chan error, 10)

		// Register 10 tools concurrently
		for i := 0; i < 10; i++ {
			go func(id int) {
				tool := protocol.Tool{
					Name:        fmt.Sprintf("tool-%d", id),
					Description: "Test tool",
				}
				errCh <- registry.RegisterLocal(tool, handler)
			}(i)
		}

		// Collect errors
		for i := 0; i < 10; i++ {
			err := <-errCh
			assert.NoError(t, err)
		}

		tools := registry.ListTools("")
		assert.Equal(t, 10, len(tools))
	})

	t.Run("concurrent executions", func(t *testing.T) {
		errCh := make(chan error, 10)

		// Execute tools concurrently
		for i := 0; i < 10; i++ {
			go func(id int) {
				_, err := registry.Execute(ctx, fmt.Sprintf("local.tool-%d", id), nil)
				errCh <- err
			}(i)
		}

		// Collect errors
		for i := 0; i < 10; i++ {
			err := <-errCh
			assert.NoError(t, err)
		}
	})
}

// Test tool discovery
func TestRegistry_Discovery(t *testing.T) {
	registry := NewRegistry()

	t.Run("returns tool count", func(t *testing.T) {
		assert.Equal(t, 0, registry.Count())

		localHandler := func(ctx context.Context, args map[string]interface{}) (*protocol.ToolCallResult, error) {
			return &protocol.ToolCallResult{}, nil
		}
		registry.RegisterLocal(protocol.Tool{Name: "tool1"}, localHandler)
		assert.Equal(t, 1, registry.Count())

		registry.RegisterRemote("client1", []protocol.Tool{
			{Name: "tool2"},
			{Name: "tool3"},
		})
		assert.Equal(t, 3, registry.Count())
	})

	t.Run("returns namespace list", func(t *testing.T) {
		namespaces := registry.ListNamespaces()
		assert.Equal(t, 2, len(namespaces))
		assert.Contains(t, namespaces, "local")
		assert.Contains(t, namespaces, "client1")
	})
}
