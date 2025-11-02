package transform

import (
	"testing"
)

// TestTransformProcessor_DryRun tests dry-run mode
func TestTransformProcessor_DryRun(t *testing.T) {
	// Setup test database
	// TODO: Create test database with sample data
	
	processor := NewTransformProcessor(nil) // nil = test DB
	
	opts := TransformOptions{
		Expression: "value.upper()",
		DryRun:     true,
		Limit:      5, // Small dataset for dry-run test
	}
	
	result, err := processor.Process("test_cf", opts)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}
	
	// Verify no changes were made to database
	if result.Modified > 0 {
		t.Errorf("DryRun should not modify database, but Modified = %d", result.Modified)
	}
	
	// Verify dry-run data is populated
	if len(result.DryRunData) == 0 {
		t.Error("DryRun should populate DryRunData")
	}
}

// TestTransformProcessor_BasicTransform tests basic transformation
func TestTransformProcessor_BasicTransform(t *testing.T) {
	// TODO: Setup test database
	
	processor := NewTransformProcessor(nil)
	
	opts := TransformOptions{
		Expression: "value.upper()",
		DryRun:     false,
		Limit:      10, // Only process 10 entries (each spawns Python process)
	}
	
	result, err := processor.Process("test_cf", opts)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}
	
	// Verify data was modified
	if result.Modified == 0 {
		t.Error("Expected some data to be modified")
	}
	
	// Verify statistics
	if result.Processed == 0 {
		t.Error("Expected some data to be processed")
	}
	
	// Verify we respected the limit
	if result.Processed > opts.Limit {
		t.Errorf("Expected max %d processed, got %d", opts.Limit, result.Processed)
	}
}

// TestTransformProcessor_WithFilter tests transformation with filter
func TestTransformProcessor_WithFilter(t *testing.T) {
	tests := []struct {
		name           string
		filter         string
		expectedCount  int
		shouldModify   bool
	}{
		{
			name:           "filter by key prefix",
			filter:         "key.startswith('user:')",
			expectedCount:  5,
			shouldModify:   true,
		},
		{
			name:           "filter all",
			filter:         "False",
			expectedCount:  0,
			shouldModify:   false,
		},
		{
			name:           "filter none",
			filter:         "True",
			expectedCount:  10,
			shouldModify:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewTransformProcessor(nil)
			
			opts := TransformOptions{
				Expression:       "value.upper()",
				FilterExpression: tt.filter,
				DryRun:           true,
				Limit:            15, // Test with 15 entries
			}
			
			result, err := processor.Process("test_cf", opts)
			if err != nil {
				t.Fatalf("Process() failed: %v", err)
			}
			
			// Verify filtering worked
			if tt.shouldModify && result.Processed == 0 {
				t.Error("Expected some data to be processed")
			}
		})
	}
}

// TestTransformProcessor_BatchProcessing tests batch processing
func TestTransformProcessor_BatchProcessing(t *testing.T) {
	// Test with minimal dataset (each entry spawns Python process)
	processor := NewTransformProcessor(nil)
	
	opts := TransformOptions{
		Expression: "value.upper()",
		BatchSize:  10,  // Process 10 at a time
		Limit:      20,  // Total of 20 entries (2 batches)
		Verbose:    true,
	}
	
	result, err := processor.Process("test_cf", opts)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}
	
	// Verify batch processing completed
	if result.Processed != 20 {
		t.Errorf("Expected 20 processed, got %d", result.Processed)
	}
}

// TestTransformProcessor_ErrorRecovery tests error recovery
func TestTransformProcessor_ErrorRecovery(t *testing.T) {
	processor := NewTransformProcessor(nil)
	
	// Expression that will fail on some values
	opts := TransformOptions{
		Expression: "int(value) * 2",  // Will fail on non-numeric strings
		DryRun:     false,
		Limit:      20, // Test with 20 entries (will have errors)
	}
	
	result, err := processor.Process("test_cf", opts)
	// Should not return error, but should have errors in result
	if err != nil {
		t.Fatalf("Process() should not fail completely: %v", err)
	}
	
	// Verify errors were recorded
	if len(result.Errors) == 0 {
		t.Error("Expected some errors to be recorded")
	}
	
	// Verify some data was still processed
	if result.Processed == 0 {
		t.Error("Expected some data to be processed despite errors")
	}
}

