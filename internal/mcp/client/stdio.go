package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"rocksdb-cli/internal/config"
	"rocksdb-cli/internal/mcp/protocol"
)

// StdioClient implements an MCP client using STDIO transport
type StdioClient struct {
	*BaseClient

	// Process management
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// Communication
	encoder *json.Encoder
	decoder *json.Decoder

	// Request tracking
	requestID  atomic.Int64
	responseCh map[int64]chan *protocol.JSONRPCResponse
	responseMu sync.RWMutex

	// Cleanup
	cancel context.CancelFunc
	done   chan struct{}
}

// NewStdioClient creates a new STDIO-based MCP client
func NewStdioClient(name string, cfg *config.MCPClientConfig) *StdioClient {
	return &StdioClient{
		BaseClient: NewBaseClient(name, cfg),
		responseCh: make(map[int64]chan *protocol.JSONRPCResponse),
		done:       make(chan struct{}),
	}
}

// Connect starts the STDIO process and establishes communication
func (sc *StdioClient) Connect(ctx context.Context) error {
	if sc.IsConnected() {
		return protocol.NewConnectionError("already connected", nil)
	}

	cfg := sc.GetConfig()

	// Create command
	cmdCtx, cancel := context.WithCancel(context.Background())
	sc.cancel = cancel

	sc.cmd = exec.CommandContext(cmdCtx, cfg.Command, cfg.Args...)

	// Set environment variables
	if len(cfg.Env) > 0 {
		env := os.Environ()
		for key, value := range cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		sc.cmd.Env = env
	}

	// Setup pipes
	stdin, err := sc.cmd.StdinPipe()
	if err != nil {
		cancel()
		return protocol.NewConnectionError("failed to create stdin pipe", map[string]string{
			"error": err.Error(),
		})
	}
	sc.stdin = stdin

	stdout, err := sc.cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		cancel()
		return protocol.NewConnectionError("failed to create stdout pipe", map[string]string{
			"error": err.Error(),
		})
	}
	sc.stdout = stdout

	stderr, err := sc.cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		cancel()
		return protocol.NewConnectionError("failed to create stderr pipe", map[string]string{
			"error": err.Error(),
		})
	}
	sc.stderr = stderr

	// Start the process
	if err := sc.cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		cancel()
		return protocol.NewConnectionError("failed to start process", map[string]string{
			"error":   err.Error(),
			"command": cfg.Command,
		})
	}

	// Setup encoder/decoder
	sc.encoder = json.NewEncoder(sc.stdin)
	sc.decoder = json.NewDecoder(bufio.NewReader(sc.stdout))

	// Start response reader
	go sc.readResponses()

	// Start stderr reader
	go sc.readStderr()

	// Monitor process exit
	go sc.monitorProcess()

	sc.SetConnected(true)
	return nil
}

// Disconnect stops the STDIO process and cleans up
func (sc *StdioClient) Disconnect(ctx context.Context) error {
	if !sc.IsConnected() {
		return nil
	}

	sc.SetConnected(false)
	sc.SetInitialized(false)

	// Cancel context to signal shutdown
	if sc.cancel != nil {
		sc.cancel()
	}

	// Close stdin to signal process to exit
	if sc.stdin != nil {
		sc.stdin.Close()
	}

	// Wait for process to exit (with timeout)
	if sc.cmd != nil && sc.cmd.Process != nil {
		done := make(chan error, 1)
		go func() {
			done <- sc.cmd.Wait()
		}()

		// Give process 1 second to exit gracefully
		timeout := time.NewTimer(1 * time.Second)
		defer timeout.Stop()

		select {
		case <-timeout.C:
			// Timeout - force kill
			if sc.cmd.Process != nil {
				sc.cmd.Process.Kill()
			}
			// Give it 500ms to die after kill
			select {
			case <-done:
				// Killed successfully
			case <-time.After(500 * time.Millisecond):
				// Process still won't die, give up
			}
		case <-ctx.Done():
			// Context cancelled - force kill
			if sc.cmd.Process != nil {
				sc.cmd.Process.Kill()
			}
			// Brief wait after kill
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
			}
		case <-done:
			// Process exited normally
		}
	}

	// Close remaining pipes
	if sc.stdout != nil {
		sc.stdout.Close()
	}
	if sc.stderr != nil {
		sc.stderr.Close()
	}

	// Clear response channels
	sc.responseMu.Lock()
	for _, ch := range sc.responseCh {
		close(ch)
	}
	sc.responseCh = make(map[int64]chan *protocol.JSONRPCResponse)
	sc.responseMu.Unlock()

	// Note: We don't wait for sc.done here as it could cause deadlock
	// The goroutines will cleanup themselves when pipes are closed

	return nil
}

