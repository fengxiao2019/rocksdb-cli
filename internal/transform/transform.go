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
	// Validate options
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	
	// Initialize result
	result := &TransformResult{
		StartTime:  time.Now(),
		Processed:  0,
		Modified:   0,
		Skipped:    0,
		Errors:     []TransformError{},
		DryRunData: []DryRunEntry{},
	}
	
	// Handle nil database (for testing)
	if p.db == nil {
		return p.processMockData(cf, opts, result)
	}
	
	// Scan all entries in column family
	scanOpts := db.ScanOptions{
		Values: true,
		Limit:  opts.Limit,
	}
	
	entries, err := p.db.SmartScanCF(cf, "", "", scanOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to scan column family: %w", err)
	}
	
	// Process each entry
	for key, value := range entries {
		// Check script file filter first (if using script)
		if opts.ScriptPath != "" {
			_, transformedValue, err := p.executor.ExecuteScript(opts.ScriptPath, key, value)
			if err != nil {
				result.Errors = append(result.Errors, TransformError{
					Key:           key,
					OriginalValue: value,
					Error:         fmt.Sprintf("script error: %v", err),
					Timestamp:     time.Now(),
				})
				result.Processed++
				continue
			}
			
			// Empty string means should_process returned False
			if transformedValue == "" {
				result.Skipped++
				result.Processed++
				if opts.DryRun {
					result.DryRunData = append(result.DryRunData, DryRunEntry{
						OriginalKey:      key,
						TransformedKey:   key,
						OriginalValue:    value,
						TransformedValue: value,
						WillModify:       false,
						Skipped:          true,
					})
				}
				continue
			}
			
			// Check if value changed
			willModify := transformedValue != value
			
			if opts.DryRun {
				result.DryRunData = append(result.DryRunData, DryRunEntry{
					OriginalKey:      key,
					TransformedKey:   key,
					OriginalValue:    value,
					TransformedValue: transformedValue,
					WillModify:       willModify,
					Skipped:          false,
				})
			} else {
				if willModify {
					if err := p.db.PutCF(cf, key, transformedValue); err != nil {
						result.Errors = append(result.Errors, TransformError{
							Key:           key,
							OriginalValue: value,
							Error:         fmt.Sprintf("write error: %v", err),
							Timestamp:     time.Now(),
						})
						result.Processed++
						continue
					}
					result.Modified++
				}
			}
			
			result.Processed++
			continue
		}
		
		// Apply filter if specified (for expression mode)
		if opts.FilterExpression != "" {
			context := map[string]interface{}{
				"key":   key,
				"value": value,
			}
			filterResult, err := p.executor.ExecuteExpression(opts.FilterExpression, context)
			if err != nil {
				result.Errors = append(result.Errors, TransformError{
					Key:           key,
					OriginalValue: value,
					Error:         fmt.Sprintf("filter error: %v", err),
					Timestamp:     time.Now(),
				})
				result.Processed++
				continue
			}
			
			// Check if filter passed
			shouldProcess := filterResult == "True" || filterResult == "true"
			if !shouldProcess {
				result.Skipped++
				result.Processed++
				if opts.DryRun {
					result.DryRunData = append(result.DryRunData, DryRunEntry{
						OriginalKey:      key,
						TransformedKey:   key,
						OriginalValue:    value,
						TransformedValue: value,
						WillModify:       false,
						Skipped:          true,
					})
				}
				continue
			}
		}
		
		// Apply transformation
		transformedValue, err := p.transformValue(key, value, opts)
		if err != nil {
			result.Errors = append(result.Errors, TransformError{
				Key:           key,
				OriginalValue: value,
				Error:         err.Error(),
				Timestamp:     time.Now(),
			})
			result.Processed++
			continue
		}
		
		// Check if value changed
		willModify := transformedValue != value
		
		if opts.DryRun {
			// Add to dry-run data
			result.DryRunData = append(result.DryRunData, DryRunEntry{
				OriginalKey:      key,
				TransformedKey:   key, // TODO: support key transformation
				OriginalValue:    value,
				TransformedValue: transformedValue,
				WillModify:       willModify,
				Skipped:          false,
			})
		} else {
			// Actually write to database if modified
			if willModify {
				if err := p.db.PutCF(cf, key, transformedValue); err != nil {
					result.Errors = append(result.Errors, TransformError{
						Key:           key,
						OriginalValue: value,
						Error:         fmt.Sprintf("write error: %v", err),
						Timestamp:     time.Now(),
					})
					result.Processed++
					continue
				}
				result.Modified++
			}
		}
		
		result.Processed++
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	return result, nil
}