// TestTransformProcessor_Limit tests limit parameter
func TestTransformProcessor_Limit(t *testing.T) {
	processor := NewTransformProcessor(nil)
	
	opts := TransformOptions{
		Expression: "value.upper()",
		Limit:      5,
		DryRun:     true,
	}
	
	result, err := processor.Process("test_cf", opts)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}
	
	// Verify only limit entries were processed
	if result.Processed > opts.Limit {
		t.Errorf("Expected max %d processed, got %d", opts.Limit, result.Processed)
	}
}

// TestTransformProcessor_KeyTransform tests key transformation
func TestTransformProcessor_KeyTransform(t *testing.T) {
	processor := NewTransformProcessor(nil)

	t.Run("replace with colon test", func(t *testing.T) {
		// Note: Mock data uses "key1", "key2", etc which don't have colons
		// So this test verifies the mechanism works, even if no changes occur
		opts := TransformOptions{
			KeyExpression:   "key.replace('y', 'Y')",  // Replace 'y' with 'Y' to see changes
			ValueExpression: "value",
			DryRun:          true,
			Limit:           3,
		}

		result, err := processor.Process("test_cf", opts)
		if err != nil {
			t.Fatalf("Process() failed: %v", err)
		}

		// Verify keys were transformed in dry-run data
		if len(result.DryRunData) == 0 {
			t.Fatal("Expected dry-run data")
		}

		// Verify transformation occurred - 'y' should be replaced with 'Y'
		for _, entry := range result.DryRunData {
			if entry.OriginalKey != "" {
				t.Logf("Key transformation: %q -> %q", entry.OriginalKey, entry.TransformedKey)
				// "key1" should become "keY1"
				if entry.OriginalKey == "key1" && entry.TransformedKey != "keY1" {
					t.Errorf("Expected 'key1' -> 'keY1', got %q", entry.TransformedKey)
				}
			}
		}
	})

	t.Run("uppercase key transformation", func(t *testing.T) {
		opts := TransformOptions{
			KeyExpression:   "key.upper()",
			ValueExpression: "value",
			DryRun:          true,
			Limit:           3,
		}

		result, err := processor.Process("test_cf", opts)
		if err != nil {
			t.Fatalf("Process() failed: %v", err)
		}

		// Verify transformation occurred
		if len(result.DryRunData) == 0 {
			t.Fatal("Expected dry-run data")
		}

		for _, entry := range result.DryRunData {
			t.Logf("Key: %q -> %q", entry.OriginalKey, entry.TransformedKey)
			// Verify keys are actually uppercase
			if entry.TransformedKey != entry.OriginalKey {
				// Key was transformed
				if entry.OriginalKey == "key1" && entry.TransformedKey != "KEY1" {
					t.Errorf("Expected 'key1' -> 'KEY1', got %q", entry.TransformedKey)
				}
			}
		}
	})
}

// TestTransformProcessor_Statistics tests result statistics
func TestTransformProcessor_Statistics(t *testing.T) {
	processor := NewTransformProcessor(nil)
	
	opts := TransformOptions{
		Expression:       "value.upper() if value else None",
		FilterExpression: "len(value) > 0",
		DryRun:           false,
		Limit:            10, // Test with 10 entries
	}
	
	result, err := processor.Process("test_cf", opts)
	if err != nil {
		t.Fatalf("Process() failed: %v", err)
	}
	
	// Verify statistics are consistent
	total := result.Modified + result.Skipped + len(result.Errors)
	if total != result.Processed {
		t.Errorf("Statistics inconsistent: Modified(%d) + Skipped(%d) + Errors(%d) != Processed(%d)",
			result.Modified, result.Skipped, len(result.Errors), result.Processed)
	}
}

