package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"
)

// TCPClient implements an MCP client using TCP transport
type TCPClient struct {
	*BaseClient

	// Connection management
	conn     net.Conn
	connMu   sync.Mutex
	dialAddr string

	// Communication
	encoder *json.Encoder
	decoder *json.Decoder

	// Request tracking
	requestID  atomic.Int64
	responseCh map[int64]chan *protocol.JSONRPCResponse
	responseMu sync.RWMutex

	// Cleanup
	done chan struct{}
}

// NewTCPClient creates a new TCP-based MCP client
func NewTCPClient(name string, cfg *config.MCPClientConfig) *TCPClient {
	return &TCPClient{
		BaseClient: NewBaseClient(name, cfg),
		dialAddr:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		responseCh: make(map[int64]chan *protocol.JSONRPCResponse),
		done:       make(chan struct{}),
	}
}

// Connect establishes a TCP connection to the MCP server
func (tc *TCPClient) Connect(ctx context.Context) error {
	if tc.IsConnected() {
		return protocol.NewConnectionError("already connected", nil)
	}

	cfg := tc.GetConfig()

	// Establish TCP connection with timeout
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	tc.connMu.Lock()
	defer tc.connMu.Unlock()

	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", tc.dialAddr)
	if err != nil {
		return protocol.NewConnectionError("failed to connect", map[string]string{
			"error":   err.Error(),
			"address": tc.dialAddr,
		})
	}

	tc.conn = conn

	// Setup encoder/decoder
	tc.encoder = json.NewEncoder(tc.conn)
	tc.decoder = json.NewDecoder(tc.conn)

	// Start response reader
	go tc.readResponses()

	tc.SetConnected(true)
	return nil
}

// Disconnect closes the TCP connection and cleans up
func (tc *TCPClient) Disconnect(ctx context.Context) error {
	if !tc.IsConnected() {
		return nil
	}

	tc.SetConnected(false)
	tc.SetInitialized(false)

	tc.connMu.Lock()
	if tc.conn != nil {
		tc.conn.Close()
		tc.conn = nil
	}
	tc.connMu.Unlock()

	// Clear response channels
	tc.responseMu.Lock()
	for _, ch := range tc.responseCh {
		close(ch)
	}
	tc.responseCh = make(map[int64]chan *protocol.JSONRPCResponse)
	tc.responseMu.Unlock()

	return nil
}

// Initialize sends the initialize request
func (tc *TCPClient) Initialize(ctx context.Context) (*protocol.InitializeResult, error) {
	if err := tc.CheckConnected(); err != nil {
		return nil, err
	}

	cfg := tc.GetConfig()
	req := protocol.InitializeRequest{
		ProtocolVersion: protocol.MCPProtocolVersion,
		ClientInfo: protocol.ClientInfo{
			Name:    cfg.Name,
			Version: "1.0.0",
		},
		Capabilities: protocol.Capabilities{},
	}

	var result protocol.InitializeResult
	if err := tc.sendRequest(ctx, protocol.MethodInitialize, req, &result); err != nil {
		return nil, err
	}

	tc.SetInitialized(true)
	tc.SetServerInfo(&result.ServerInfo)

	return &result, nil
}

// Ping sends a ping request
func (tc *TCPClient) Ping(ctx context.Context) error {
	if err := tc.CheckConnected(); err != nil {
		return err
	}

	var result interface{}
	return tc.sendRequest(ctx, protocol.MethodPing, nil, &result)
}

// ListTools lists available tools
func (tc *TCPClient) ListTools(ctx context.Context, cursor string) (*protocol.ListToolsResult, error) {
	if err := tc.CheckInitialized(); err != nil {
		return nil, err
	}

	req := protocol.ListToolsRequest{
		Cursor: cursor,
	}

	var result protocol.ListToolsResult
	if err := tc.sendRequest(ctx, protocol.MethodListTools, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CallTool calls a tool with given arguments
func (tc *TCPClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.ToolCallResult, error) {
	if err := tc.CheckInitialized(); err != nil {
		return nil, err
	}

	req := protocol.CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}

	var result protocol.ToolCallResult
	if err := tc.sendRequest(ctx, protocol.MethodCallTool, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// sendRequest sends a JSON-RPC request and waits for response
func (tc *TCPClient) sendRequest(ctx context.Context, method string, params interface{}, result interface{}) error {
	// Generate request ID
	id := tc.requestID.Add(1)

	// Create response channel
	respCh := make(chan *protocol.JSONRPCResponse, 1)
	tc.responseMu.Lock()
	tc.responseCh[id] = respCh
	tc.responseMu.Unlock()

	// Cleanup channel when done
	defer func() {
		tc.responseMu.Lock()
		delete(tc.responseCh, id)
		tc.responseMu.Unlock()
		close(respCh)
	}()

	// Create request
	request := protocol.JSONRPCRequest{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Send request
	tc.connMu.Lock()
	if tc.conn == nil {
		tc.connMu.Unlock()
		return protocol.NewConnectionError("connection closed", nil)
	}
	if err := tc.encoder.Encode(request); err != nil {
		tc.connMu.Unlock()
		return protocol.NewConnectionError("failed to send request", map[string]string{
			"error":  err.Error(),
			"method": method,
		})
	}
	tc.connMu.Unlock()

	// Wait for response with timeout
	cfg := tc.GetConfig()
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case resp := <-respCh:
		if resp == nil {
			return protocol.NewConnectionError("connection closed", nil)
		}

		// Check for error
		if resp.Error != nil {
			return &protocol.MCPError{
				Code:    resp.Error.Code,
				Message: resp.Error.Message,
				Data:    resp.Error.Data,
			}
		}

		// Parse result
		if result != nil {
			// Re-marshal and unmarshal to convert interface{} to typed struct
			data, err := json.Marshal(resp.Result)
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}

			if err := json.Unmarshal(data, result); err != nil {
				return fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}

		return nil

	case <-ctxWithTimeout.Done():
		return protocol.NewConnectionTimeoutError("request timeout")

	case <-tc.done:
		return protocol.NewConnectionError("connection closed", nil)
	}
}

// readResponses reads responses from the TCP connection
func (tc *TCPClient) readResponses() {
	defer close(tc.done)

	for {
		var resp protocol.JSONRPCResponse
		if err := tc.decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				return
			}
			// Connection error, close all pending requests
			tc.responseMu.RLock()
			for _, ch := range tc.responseCh {
				select {
				case ch <- nil:
				default:
				}
			}
			tc.responseMu.RUnlock()
			return
		}

		// Route response to waiting request
		if resp.ID != nil {
			// Convert ID to int64
			var id int64
			switch v := resp.ID.(type) {
			case float64:
				id = int64(v)
			case int:
				id = int64(v)
			case int64:
				id = v
			default:
				continue
			}

			tc.responseMu.RLock()
			ch, ok := tc.responseCh[id]
			tc.responseMu.RUnlock()

			if ok {
				select {
				case ch <- &resp:
				default:
				}
			}
		}
	}
}
