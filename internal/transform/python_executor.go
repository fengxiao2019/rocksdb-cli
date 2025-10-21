package transform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// pythonExecutor implements PythonExecutor interface using external Python process
type pythonExecutor struct {
	timeout        time.Duration
	pythonCommand  string
}

// NewPythonExecutor creates a new Python executor
func NewPythonExecutor() PythonExecutor {
	return &pythonExecutor{
		timeout:       30 * time.Second, // default timeout
		pythonCommand: "python3",        // default Python command
	}
}

// ExecuteExpression executes a Python expression with given context
func (e *pythonExecutor) ExecuteExpression(expr string, ctxMap map[string]interface{}) (interface{}, error) {
	if expr == "" {
		return nil, fmt.Errorf("expression cannot be empty")
	}

	// Build Python script with context
	script, err := e.buildScript(expr, ctxMap)
	if err != nil {
		return nil, fmt.Errorf("failed to build script: %w", err)
	}

	// Execute Python script with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.pythonCommand, "-c", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("execution timeout after %v", e.timeout)
		}
		// Return Python error
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return nil, fmt.Errorf("python error: %s", errMsg)
		}
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Parse and return result
	result := strings.TrimSpace(stdout.String())
	
	// If result looks like JSON, re-format it compactly
	if (strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}")) ||
		(strings.HasPrefix(result, "[") && strings.HasSuffix(result, "]")) {
		var jsonObj interface{}
		if err := json.Unmarshal([]byte(result), &jsonObj); err == nil {
			// Successfully parsed as JSON, re-encode compactly
			if compactJSON, err := json.Marshal(jsonObj); err == nil {
				return string(compactJSON), nil
			}
		}
	}
	
	return result, nil
}

// buildScript builds a complete Python script with context variables
func (e *pythonExecutor) buildScript(expr string, ctxMap map[string]interface{}) (string, error) {
	var script strings.Builder
	
	// Add standard imports that might be needed
	script.WriteString("import json\n")
	script.WriteString("import sys\n\n")
	
	// Set context variables
	for key, value := range ctxMap {
		// Convert value to Python representation
		pyValue, err := e.toPythonValue(value)
		if err != nil {
			return "", fmt.Errorf("failed to convert context variable %s: %w", key, err)
		}
		script.WriteString(fmt.Sprintf("%s = %s\n", key, pyValue))
	}
	
	script.WriteString("\n")
	
	// Check if expression is multiline (script) or single expression
	trimmedExpr := strings.TrimSpace(expr)
	if strings.Contains(trimmedExpr, "\n") {
		// Multi-line script - execute and print last expression
		lines := strings.Split(trimmedExpr, "\n")
		script.WriteString("# Multi-line script\n")
		
		// Write all lines except the last
		for i := 0; i < len(lines)-1; i++ {
			script.WriteString(lines[i] + "\n")
		}
		
		// For the last line, wrap it to capture and print the result
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if lastLine != "" {
			script.WriteString("_result = " + lastLine + "\n")
			script.WriteString("print(_result)\n")
		}
	} else {
		// Single expression - evaluate and print result
		script.WriteString("result = ")
		script.WriteString(trimmedExpr)
		script.WriteString("\nprint(result)\n")
	}
	
	return script.String(), nil
}

// toPythonValue converts Go value to Python representation
func (e *pythonExecutor) toPythonValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		// Escape string and wrap in quotes
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes), nil
	case int, int64, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		if v {
			return "True", nil
		}
		return "False", nil
	case nil:
		return "None", nil
	default:
		// Try JSON encoding for complex types
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	}
}

// ExecuteScript executes a Python script file
func (e *pythonExecutor) ExecuteScript(scriptPath string, key string, value string) (string, string, error) {
	// TODO: Implement Python script file execution
	return "", "", fmt.Errorf("not implemented: ExecuteScript")
}

// ValidateExpression validates Python expression syntax
func (e *pythonExecutor) ValidateExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("expression cannot be empty")
	}

	// Determine mode based on expression type
	mode := "eval"
	if strings.Contains(strings.TrimSpace(expr), "\n") {
		mode = "exec" // Multi-line scripts need exec mode
	}

	// Try to compile the expression using Python's compile() function
	script := fmt.Sprintf(`
import sys
try:
    compile(%s, '<string>', '%s')
    sys.exit(0)
except SyntaxError as e:
    print(f"Syntax error: {e}", file=sys.stderr)
    sys.exit(1)
`, e.quotePythonString(expr), mode)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.pythonCommand, "-c", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("invalid expression: %s", errMsg)
		}
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// quotePythonString safely quotes a string for Python
func (e *pythonExecutor) quotePythonString(s string) string {
	jsonBytes, _ := json.Marshal(s)
	return string(jsonBytes)
}

// SetTimeout sets execution timeout
func (e *pythonExecutor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}
