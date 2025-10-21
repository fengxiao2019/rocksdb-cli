package transform

import "time"

// TransformOptions defines options for the transform operation
type TransformOptions struct {
	// Expression is a Python expression to transform value
	Expression string
	
	// KeyExpression is a Python expression to transform key (optional)
	KeyExpression string
	
	// ValueExpression is a Python expression to transform value (explicit)
	ValueExpression string
	
	// FilterExpression is a Python expression that returns bool to filter entries
	FilterExpression string
	
	// ScriptPath is the path to a Python script file
	ScriptPath string
	
	// DryRun mode - preview changes without writing to database
	DryRun bool
	
	// Limit maximum number of entries to process (0 = no limit)
	Limit int
	
	// BatchSize for batch processing (default: 1000)
	BatchSize int
	
	// Verbose mode - show detailed processing information
	Verbose bool
	
	// Backup file path (optional)
	BackupPath string
	
	// Timeout for each expression execution
	Timeout time.Duration
}

// TransformResult contains the results of a transform operation
type TransformResult struct {
	// Processed is the total number of entries processed
	Processed int
	
	// Modified is the number of entries actually modified
	Modified int
	
	// Skipped is the number of entries skipped (filtered out)
	Skipped int
	
	// Errors is a list of errors encountered during processing
	Errors []TransformError
	
	// DryRunData contains preview data for dry-run mode
	DryRunData []DryRunEntry
	
	// Duration is the total processing time
	Duration time.Duration
	
	// StartTime is when processing started
	StartTime time.Time
	
	// EndTime is when processing finished
	EndTime time.Time
}

// TransformError represents an error that occurred during transformation
type TransformError struct {
	// Key is the key being processed when error occurred
	Key string
	
	// OriginalValue is the original value
	OriginalValue string
	
	// Error is the error message
	Error string
	
	// Timestamp when error occurred
	Timestamp time.Time
}

// DryRunEntry represents a single entry in dry-run preview
type DryRunEntry struct {
	// OriginalKey is the key before transformation
	OriginalKey string
	
	// TransformedKey is the key after transformation (if key transform applied)
	TransformedKey string
	
	// OriginalValue is the value before transformation
	OriginalValue string
	
	// TransformedValue is the value after transformation
	TransformedValue string
	
	// WillModify indicates if this entry will be modified
	WillModify bool
	
	// Skipped indicates if this entry was filtered out
	Skipped bool
}

// PythonExecutor defines the interface for executing Python code
type PythonExecutor interface {
	// ExecuteExpression executes a Python expression with given context
	ExecuteExpression(expr string, context map[string]interface{}) (interface{}, error)
	
	// ExecuteScript executes a Python script file
	ExecuteScript(scriptPath string, key string, value string) (string, string, error)
	
	// ValidateExpression validates Python expression syntax
	ValidateExpression(expr string) error
	
	// SetTimeout sets execution timeout
	SetTimeout(timeout time.Duration)
}

// TransformProcessor defines the interface for processing transformations
type TransformProcessor interface {
	// Process executes the transformation on a column family
	Process(cf string, opts TransformOptions) (*TransformResult, error)
	
	// ProcessWithCallback executes transformation with progress callback
	ProcessWithCallback(cf string, opts TransformOptions, callback ProgressCallback) (*TransformResult, error)
}

// ProgressCallback is called periodically during processing
type ProgressCallback func(processed int, total int, current DryRunEntry)

// ScriptDefinition defines the structure of a transform script file
type ScriptDefinition struct {
	// TransformKey function (optional)
	TransformKey func(key string) string
	
	// TransformValue function (required)
	TransformValue func(key string, value string) string
	
	// ShouldProcess function (optional, default: always true)
	ShouldProcess func(key string, value string) bool
	
	// OnError function (optional, default: log and continue)
	OnError func(key string, value string, err error) bool
}