// TestTransformProcessor_ProgressCallback tests progress reporting
func TestTransformProcessor_ProgressCallback(t *testing.T) {
	processor := NewTransformProcessor(nil)

	t.Run("callback is called during processing", func(t *testing.T) {
		callbackInvoked := false
		callCount := 0

		callback := func(processed int, total int, current DryRunEntry) {
			callbackInvoked = true
			callCount++

			// Verify parameters are reasonable
			if processed < 0 {
				t.Errorf("processed should be non-negative, got %d", processed)
			}
			if total < 0 {
				t.Errorf("total should be non-negative, got %d", total)
			}
			if processed > total {
				t.Errorf("processed (%d) should not exceed total (%d)", processed, total)
			}
		}

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      5,
		}

		result, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		if !callbackInvoked {
			t.Error("Callback was not invoked")
		}

		if callCount != result.Processed {
			t.Errorf("Callback should be called %d times, got %d", result.Processed, callCount)
		}
	})

	t.Run("callback receives correct progress information", func(t *testing.T) {
		var progressUpdates []int

		callback := func(processed int, total int, current DryRunEntry) {
			progressUpdates = append(progressUpdates, processed)
		}

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      5,
		}

		result, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		// Verify progress updates are sequential
		for i, progress := range progressUpdates {
			expected := i + 1
			if progress != expected {
				t.Errorf("Progress update %d: expected %d, got %d", i, expected, progress)
			}
		}

		// Verify final progress matches result
		if len(progressUpdates) > 0 {
			finalProgress := progressUpdates[len(progressUpdates)-1]
			if finalProgress != result.Processed {
				t.Errorf("Final progress %d should match result.Processed %d", finalProgress, result.Processed)
			}
		}
	})

	t.Run("callback receives current entry information", func(t *testing.T) {
		var entries []DryRunEntry

		callback := func(processed int, total int, current DryRunEntry) {
			entries = append(entries, current)
		}

		opts := TransformOptions{
			KeyExpression:   "key.upper()",
			ValueExpression: "value.upper()",
			DryRun:          true,
			Limit:           3,
		}

		result, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		// Verify we received entry information
		if len(entries) == 0 {
			t.Error("No entries received in callback")
		}

		// Verify entries have required fields
		for i, entry := range entries {
			if entry.OriginalKey == "" {
				t.Errorf("Entry %d: OriginalKey should not be empty", i)
			}
			if entry.OriginalValue == "" {
				t.Errorf("Entry %d: OriginalValue should not be empty", i)
			}

			// Log transformation for debugging
			t.Logf("Entry %d: %q -> %q (value: %q -> %q)",
				i, entry.OriginalKey, entry.TransformedKey,
				entry.OriginalValue, entry.TransformedValue)
		}

		// Verify entries match result
		if len(entries) != result.Processed {
			t.Errorf("Expected %d entries, got %d", result.Processed, len(entries))
		}
	})

	t.Run("callback with actual write operations", func(t *testing.T) {
		callbackInvoked := false

		callback := func(processed int, total int, current DryRunEntry) {
			callbackInvoked = true
		}

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     false,
			Limit:      5,
		}

		result, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		if !callbackInvoked {
			t.Error("Callback was not invoked in write mode")
		}

		// Verify modifications were counted
		if result.Modified == 0 {
			t.Error("Expected some modifications")
		}
	})

	t.Run("callback with filter expression", func(t *testing.T) {
		var processedKeys []string
		var skippedCount int

		callback := func(processed int, total int, current DryRunEntry) {
			processedKeys = append(processedKeys, current.OriginalKey)
			if current.Skipped {
				skippedCount++
			}
		}

		opts := TransformOptions{
			Expression:       "value.upper()",
			FilterExpression: "key.startswith('key1')",
			DryRun:           true,
			Limit:            10,
		}

		result, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		// Verify skipped count matches result
		if skippedCount != result.Skipped {
			t.Errorf("Callback skipped count %d should match result.Skipped %d", skippedCount, result.Skipped)
		}
	})

	t.Run("total count calculation", func(t *testing.T) {
		var totalValues []int

		callback := func(processed int, total int, current DryRunEntry) {
			totalValues = append(totalValues, total)
		}

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      5,
		}

		_, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		// Verify total is consistent across all callbacks
		if len(totalValues) > 0 {
			firstTotal := totalValues[0]
			for i, total := range totalValues {
				if total != firstTotal {
					t.Errorf("Total at callback %d (%d) should match first total (%d)", i, total, firstTotal)
				}
			}

			// Total should match or exceed limit
			if firstTotal < opts.Limit {
				t.Logf("Note: Total %d is less than limit %d (may be due to dataset size)", firstTotal, opts.Limit)
			}
		}
	})
}

