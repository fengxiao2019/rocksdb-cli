package client

import (
	"context"
	"testing"
	"time"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Client interface conformance
func TestClient_Interface(t *testing.T) {
	// This test ensures that implementations conform to the Client interface
	var _ Client = (*BaseClient)(nil) // BaseClient should implement Client
}

// Test Client connection lifecycle
func TestClient_Connect(t *testing.T) {
	t.Run("connect establishes connection", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)
		assert.True(t, client.IsConnected())
	})

	t.Run("connect with timeout", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := client.Connect(ctx)
		require.NoError(t, err)
		assert.True(t, client.IsConnected())
	})

	t.Run("connect returns error when already connected", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		// Try connecting again
		err = client.Connect(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already connected")
	})
}

func TestClient_Disconnect(t *testing.T) {
	t.Run("disconnect closes connection", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		err = client.Disconnect(ctx)
		require.NoError(t, err)
		assert.False(t, client.IsConnected())
	})

	t.Run("disconnect when not connected", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Disconnect(ctx)
		assert.NoError(t, err) // Should not error
	})
}

func TestClient_Initialize(t *testing.T) {
	t.Run("initialize sends initialize request", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		result, err := client.Initialize(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, protocol.MCPProtocolVersion, result.ProtocolVersion)
		assert.NotNil(t, result.ServerInfo)
	})

	t.Run("initialize fails when not connected", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		_, err := client.Initialize(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

func TestClient_ListTools(t *testing.T) {
	t.Run("list tools returns available tools", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		_, err = client.Initialize(ctx)
		require.NoError(t, err)

		tools, err := client.ListTools(ctx, "")
		require.NoError(t, err)
		assert.NotNil(t, tools)
		assert.GreaterOrEqual(t, len(tools.Tools), 0)
	})

	t.Run("list tools with cursor pagination", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		_, err = client.Initialize(ctx)
		require.NoError(t, err)

		result, err := client.ListTools(ctx, "")
		require.NoError(t, err)

		if result.NextCursor != "" {
			// If there's a cursor, fetch next page
			nextResult, err := client.ListTools(ctx, result.NextCursor)
			require.NoError(t, err)
			assert.NotNil(t, nextResult)
		}
	})
}

func TestClient_CallTool(t *testing.T) {
	t.Run("call tool executes successfully", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		_, err = client.Initialize(ctx)
		require.NoError(t, err)

		result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{
			"param1": "value1",
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
	})

	t.Run("call tool with error result", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		_, err = client.Initialize(ctx)
		require.NoError(t, err)

		result, err := client.CallTool(ctx, "error_tool", nil)
		require.NoError(t, err) // No transport error
		assert.True(t, result.IsError)
	})

	t.Run("call tool fails when not initialized", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		_, err = client.CallTool(ctx, "test_tool", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

func TestClient_Ping(t *testing.T) {
	t.Run("ping succeeds when connected", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Connect(ctx)
		require.NoError(t, err)

		err = client.Ping(ctx)
		require.NoError(t, err)
	})

	t.Run("ping fails when not connected", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		err := client.Ping(ctx)
		assert.Error(t, err)
	})
}

func TestClient_GetConfig(t *testing.T) {
	t.Run("get config returns client configuration", func(t *testing.T) {
		cfg := &config.MCPClientConfig{
			Name:      "test-client",
			Enabled:   true,
			Transport: "stdio",
			Timeout:   30 * time.Second,
		}

		client := NewMockClient("test-client")
		client.SetConfig(cfg)

		gotConfig := client.GetConfig()
		assert.NotNil(t, gotConfig)
		assert.Equal(t, "test-client", gotConfig.Name)
	})
}

func TestClient_GetName(t *testing.T) {
	t.Run("get name returns client name", func(t *testing.T) {
		client := NewMockClient("my-client")
		assert.Equal(t, "my-client", client.GetName())
	})
}

func TestClient_StateTransitions(t *testing.T) {
	t.Run("proper state transitions", func(t *testing.T) {
		client := NewMockClient("test-client")
		ctx := context.Background()

		// Initial state: not connected
		assert.False(t, client.IsConnected())

		// Connect
		err := client.Connect(ctx)
		require.NoError(t, err)
		assert.True(t, client.IsConnected())

		// Initialize
		_, err = client.Initialize(ctx)
		require.NoError(t, err)

		// Use client (call tool)
		_, err = client.CallTool(ctx, "test_tool", nil)
		require.NoError(t, err)

		// Disconnect
		err = client.Disconnect(ctx)
		require.NoError(t, err)
		assert.False(t, client.IsConnected())
	})
}

// Mock client for testing
type MockClient struct {
	name        string
	config      *config.MCPClientConfig
	connected   bool
	initialized bool
}

func NewMockClient(name string) *MockClient {
	return &MockClient{
		name:        name,
		connected:   false,
		initialized: false,
	}
}

func (m *MockClient) Connect(ctx context.Context) error {
	if m.connected {
		return protocol.NewConnectionError("already connected", nil)
	}
	m.connected = true
	return nil
}

func (m *MockClient) Disconnect(ctx context.Context) error {
	m.connected = false
	m.initialized = false
	return nil
}

func (m *MockClient) Initialize(ctx context.Context) (*protocol.InitializeResult, error) {
	if !m.connected {
		return nil, protocol.NewConnectionError("not connected", nil)
	}
	m.initialized = true
	return &protocol.InitializeResult{
		ProtocolVersion: protocol.MCPProtocolVersion,
		ServerInfo: protocol.ServerInfo{
			Name:    "mock-server",
			Version: "1.0.0",
		},
		Capabilities: protocol.Capabilities{
			Tools: &protocol.ToolsCapability{
				ListChanged: true,
			},
		},
	}, nil
}

func (m *MockClient) ListTools(ctx context.Context, cursor string) (*protocol.ListToolsResult, error) {
	if !m.initialized {
		return nil, protocol.NewConnectionError("not initialized", nil)
	}
	return &protocol.ListToolsResult{
		Tools: []protocol.Tool{
			{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		},
	}, nil
}

func (m *MockClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.ToolCallResult, error) {
	if !m.initialized {
		return nil, protocol.NewConnectionError("not initialized", nil)
	}
	if name == "error_tool" {
		return &protocol.ToolCallResult{
			Content: []protocol.Content{
				{
					Type: "text",
					Text: "Error occurred",
				},
			},
			IsError: true,
		}, nil
	}
	return &protocol.ToolCallResult{
		Content: []protocol.Content{
			{
				Type: "text",
				Text: "Success",
			},
		},
		IsError: false,
	}, nil
}

func (m *MockClient) Ping(ctx context.Context) error {
	if !m.connected {
		return protocol.NewConnectionError("not connected", nil)
	}
	return nil
}

func (m *MockClient) IsConnected() bool {
	return m.connected
}

func (m *MockClient) GetConfig() *config.MCPClientConfig {
	return m.config
}

func (m *MockClient) GetName() string {
	return m.name
}

func (m *MockClient) SetConfig(cfg *config.MCPClientConfig) {
	m.config = cfg
}