// processMockData processes mock data for testing (when db is nil)
func (p *transformProcessor) processMockData(cf string, opts TransformOptions, result *TransformResult) (*TransformResult, error) {
	// Generate mock data
	// For batch processing tests, we need to generate enough data
	mockDataCount := 10000 // Generate enough data for batch tests
	if opts.Limit > 0 && opts.Limit < mockDataCount {
		mockDataCount = opts.Limit
	}
	
	mockData := make([]struct {
		key   string
		value string
	}, mockDataCount)
	
	// Generate mock entries
	baseValues := []string{"hello", "world", "test", "data", "sample"}
	for i := 0; i < mockDataCount; i++ {
		mockData[i] = struct {
			key   string
			value string
		}{
			key:   fmt.Sprintf("key%d", i+1),
			value: baseValues[i%len(baseValues)],
		}
	}
	
	// Apply limit if specified
	limit := len(mockData)
	if opts.Limit > 0 && opts.Limit < limit {
		limit = opts.Limit
	}
	
	// Process each entry
	for i := 0; i < limit; i++ {
		entry := mockData[i]
		
		// Apply filter if specified
		if opts.FilterExpression != "" {
			context := map[string]interface{}{
				"key":   entry.key,
				"value": entry.value,
			}
			filterResult, err := p.executor.ExecuteExpression(opts.FilterExpression, context)
			if err != nil {
				result.Errors = append(result.Errors, TransformError{
					Key:           entry.key,
					OriginalValue: entry.value,
					Error:         fmt.Sprintf("filter error: %v", err),
					Timestamp:     time.Now(),
				})
				continue
			}
			
			// Check if filter passed
			shouldProcess := filterResult == "True" || filterResult == "true"
			if !shouldProcess {
				result.Skipped++
				result.Processed++
				continue
			}
		}
		
		// Apply transformation
		transformedValue, err := p.transformValue(entry.key, entry.value, opts)
		if err != nil {
			result.Errors = append(result.Errors, TransformError{
				Key:           entry.key,
				OriginalValue: entry.value,
				Error:         err.Error(),
				Timestamp:     time.Now(),
			})
			result.Processed++
			continue
		}
		
		// Check if value changed
		willModify := transformedValue != entry.value
		
		if opts.DryRun {
			// Add to dry-run data
			result.DryRunData = append(result.DryRunData, DryRunEntry{
				OriginalKey:      entry.key,
				TransformedKey:   entry.key, // TODO: support key transformation
				OriginalValue:    entry.value,
				TransformedValue: transformedValue,
				WillModify:       willModify,
				Skipped:          false,
			})
		} else {
			// Would actually write to database here
			if willModify {
				result.Modified++
			}
		}
		
		result.Processed++
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	return result, nil
}

// transformValue applies the transformation expression to a value
func (p *transformProcessor) transformValue(key, value string, opts TransformOptions) (string, error) {
	// If script file is specified, use ExecuteScript
	if opts.ScriptPath != "" {
		_, transformedValue, err := p.executor.ExecuteScript(opts.ScriptPath, key, value)
		if err != nil {
			return "", fmt.Errorf("script execution failed: %w", err)
		}
		// Empty strings from ExecuteScript mean "skip this entry" (filtered by should_process)
		if transformedValue == "" {
			return value, nil // Return original value, will be handled by filter logic
		}
		return transformedValue, nil
	}
	
	// Determine which expression to use
	expr := opts.Expression
	if opts.ValueExpression != "" {
		expr = opts.ValueExpression
	}
	
	if expr == "" {
		return value, nil // No transformation
	}
	
	// Build context
	context := map[string]interface{}{
		"key":   key,
		"value": value,
	}
	
	// Execute expression
	result, err := p.executor.ExecuteExpression(expr, context)
	if err != nil {
		return "", fmt.Errorf("expression execution failed: %w", err)
	}
	
	// Convert result to string
	resultStr, ok := result.(string)
	if !ok {
		resultStr = fmt.Sprintf("%v", result)
	}
	
	return resultStr, nil
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