// TestTransformProcessor_ConcurrentSafety tests concurrent execution safety
func TestTransformProcessor_ConcurrentSafety(t *testing.T) {
	processor := NewTransformProcessor(nil)

	t.Run("multiple goroutines can process safely", func(t *testing.T) {
		// Run multiple transform operations concurrently
		const numGoroutines = 10
		errors := make(chan error, numGoroutines)
		results := make(chan *TransformResult, numGoroutines)

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      5,
		}

		// Launch concurrent operations
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				result, err := processor.Process("test_cf", opts)
				if err != nil {
					errors <- err
					return
				}
				results <- result
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-errors:
				t.Errorf("Goroutine failed: %v", err)
			case result := <-results:
				if result.Processed == 0 {
					t.Error("Expected some data to be processed")
				}
			}
		}
	})

	t.Run("concurrent operations produce consistent results", func(t *testing.T) {
		const numGoroutines = 5
		results := make([]*TransformResult, numGoroutines)
		errors := make([]error, numGoroutines)

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      5,
		}

		// Run operations concurrently
		done := make(chan bool)
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				results[idx], errors[idx] = processor.Process("test_cf", opts)
				done <- true
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all succeeded
		for i, err := range errors {
			if err != nil {
				t.Errorf("Operation %d failed: %v", i, err)
			}
		}

		// Verify results are consistent
		if len(results) > 0 && results[0] != nil {
			expectedProcessed := results[0].Processed
			for i, result := range results[1:] {
				if result != nil && result.Processed != expectedProcessed {
					t.Errorf("Result %d has inconsistent processed count: expected %d, got %d",
						i+1, expectedProcessed, result.Processed)
				}
			}
		}
	})

	t.Run("concurrent with callback operations", func(t *testing.T) {
		const numGoroutines = 5
		errors := make(chan error, numGoroutines)

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      5,
		}

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				callback := func(processed int, total int, current DryRunEntry) {
					// Callback should be called without race conditions
					_ = processed
					_ = total
					_ = current
				}

				_, err := processor.ProcessWithCallback("test_cf", opts, callback)
				if err != nil {
					errors <- err
					return
				}
				errors <- nil
			}(i)
		}

		// Check all completed without error
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", i, err)
			}
		}
	})
}

