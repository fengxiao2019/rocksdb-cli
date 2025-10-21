package transform

import (
	"fmt"
	"time"
	
	"rocksdb-cli/internal/db"
)

// transformProcessor implements TransformProcessor interface
type transformProcessor struct {
	db       db.KeyValueDB
	executor PythonExecutor
}

// NewTransformProcessor creates a new transform processor
func NewTransformProcessor(database db.KeyValueDB) TransformProcessor {
	return &transformProcessor{
		db:       database,
		executor: NewPythonExecutor(),
	}
}

// Process executes the transformation on a column family
func (p *transformProcessor) Process(cf string, opts TransformOptions) (*TransformResult, error) {
	// TODO: Implement transformation processing
	// For now, return basic structure to make tests compile
	result := &TransformResult{
		StartTime:  time.Now(),
		Processed:  0,
		Modified:   0,
		Skipped:    0,
		Errors:     []TransformError{},
		DryRunData: []DryRunEntry{},
	}
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	// Return error to make tests fail (TDD approach)
	return result, fmt.Errorf("not implemented: Process")
}

// ProcessWithCallback executes transformation with progress callback
func (p *transformProcessor) ProcessWithCallback(cf string, opts TransformOptions, callback ProgressCallback) (*TransformResult, error) {
	// TODO: Implement with callback support
	return nil, fmt.Errorf("not implemented: ProcessWithCallback")
}

// validateOptions validates transform options
func validateOptions(opts TransformOptions) error {
	// Must have at least one transformation expression or script
	if opts.Expression == "" && opts.ValueExpression == "" && opts.ScriptPath == "" {
		return fmt.Errorf("must specify at least one of: Expression, ValueExpression, or ScriptPath")
	}
	
	// BatchSize must be positive if specified
	if opts.BatchSize < 0 {
		return fmt.Errorf("BatchSize must be non-negative")
	}
	
	// Limit must be non-negative
	if opts.Limit < 0 {
		return fmt.Errorf("Limit must be non-negative")
	}
	
	return nil
}