// Initialize sends the initialize request
func (sc *StdioClient) Initialize(ctx context.Context) (*protocol.InitializeResult, error) {
	if err := sc.CheckConnected(); err != nil {
		return nil, err
	}

	cfg := sc.GetConfig()
	req := protocol.InitializeRequest{
		ProtocolVersion: protocol.MCPProtocolVersion,
		ClientInfo: protocol.ClientInfo{
			Name:    cfg.Name,
			Version: "1.0.0",
		},
		Capabilities: protocol.Capabilities{},
	}

	var result protocol.InitializeResult
	if err := sc.sendRequest(ctx, protocol.MethodInitialize, req, &result); err != nil {
		return nil, err
	}

	sc.SetInitialized(true)
	sc.SetServerInfo(&result.ServerInfo)

	return &result, nil
}

// Ping sends a ping request
func (sc *StdioClient) Ping(ctx context.Context) error {
	if err := sc.CheckConnected(); err != nil {
		return err
	}

	var result interface{}
	return sc.sendRequest(ctx, protocol.MethodPing, nil, &result)
}

// ListTools lists available tools
func (sc *StdioClient) ListTools(ctx context.Context, cursor string) (*protocol.ListToolsResult, error) {
	if err := sc.CheckInitialized(); err != nil {
		return nil, err
	}

	req := protocol.ListToolsRequest{
		Cursor: cursor,
	}

	var result protocol.ListToolsResult
	if err := sc.sendRequest(ctx, protocol.MethodListTools, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CallTool calls a tool with given arguments
func (sc *StdioClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*protocol.ToolCallResult, error) {
	if err := sc.CheckInitialized(); err != nil {
		return nil, err
	}

	req := protocol.CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}

	var result protocol.ToolCallResult
	if err := sc.sendRequest(ctx, protocol.MethodCallTool, req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// sendRequest sends a JSON-RPC request and waits for response
func (sc *StdioClient) sendRequest(ctx context.Context, method string, params interface{}, result interface{}) error {
	// Generate request ID
	id := sc.requestID.Add(1)

	// Create response channel
	respCh := make(chan *protocol.JSONRPCResponse, 1)
	sc.responseMu.Lock()
	sc.responseCh[id] = respCh
	sc.responseMu.Unlock()

	// Cleanup channel when done
	defer func() {
		sc.responseMu.Lock()
		delete(sc.responseCh, id)
		sc.responseMu.Unlock()
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
	if err := sc.encoder.Encode(request); err != nil {
		return protocol.NewConnectionError("failed to send request", map[string]string{
			"error":  err.Error(),
			"method": method,
		})
	}

	// Wait for response with timeout
	cfg := sc.GetConfig()
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * 1e9 // 30 seconds default
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

	case <-sc.done:
		return protocol.NewConnectionError("connection closed", nil)
	}
}

// readResponses reads responses from stdout
func (sc *StdioClient) readResponses() {
	defer close(sc.done)

	for {
		var resp protocol.JSONRPCResponse
		if err := sc.decoder.Decode(&resp); err != nil {
			if err == io.EOF {
				return
			}
			// Connection error, close all pending requests
			sc.responseMu.RLock()
			for _, ch := range sc.responseCh {
				select {
				case ch <- nil:
				default:
				}
			}
			sc.responseMu.RUnlock()
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

			sc.responseMu.RLock()
			ch, ok := sc.responseCh[id]
			sc.responseMu.RUnlock()

			if ok {
				select {
				case ch <- &resp:
				default:
				}
			}
		}
	}
}

// readStderr reads and logs stderr output
func (sc *StdioClient) readStderr() {
	scanner := bufio.NewScanner(sc.stderr)
	for scanner.Scan() {
		// Log stderr output (in production, use proper logger)
		// For now, we just consume it to prevent blocking
		_ = scanner.Text()
	}
}

// monitorProcess monitors the process and handles unexpected exits
func (sc *StdioClient) monitorProcess() {
	if sc.cmd == nil || sc.cmd.Process == nil {
		return
	}

	sc.cmd.Wait()

	// Process exited, clean up
	if sc.IsConnected() {
		sc.SetConnected(false)
		sc.SetInitialized(false)
	}
}