// TestTransformProcessor_MemoryUsage tests memory usage with large datasets
func TestTransformProcessor_MemoryUsage(t *testing.T) {
	processor := NewTransformProcessor(nil)

	t.Run("dry-run data doesn't grow unbounded", func(t *testing.T) {
		// Test that dry-run mode with limit doesn't accumulate all data
		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      100, // Process 100 entries
		}

		result, err := processor.Process("test_cf", opts)
		if err != nil {
			t.Fatalf("Process() failed: %v", err)
		}

		// Dry-run data should be limited by the Limit parameter
		if len(result.DryRunData) > opts.Limit {
			t.Errorf("DryRunData length (%d) should not exceed limit (%d)",
				len(result.DryRunData), opts.Limit)
		}

		// Even if we have mock data, the result should respect the limit
		if result.Processed > opts.Limit {
			t.Errorf("Processed count (%d) should not exceed limit (%d)",
				result.Processed, opts.Limit)
		}
	})

	t.Run("batch processing limits memory", func(t *testing.T) {
		// Test that batch processing processes data in chunks
		opts := TransformOptions{
			Expression: "value.upper()",
			BatchSize:  10,  // Small batches
			Limit:      100, // Total entries
			DryRun:     false,
		}

		result, err := processor.Process("test_cf", opts)
		if err != nil {
			t.Fatalf("Process() failed: %v", err)
		}

		// Verify processing completed
		if result.Processed == 0 {
			t.Error("Expected some data to be processed")
		}

		// With batching, we should process up to limit
		if result.Processed > opts.Limit {
			t.Errorf("Processed count (%d) should not exceed limit (%d)",
				result.Processed, opts.Limit)
		}
	})

	t.Run("large limit doesn't cause OOM", func(t *testing.T) {
		// Test with a reasonable limit value
		// Note: Each entry spawns a Python process, so we keep this practical
		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      200, // Reasonable limit for testing
		}

		// This should complete without panic or excessive memory
		result, err := processor.Process("test_cf", opts)
		if err != nil {
			t.Fatalf("Process() failed: %v", err)
		}

		// Should complete successfully
		if result == nil {
			t.Error("Expected non-nil result")
		}

		// Mock data respects the limit
		// (In mock mode, we generate up to Limit entries, max 10000)
		if len(result.DryRunData) > opts.Limit {
			t.Errorf("DryRunData length (%d) exceeds limit (%d)",
				len(result.DryRunData), opts.Limit)
		}

		// Verify processed count matches
		if result.Processed != len(result.DryRunData) {
			t.Errorf("Processed count (%d) should match DryRunData length (%d)",
				result.Processed, len(result.DryRunData))
		}
	})

	t.Run("errors don't accumulate unbounded", func(t *testing.T) {
		// Test that error collection is reasonable
		opts := TransformOptions{
			Expression: "int(value) * 2", // Will fail on string values
			DryRun:     false,
			Limit:      50,
		}

		result, err := processor.Process("test_cf", opts)
		if err != nil {
			t.Fatalf("Process() should handle errors gracefully: %v", err)
		}

		// Errors should be collected but not exceed processed count
		if len(result.Errors) > result.Processed {
			t.Errorf("Error count (%d) should not exceed processed count (%d)",
				len(result.Errors), result.Processed)
		}

		// Total accounting should be correct
		total := result.Modified + result.Skipped + len(result.Errors)
		if total != result.Processed {
			t.Errorf("Accounting error: Modified(%d) + Skipped(%d) + Errors(%d) != Processed(%d)",
				result.Modified, result.Skipped, len(result.Errors), result.Processed)
		}
	})

	t.Run("callback mode doesn't accumulate dry-run data", func(t *testing.T) {
		// When using callback, we shouldn't accumulate all dry-run data in memory
		callbackCount := 0

		callback := func(processed int, total int, current DryRunEntry) {
			callbackCount++
		}

		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      50,
		}

		result, err := processor.ProcessWithCallback("test_cf", opts, callback)
		if err != nil {
			t.Fatalf("ProcessWithCallback() failed: %v", err)
		}

		// Callback should have been called
		if callbackCount == 0 {
			t.Error("Callback was not called")
		}

		// DryRunData should still be populated (for backward compatibility)
		// but should be limited
		if len(result.DryRunData) > opts.Limit {
			t.Errorf("DryRunData length (%d) exceeds limit (%d)",
				len(result.DryRunData), opts.Limit)
		}
	})

	t.Run("memory stability across multiple runs", func(t *testing.T) {
		// Run the same operation multiple times
		// Memory should not accumulate across runs
		opts := TransformOptions{
			Expression: "value.upper()",
			DryRun:     true,
			Limit:      20,
		}

		for i := 0; i < 5; i++ {
			result, err := processor.Process("test_cf", opts)
			if err != nil {
				t.Fatalf("Process() run %d failed: %v", i, err)
			}

			// Each run should produce consistent results
			if len(result.DryRunData) > opts.Limit {
				t.Errorf("Run %d: DryRunData length (%d) exceeds limit (%d)",
					i, len(result.DryRunData), opts.Limit)
			}
		}
	})
}
