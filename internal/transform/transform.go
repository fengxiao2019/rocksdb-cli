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
// This is a convenience wrapper around ProcessWithCallback with a nil callback
func (p *transformProcessor) Process(cf string, opts TransformOptions) (*TransformResult, error) {
	return p.ProcessWithCallback(cf, opts, nil)
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
		return p.processMockDataWithCallback(cf, opts, result, callback)
	}

	// Scan all entries in column family to get total count first
	scanOpts := db.ScanOptions{
		Values: true,
		Limit:  opts.Limit,
	}

	entries, err := p.db.SmartScanCF(cf, "", "", scanOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to scan column family: %w", err)
	}

	// Calculate total count
	total := len(entries)

	// Process each entry
	for key, value := range entries {
		// Handle script file mode
		if opts.ScriptPath != "" {
			processedCount := result.Processed + 1
			entry := p.processScriptEntry(key, value, cf, opts, result)

			// Invoke callback
			if callback != nil {
				callback(processedCount, total, entry)
			}

			result.Processed = processedCount
			continue
		}

		// Apply filter if specified (for expression mode)
		if opts.FilterExpression != "" {
			shouldProcess, filterErr := p.applyFilter(key, value, opts)
			if filterErr != nil {
				result.Errors = append(result.Errors, TransformError{
					Key:           key,
					OriginalValue: value,
					Error:         fmt.Sprintf("filter error: %v", filterErr),
					Timestamp:     time.Now(),
				})
				result.Processed++

				// Still invoke callback for error case
				if callback != nil {
					entry := DryRunEntry{
						OriginalKey:      key,
						TransformedKey:   key,
						OriginalValue:    value,
						TransformedValue: value,
						WillModify:       false,
						Skipped:          false,
					}
					callback(result.Processed, total, entry)
				}
				continue
			}

			if !shouldProcess {
				result.Skipped++
				result.Processed++

				entry := DryRunEntry{
					OriginalKey:      key,
					TransformedKey:   key,
					OriginalValue:    value,
					TransformedValue: value,
					WillModify:       false,
					Skipped:          true,
				}

				if opts.DryRun {
					result.DryRunData = append(result.DryRunData, entry)
				}

				// Invoke callback for skipped entry
				if callback != nil {
					callback(result.Processed, total, entry)
				}
				continue
			}
		}

		// Apply key transformation
		transformedKey, err := p.transformKey(key, value, opts)
		if err != nil {
			result.Errors = append(result.Errors, TransformError{
				Key:           key,
				OriginalValue: value,
				Error:         err.Error(),
				Timestamp:     time.Now(),
			})
			result.Processed++

			// Invoke callback for error case
			if callback != nil {
				entry := DryRunEntry{
					OriginalKey:      key,
					TransformedKey:   key,
					OriginalValue:    value,
					TransformedValue: value,
					WillModify:       false,
					Skipped:          false,
				}
				callback(result.Processed, total, entry)
			}
			continue
		}

		// Apply value transformation
		transformedValue, err := p.transformValue(key, value, opts)
		if err != nil {
			result.Errors = append(result.Errors, TransformError{
				Key:           key,
				OriginalValue: value,
				Error:         err.Error(),
				Timestamp:     time.Now(),
			})
			result.Processed++

			// Invoke callback for error case
			if callback != nil {
				entry := DryRunEntry{
					OriginalKey:      key,
					TransformedKey:   transformedKey,
					OriginalValue:    value,
					TransformedValue: value,
					WillModify:       false,
					Skipped:          false,
				}
				callback(result.Processed, total, entry)
			}
			continue
		}

		// Check if key or value changed
		keyChanged := transformedKey != key
		valueChanged := transformedValue != value
		willModify := keyChanged || valueChanged

		entry := DryRunEntry{
			OriginalKey:      key,
			TransformedKey:   transformedKey,
			OriginalValue:    value,
			TransformedValue: transformedValue,
			WillModify:       willModify,
			Skipped:          false,
		}

		if opts.DryRun {
			// Add to dry-run data
			result.DryRunData = append(result.DryRunData, entry)
		} else {
			// Actually write to database if modified
			if willModify {
				if keyChanged {
					if err := p.db.PutCF(cf, transformedKey, transformedValue); err != nil {
						result.Errors = append(result.Errors, TransformError{
							Key:           key,
							OriginalValue: value,
							Error:         fmt.Sprintf("write error: %v", err),
							Timestamp:     time.Now(),
						})
						result.Processed++

						// Invoke callback for error case
						if callback != nil {
							callback(result.Processed, total, entry)
						}
						continue
					}
				} else {
					if err := p.db.PutCF(cf, key, transformedValue); err != nil {
						result.Errors = append(result.Errors, TransformError{
							Key:           key,
							OriginalValue: value,
							Error:         fmt.Sprintf("write error: %v", err),
							Timestamp:     time.Now(),
						})
						result.Processed++

						// Invoke callback for error case
						if callback != nil {
							callback(result.Processed, total, entry)
						}
						continue
					}
				}
				result.Modified++
			}
		}

		result.Processed++

		// Invoke callback after processing each entry
		if callback != nil {
			callback(result.Processed, total, entry)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// applyFilter checks if an entry should be processed based on filter expression
func (p *transformProcessor) applyFilter(key, value string, opts TransformOptions) (bool, error) {
	context := map[string]interface{}{
		"key":   key,
		"value": value,
	}
	filterResult, err := p.executor.ExecuteExpression(opts.FilterExpression, context)
	if err != nil {
		return false, err
	}

	// Check if filter passed
	return filterResult == "True" || filterResult == "true", nil
}

// processScriptEntry processes a single entry using script file
func (p *transformProcessor) processScriptEntry(key, value, cf string, opts TransformOptions, result *TransformResult) DryRunEntry {
	_, transformedValue, err := p.executor.ExecuteScript(opts.ScriptPath, key, value)
	if err != nil {
		result.Errors = append(result.Errors, TransformError{
			Key:           key,
			OriginalValue: value,
			Error:         fmt.Sprintf("script error: %v", err),
			Timestamp:     time.Now(),
		})
		return DryRunEntry{
			OriginalKey:      key,
			TransformedKey:   key,
			OriginalValue:    value,
			TransformedValue: value,
			WillModify:       false,
			Skipped:          false,
		}
	}

	// Empty string means should_process returned False
	if transformedValue == "" {
		result.Skipped++
		entry := DryRunEntry{
			OriginalKey:      key,
			TransformedKey:   key,
			OriginalValue:    value,
			TransformedValue: value,
			WillModify:       false,
			Skipped:          true,
		}
		if opts.DryRun {
			result.DryRunData = append(result.DryRunData, entry)
		}
		return entry
	}

	// Check if value changed
	willModify := transformedValue != value
	entry := DryRunEntry{
		OriginalKey:      key,
		TransformedKey:   key,
		OriginalValue:    value,
		TransformedValue: transformedValue,
		WillModify:       willModify,
		Skipped:          false,
	}

	if opts.DryRun {
		result.DryRunData = append(result.DryRunData, entry)
	} else {
		if willModify {
			if err := p.db.PutCF(cf, key, transformedValue); err != nil {
				result.Errors = append(result.Errors, TransformError{
					Key:           key,
					OriginalValue: value,
					Error:         fmt.Sprintf("write error: %v", err),
					Timestamp:     time.Now(),
				})
				return entry
			}
			result.Modified++
		}
	}

	return entry
}

// processMockDataWithCallback processes mock data with callback support
func (p *transformProcessor) processMockDataWithCallback(cf string, opts TransformOptions, result *TransformResult, callback ProgressCallback) (*TransformResult, error) {
	// Generate mock data
	mockDataCount := 10000
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

	total := limit

	// Process each entry
	for i := 0; i < limit; i++ {
		entry := mockData[i]

		// Apply filter if specified
		if opts.FilterExpression != "" {
			shouldProcess, err := p.applyFilter(entry.key, entry.value, opts)
			if err != nil {
				result.Errors = append(result.Errors, TransformError{
					Key:           entry.key,
					OriginalValue: entry.value,
					Error:         fmt.Sprintf("filter error: %v", err),
					Timestamp:     time.Now(),
				})

				dryRunEntry := DryRunEntry{
					OriginalKey:      entry.key,
					TransformedKey:   entry.key,
					OriginalValue:    entry.value,
					TransformedValue: entry.value,
					WillModify:       false,
					Skipped:          false,
				}

				result.Processed++
				if callback != nil {
					callback(result.Processed, total, dryRunEntry)
				}
				continue
			}

			if !shouldProcess {
				result.Skipped++
				result.Processed++

				dryRunEntry := DryRunEntry{
					OriginalKey:      entry.key,
					TransformedKey:   entry.key,
					OriginalValue:    entry.value,
					TransformedValue: entry.value,
					WillModify:       false,
					Skipped:          true,
				}

				if callback != nil {
					callback(result.Processed, total, dryRunEntry)
				}
				continue
			}
		}

		// Apply key transformation
		transformedKey, err := p.transformKey(entry.key, entry.value, opts)
		if err != nil {
			result.Errors = append(result.Errors, TransformError{
				Key:           entry.key,
				OriginalValue: entry.value,
				Error:         err.Error(),
				Timestamp:     time.Now(),
			})

			dryRunEntry := DryRunEntry{
				OriginalKey:      entry.key,
				TransformedKey:   entry.key,
				OriginalValue:    entry.value,
				TransformedValue: entry.value,
				WillModify:       false,
				Skipped:          false,
			}

			result.Processed++
			if callback != nil {
				callback(result.Processed, total, dryRunEntry)
			}
			continue
		}

		// Apply value transformation
		transformedValue, err := p.transformValue(entry.key, entry.value, opts)
		if err != nil {
			result.Errors = append(result.Errors, TransformError{
				Key:           entry.key,
				OriginalValue: entry.value,
				Error:         err.Error(),
				Timestamp:     time.Now(),
			})

			dryRunEntry := DryRunEntry{
				OriginalKey:      entry.key,
				TransformedKey:   transformedKey,
				OriginalValue:    entry.value,
				TransformedValue: entry.value,
				WillModify:       false,
				Skipped:          false,
			}

			result.Processed++
			if callback != nil {
				callback(result.Processed, total, dryRunEntry)
			}
			continue
		}

		// Check if key or value changed
		keyChanged := transformedKey != entry.key
		valueChanged := transformedValue != entry.value
		willModify := keyChanged || valueChanged

		dryRunEntry := DryRunEntry{
			OriginalKey:      entry.key,
			TransformedKey:   transformedKey,
			OriginalValue:    entry.value,
			TransformedValue: transformedValue,
			WillModify:       willModify,
			Skipped:          false,
		}

		if opts.DryRun {
			result.DryRunData = append(result.DryRunData, dryRunEntry)
		} else {
			if willModify {
				result.Modified++
			}
		}

		result.Processed++

		// Invoke callback after processing each entry
		if callback != nil {
			callback(result.Processed, total, dryRunEntry)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
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
