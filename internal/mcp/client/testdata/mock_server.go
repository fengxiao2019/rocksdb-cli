// +build ignore

package main

import (
	"bufio"
	"encoding/json"
	"os"
)

type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONError  `json:"error,omitempty"`
}

type JSONError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		var resp Response
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "test-server",
					"version": "1.0.0",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]bool{
						"listChanged": true,
					},
				},
			}

		case "ping":
			resp.Result = map[string]interface{}{}

		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "echo",
						"description": "Echo tool",
						"inputSchema": map[string]string{
							"type": "object",
						},
					},
				},
			}

		case "tools/call":
			resp.Result = map[string]interface{}{
				"content": []map[string]string{
					{
						"type": "text",
						"text": "success",
					},
				},
				"isError": false,
			}

		default:
			resp.Error = &JSONError{
				Code:    -32601,
				Message: "Method not found",
			}
		}

		encoder.Encode(resp)
	}
}
