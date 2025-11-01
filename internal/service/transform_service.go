package service

import (
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/transform"
)

// TransformService provides data transformation operations
type TransformService struct {
	processor transform.TransformProcessor
}

// TransformOptions contains options for transform operations
type TransformOptions struct {
	CF               string `json:"cf"`                // Column family to transform
	Expression       string `json:"expression"`        // Python expression for transformation
	KeyExpression    string `json:"key_expression"`    // Python expression for key transformation
	ValueExpression  string `json:"value_expression"`  // Python expression for value transformation
	FilterExpression string `json:"filter_expression"` // Python expression for filtering
	ScriptPath       string `json:"script_path"`       // Path to Python script file
	DryRun           bool   `json:"dry_run"`           // Preview mode - don't apply changes
	Limit            int    `json:"limit"`             // Maximum number of entries to process
	BatchSize        int    `json:"batch_size"`        // Batch size for processing
	Verbose          bool   `json:"verbose"`           // Verbose output
}

// TransformResult contains the results of a transform operation
type TransformResult struct {
	Processed  int                  `json:"processed"`  // Total number of entries processed
	Modified   int                  `json:"modified"`   // Number of entries modified
	Skipped    int                  `json:"skipped"`    // Number of entries skipped
	Errors     []TransformError     `json:"errors"`     // List of errors encountered
	Preview    []TransformPreview   `json:"preview"`    // Preview data for dry-run mode
	Duration   string               `json:"duration"`   // Total processing time
}

// TransformError represents an error during transformation
type TransformError struct {
	Key           string `json:"key"`
	OriginalValue string `json:"original_value"`
	Error         string `json:"error"`
	Timestamp     string `json:"timestamp"`
}

// TransformPreview represents a preview entry in dry-run mode
type TransformPreview struct {
	OriginalKey      string `json:"original_key"`
	TransformedKey   string `json:"transformed_key"`
	OriginalValue    string `json:"original_value"`
	TransformedValue string `json:"transformed_value"`
	WillModify       bool   `json:"will_modify"`
	Skipped          bool   `json:"skipped"`
}

// NewTransformService creates a new TransformService instance
func NewTransformService(database db.KeyValueDB) *TransformService {
	return &TransformService{
		processor: transform.NewTransformProcessor(database),
	}
}

// Transform executes a data transformation operation
func (s *TransformService) Transform(opts TransformOptions) (*TransformResult, error) {
	// Convert service options to transform options
	transformOpts := transform.TransformOptions{
		Expression:       opts.Expression,
		KeyExpression:    opts.KeyExpression,
		ValueExpression:  opts.ValueExpression,
		FilterExpression: opts.FilterExpression,
		ScriptPath:       opts.ScriptPath,
		DryRun:           opts.DryRun,
		Limit:            opts.Limit,
		BatchSize:        opts.BatchSize,
		Verbose:          opts.Verbose,
	}

	// Execute transformation
	result, err := s.processor.Process(opts.CF, transformOpts)
	if err != nil {
		return nil, err
	}

	// Convert transform errors
	errors := make([]TransformError, 0, len(result.Errors))
	for _, e := range result.Errors {
		errors = append(errors, TransformError{
			Key:           e.Key,
			OriginalValue: e.OriginalValue,
			Error:         e.Error,
			Timestamp:     e.Timestamp.Format("2006-01-02 15:04:05"),
		})
	}

	// Convert dry-run data
	preview := make([]TransformPreview, 0, len(result.DryRunData))
	for _, d := range result.DryRunData {
		preview = append(preview, TransformPreview{
			OriginalKey:      d.OriginalKey,
			TransformedKey:   d.TransformedKey,
			OriginalValue:    d.OriginalValue,
			TransformedValue: d.TransformedValue,
			WillModify:       d.WillModify,
			Skipped:          d.Skipped,
		})
	}

	return &TransformResult{
		Processed: result.Processed,
		Modified:  result.Modified,
		Skipped:   result.Skipped,
		Errors:    errors,
		Preview:   preview,
		Duration:  result.Duration.String(),
	}, nil
}
